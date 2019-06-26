package hosts

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/provisioning-agent/host"
	"github.com/tidwall/gjson"

	"github.com/choria-io/go-choria/choria"
)

func connect(ctx context.Context) (choria.Connector, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Existing on shut down")
	}

	return fw.NewConnector(ctx, fw.MiddlewareServers, fw.Certname(), log)
}

func listen(ctx context.Context, wg *sync.WaitGroup, conn choria.Connector) {
	defer wg.Done()

	events := make(chan *choria.ConnectorMessage, 1000)

	rid, err := fw.NewRequestID()
	if err != nil {
		log.Errorf("Could not create provisioning data listener unique id: %s", err)
		return
	}

	err = conn.QueueSubscribe(ctx, rid, "choria.provisioning_data", "", events)
	if err != nil {
		log.Errorf("Could not listen for provisioning data events: %s", err)
		return
	}

	rid, err = fw.NewRequestID()
	if err != nil {
		log.Errorf("Could not create lifecycle event listener unique id: %s", err)
		return
	}

	err = conn.QueueSubscribe(ctx, rid, "choria.lifecycle.event.startup.provision_mode_server", "", events)
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

	proto := gjson.GetBytes(msg.Data, "protocol")
	if !proto.Exists() {
		return "", fmt.Errorf("received a message with no protocol description")
	}

	if strings.HasPrefix(proto.String(), "choria:lifecycle:startup") {
		return handleStartupEvent(msg)
	}

	return handleRegistration(msg)
}

func handleStartupEvent(msg *choria.ConnectorMessage) (string, error) {
	component := gjson.GetBytes(msg.Data, "component").String()
	if component != "provision_mode_server" {
		return "", nil
	}

	return gjson.GetBytes(msg.Data, "identity").String(), nil
}

func handleRegistration(msg *choria.ConnectorMessage) (string, error) {
	t, err := fw.NewTransportFromJSON(string(msg.Data))
	if err != nil {
		return "", fmt.Errorf("could not create transport: %s", err)
	}

	return t.SenderID(), nil
}
