package hosts

import (
	"context"
	"sync"
	"time"

	"github.com/choria-io/provisioning-agent/host"
)

func provisioner(ctx context.Context, wg *sync.WaitGroup, i int) {
	defer wg.Done()

	log.Debugf("Provisioner worker %d starting", i)

	for {
		select {
		case host := <-work:
			log.Infof("Provisioning %s", host.Identity)

			err := provisionTarget(ctx, host)
			if err != nil {
				provErrCtr.WithLabelValues(conf.Site).Inc()
				log.Errorf("Could not provision %s: %s", host.Identity, err)
			}

			// delay removing the node to avoid a race between discovery and node restarting splay
			go func() {
				<-time.NewTimer(20 * time.Second).C
				done <- host
			}()

		case <-ctx.Done():
			log.Infof("Worker %d exiting on context", i)
			return
		}
	}
}

func provisionTarget(ctx context.Context, target *host.Host) error {
	busyWorkerGauge.WithLabelValues(conf.Site).Inc()
	defer busyWorkerGauge.WithLabelValues(conf.Site).Dec()

	err := target.Provision(ctx, fw)
	if err != nil {
		return err
	}

	provisionedCtr.WithLabelValues(conf.Site).Inc()

	return nil
}

func finisher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case host := <-done:
			log.Infof("Removing %s from the provision list", host.Identity)
			remove(host)
		case <-ctx.Done():
			log.Info("Finisher exiting on context")
			return
		}
	}
}
