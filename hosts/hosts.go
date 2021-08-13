package hosts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/lifecycle"
	addl "github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/client"
	rpc "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
	"github.com/choria-io/go-choria/providers/discovery/broadcast"
	"github.com/choria-io/provisioner/config"
	"github.com/choria-io/provisioner/host"
	"github.com/sirupsen/logrus"
)

var (
	hosts = make(map[string]*host.Host)
	work  = make(chan *host.Host, 1000)
	done  = make(chan *host.Host, 1000)
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

	ddl, err := addl.CachedDDL("choria_provision")
	if err != nil {
		return fmt.Errorf("could not find DDL for agent choria_provision in the agent cache")
	}

	agent, err := rpc.New(fw, "choria_provision", rpc.DDL(ddl))
	if err != nil {
		return fmt.Errorf("could not create RPC client: %s", err)
	}

	conn, err := connect(ctx)
	if err != nil {
		return fmt.Errorf("could not create initial events connection: %s", err)
	}

	err = publishStartupEvent(conn)
	if err != nil {
		log.Errorf("Could not publish startup event: %s", err)
	}

	discoverTrigger := make(chan struct{}, 1)

	if conf.LeaderElectionName != "" {
		conf.Pause()

		wg.Add(1)
		go startElection(ctx, wg, conn, conf, discoverTrigger, log)
	}

	wg.Add(1)
	go listen(ctx, wg, cfg.LifecycleComponent, conn)

	wg.Add(1)
	go finisher(ctx, wg)

	switch {
	case conf.Management != nil && conf.LeaderElectionName != "":
		log.Warnf("Backplane management mode is not compatible with Leader elections")

	case conf.Management != nil:
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

		case <-discoverTrigger:
			discover(ctx, agent)

		case <-ctx.Done():
			log.Infof("Existing on context interrupt")
			return nil
		}
	}
}

func publishStartupEvent(conn choria.Connector) error {
	event, err := lifecycle.New(lifecycle.Startup, lifecycle.Component("provisioner"), lifecycle.Identity(fw.Config.Identity), lifecycle.Version(config.Version))
	if err != nil {
		return fmt.Errorf("could not create event: %s", err)
	}

	err = lifecycle.PublishEvent(event, conn)
	if err != nil {
		return fmt.Errorf("could not publish: %s", err)
	}

	return nil
}

func remove(host *host.Host) {
	mu.Lock()
	defer mu.Unlock()

	delete(hosts, host.Identity)
}

func add(host *host.Host) bool {
	mu.Lock()
	defer mu.Unlock()

	if len(work) == cap(work) {
		log.Warnf("Work queue is full at %d entries, cannot add %s", len(work), host.Identity)
		return false
	}

	_, known := hosts[host.Identity]
	if known {
		return false
	}

	log.Debugf("Adding %s to the work queue with %d entries", host.Identity, len(hosts))
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

	bd := broadcast.New(fw)
	nodes, err := bd.Discover(ctx, broadcast.Collective("provisioning"), broadcast.Filter(f), broadcast.Timeout(1*time.Second))
	if err != nil {
		return err
	}

	for _, n := range nodes {
		if add(host.NewHost(n, conf)) {
			log.Infof("Adding %s to the provision list after discovering it", n)
			discoveredCtr.WithLabelValues(conf.Site).Inc()
		}
	}

	return nil
}
