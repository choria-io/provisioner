// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package hosts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/lifecycle"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/client/client"
	"github.com/choria-io/go-choria/providers/discovery/broadcast"
	"github.com/choria-io/provisioner/config"
	"github.com/choria-io/provisioner/host"
	"github.com/sirupsen/logrus"
)

var (
	hosts = make(map[string]*host.Host)
	work  = make(chan *host.Host, 50000)
	done  = make(chan *host.Host, 50000)
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

	conn, err := connect(ctx)
	if err != nil {
		return fmt.Errorf("could not create initial events connection: %s", err)
	}

	err = publishStartupEvent(conn)
	if err != nil {
		log.Errorf("Could not publish startup event: %s", err)
	}

	discoverTrigger := make(chan struct{}, 1)

	if conf.LeaderElection {
		conf.Pause()

		wg.Add(1)
		go startElection(ctx, wg, conn, fw, discoverTrigger, log)
	}

	wg.Add(1)
	go listen(ctx, wg, cfg.LifecycleComponent, conn)

	wg.Add(1)
	go finisher(ctx, wg)

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go provisioner(ctx, wg, i+1)
	}

	timer := time.NewTicker(cfg.IntervalDuration)

	discoveredCtr.WithLabelValues(conf.Site).Add(0.0)
	provisionedCtr.WithLabelValues(conf.Site).Add(0.0)
	waitingGauge.WithLabelValues(conf.Site).Set(0.0)
	unprovisionedGauge.WithLabelValues(conf.Site).Set(0.0)

	discover(ctx)

	for {
		select {
		case <-timer.C:
			discover(ctx)

		case <-discoverTrigger:
			discover(ctx)

		case <-ctx.Done():
			log.Infof("Existing on context interrupt")
			return nil
		}
	}
}

func publishStartupEvent(conn inter.Connector) error {
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

	removeUnlocked(host)
}

func removeUnlocked(host *host.Host) {
	delete(hosts, host.Identity)
	unprovisionedGauge.WithLabelValues(conf.Site).Set(float64(len(hosts)))

	waitingGauge.WithLabelValues(conf.Site).Set(float64(len(work)))
}

func removeAllHosts() {
	mu.Lock()
	defer mu.Unlock()

	for _, h := range hosts {
		delete(hosts, h.Identity)
	}

	for {
		select {
		case <-work:
		default:
			unprovisionedGauge.WithLabelValues(conf.Site).Set(float64(len(hosts)))
			waitingGauge.WithLabelValues(conf.Site).Set(float64(len(work)))

			return
		}
	}
}

func isCurrent(h *host.Host) bool {
	mu.Lock()
	defer mu.Unlock()

	_, ok := hosts[h.Identity]
	return ok
}

func add(host *host.Host) bool {
	mu.Lock()
	defer mu.Unlock()

	_, known := hosts[host.Identity]
	if known {
		// if it was recently added don't add it again else we remove it
		// and add it again, might cause a dupe provision but better than
		// nodes dying on us, a dupe provision will time out on first rpc
		// failure and provision will also not provision nodes added too
		// long ago
		if time.Since(host.DiscoveredTime()) < conf.IntervalDuration {
			return false
		}
		removeUnlocked(host)
	}

	log.Infof("Adding %s to the work queue with %d entries", host.Identity, len(hosts))
	hosts[host.Identity] = host

	select {
	case work <- host:
	default:
		log.Warnf("Adding host to work queue failed with %d / %d entries", len(work), cap(work))
		removeUnlocked(host)
	}

	unprovisionedGauge.WithLabelValues(conf.Site).Set(float64(len(hosts)))
	waitingGauge.WithLabelValues(conf.Site).Set(float64(len(work)))

	return true
}

func discover(ctx context.Context) {
	if conf.Paused() {
		log.Warnf("Skipping discovery while paused")
		return
	}

	discoverCycleCtr.WithLabelValues(conf.Site).Inc()

	err := discoverProvisionableNodes(ctx)
	if err != nil {
		errCtr.WithLabelValues(conf.Site).Inc()
		log.Errorf("Could not discover nodes: %s", err)
	}
}

func discoverProvisionableNodes(ctx context.Context) error {
	log.Infof("Looking for provisionable hosts")

	f, err := client.NewFilter(client.AgentFilter("choria_provision"))
	if err != nil {
		return err
	}

	bd := broadcast.New(fw)
	nodes, err := bd.Discover(ctx, broadcast.Collective("provisioning"), broadcast.Filter(f), broadcast.SlidingWindow(), broadcast.Timeout(2*time.Second))
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
