package hosts

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/lifecycle"

	"github.com/choria-io/provisioner/host"

	"github.com/choria-io/go-choria/choria"
)

func connect(ctx context.Context) (choria.Connector, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Existing on shut down")
	}

	return fw.NewConnector(ctx, fw.MiddlewareServers, fw.Certname(), log)
}

func listen(ctx context.Context, wg *sync.WaitGroup, component string, conn choria.Connector) {
	defer wg.Done()

	events := make(chan *choria.ConnectorMessage, 1000)

	rid, err := fw.NewRequestID()
	if err != nil {
		log.Errorf("Could not create provisioning data listener unique id: %s", err)
		return
	}

	err = conn.QueueSubscribe(ctx, rid, fmt.Sprintf("choria.lifecycle.event.startup.%s", component), "", events)
	if err != nil {
		log.Errorf("Could not listen for lifecycle events: %s", err)
		return
	}

	for {
		select {
		case e := <-events:
			node, err := handle(e)
			if err != nil {
				log.Errorf("could not handle message: %s", err)
			}

			if node != "" && add(host.NewHost(node, conf)) {
				log.Infof("Adding %s to the provision list after receiving an event", node)
				eventsCtr.WithLabelValues(conf.Site).Inc()
			}

		case <-ctx.Done():
			return
		}
	}
}

func handle(msg *choria.ConnectorMessage) (string, error) {
	if conf.Paused() {
		log.Warnf("Skipping event processing while paused")
		return "", nil
	}

	event, err := lifecycle.NewFromJSON(msg.Bytes())
	if err != nil {
		return "", err
	}

	return handleEvent(event)
}

func handleEvent(event lifecycle.Event) (string, error) {
	if event.Type() != lifecycle.Startup {
		return "", nil
	}

	return event.Identity(), nil
}
