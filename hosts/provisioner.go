// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package hosts

import (
	"context"
	"sync"
	"time"

	"github.com/choria-io/provisioner/host"
)

func provisioner(ctx context.Context, wg *sync.WaitGroup, i int) {
	defer wg.Done()

	log.Debugf("Provisioner worker %d starting", i)

	for {
		select {
		case host := <-work:
			log.Infof("Provisioning %s", host.Identity)

			// the work queue might have an old entry that is not in the host lists anymore
			// so we short circuit here to avoid provisioning machines that might be stale.
			//
			// hosts are removed when elections pause processing etc
			//
			// we dont pass to done, its already not on the hosts list.
			if !isCurrent(host) {
				continue
			}

			delay, err := provisionTarget(ctx, host)
			if err != nil {
				provErrCtr.WithLabelValues(conf.Site).Inc()
				log.Errorf("Could not provision %s: %s", host.Identity, err)
				done <- host
				continue
			}

			log.Infof("Provisioned %s", host.Identity)
			if delay {
				time.AfterFunc(60*time.Second, func() { done <- host })
			} else {
				done <- host
			}

		case <-ctx.Done():
			log.Infof("Worker %d exiting on context", i)
			return
		}
	}
}

func provisionTarget(ctx context.Context, target *host.Host) (bool, error) {
	busyWorkerGauge.WithLabelValues(conf.Site).Inc()
	defer busyWorkerGauge.WithLabelValues(conf.Site).Dec()

	delay, err := target.Provision(ctx, fw)
	if err != nil {
		return false, err
	}

	provisionedCtr.WithLabelValues(conf.Site).Inc()

	return delay, nil
}

func finisher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case host := <-done:
			log.Debugf("Removing %s from the provision list", host.Identity)
			remove(host)
		case <-ctx.Done():
			log.Info("Finisher exiting on context")
			return
		}
	}
}
