package hosts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-client/client"
	rpc "github.com/choria-io/mcorpc-agent-provider/mcorpc/client"
	provision "github.com/choria-io/provisioning-agent/agent"
	"github.com/choria-io/provisioning-agent/config"
	"github.com/choria-io/provisioning-agent/host"
	"github.com/sirupsen/logrus"
)

var (
	hosts = make(map[string]*host.Host)
	work  = make(chan *host.Host, 1000)
	done  = make(chan *host.Host, 100)
	mu    = &sync.Mutex{}
	log   *logrus.Entry
	fw    *choria.Framework
	conf  *config.Config
	wg    = &sync.WaitGroup{}
)

// Process starts the provisioning process
func Process(ctx context.Context, cfg *config.Config, cfw *choria.Framework) error {
	fw = cfw
	conf = cfg
	log = fw.Logger("hosts")

	log.Infof("Choria Provisioner starting using configuration file %s. Discovery interval %s using %d workers", conf.File, conf.Interval, conf.Workers)

	ddl, ok := provision.DDL["choria_provision"]
	if !ok {
		return fmt.Errorf("could not find DDL for agent choria_provision in the provision.DDL structure")
	}

	agent, err := rpc.New(fw, "choria_provision", rpc.DDL(ddl))
	if err != nil {
		return fmt.Errorf("could not create RPC client: %s", err)
	}

	conn, err := connect(ctx)
	if err != nil {
		return fmt.Errorf("could not create initial events connection: %s", err)
	}

	wg.Add(1)
	go listen(ctx, wg, conn)

	wg.Add(1)
	go finisher(ctx, wg)

	if conf.Management != nil {
		wg.Add(1)
		go startBackplane(ctx, wg)
	}

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go provisioner(ctx, wg, i+1)
	}

	timer := time.NewTicker(cfg.IntervalDuration)

	discoveredCtr.WithLabelValues(conf.Site).Add(0.0)
	provisionedCtr.WithLabelValues(conf.Site).Add(0.0)

	discover(ctx, agent)

	for {
		select {
		case <-timer.C:
			discover(ctx, agent)

		case <-ctx.Done():
			log.Infof("Existing on context interrupt")
			return nil
		}
	}
}

func remove(host *host.Host) {
	mu.Lock()
	defer mu.Unlock()

	delete(hosts, host.Identity)
}

func add(host *host.Host) bool {
	mu.Lock()
	defer mu.Unlock()

	_, known := hosts[host.Identity]
	if known {
		return false
	}

	log.Debugf("Adding %s to the work queue", host)
	hosts[host.Identity] = host
	work <- host

	return true
}

func discover(ctx context.Context, agent *rpc.RPC) {
	if conf.Paused() {
		log.Warnf("Skipping discovery while paused")
		return
	}

	discoverCycleCtr.WithLabelValues(conf.Site).Inc()

	err := discoverProvisionableNodes(ctx, agent)
	if err != nil {
		errCtr.WithLabelValues(conf.Site).Inc()
		log.Errorf("Could not discover nodes: %s", err)
	}
}

func discoverProvisionableNodes(ctx context.Context, agent *rpc.RPC) error {
	log.Infof("Looking for provisionable hosts")

	f, err := client.NewFilter(client.AgentFilter("choria_provision"))
	if err != nil {
		return err
	}

	nodes, err := agent.Discover(ctx, f)
	if err != nil {
		return err
	}

	for _, n := range nodes {
		if add(host.NewHost(n, conf)) {
			discoveredCtr.WithLabelValues(conf.Site).Inc()
		}
	}

	return nil
}
