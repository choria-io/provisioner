package hosts

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/provisioning-agent/host"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
)

func connect(ctx context.Context) (choria.Connector, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Existing on shut down")
	}

	return fw.NewConnector(ctx, brokerUrls, fw.Certname(), log)
}

func brokerUrls() ([]srvcache.Server, error) {
	return fw.MiddlewareServers()
}

func listen(ctx context.Context, wg *sync.WaitGroup, conn choria.Connector) {
	defer wg.Done()

	events := make(chan *choria.ConnectorMessage, 1000)

	err := conn.QueueSubscribe(ctx, "events", "choria.provisioning_data", "", events)
	if err != nil {
		log.Errorf("Could not listen for events: %s", err)
		return
	}

	for {
		select {
		case e := <-events:
			if conf.Paused() {
				log.Warnf("Skipping event processing while paused")
				continue
			}

			t, err := fw.NewTransportFromJSON(string(e.Data))
			if err != nil {
				log.Errorf("Could not create transport for received event: %s", err)
				continue
			}

			log.Infof("Adding %s to the provision list after receiving an event", t.SenderID())

			if add(host.NewHost(t.SenderID(), conf)) {
				eventsCtr.WithLabelValues(conf.Site).Inc()
			}
		case <-ctx.Done():
			return
		}
	}
}
