package hosts

import (
	"context"
	"fmt"
	"time"

	"github.com/choria-io/provisioning-agent/provisioner/host"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/srvcache"
)

func startEventListener(ctx context.Context) {
	for {
		if ctx.Err() == nil {
			conn, err := connect(ctx)
			if err != nil {
				log.Errorf("Initial connection for event stream failed: %s", err)
				time.Sleep(time.Second)
				continue
			}

			listen(ctx, conn)
		}

		return
	}
}
func connect(ctx context.Context) (choria.Connector, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("Existing on shut down")
	}

	return fw.NewConnector(ctx, brokerUrls, fw.Certname(), log)
}

func brokerUrls() ([]srvcache.Server, error) {
	return fw.MiddlewareServers()
}

func listen(ctx context.Context, conn choria.Connector) {
	events := make(chan *choria.ConnectorMessage, 1000)

	err := conn.QueueSubscribe(ctx, "events", "choria.provisioning_data", "", events)
	if err != nil {
		log.Errorf("Could not listen for events: %s", err)
		return
	}

	for {
		select {
		case e := <-events:
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
