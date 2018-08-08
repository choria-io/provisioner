package hosts

import (
	"context"
	"sync"

	"github.com/choria-io/provisioning-agent/host"
)

func provisioner(ctx context.Context, wg *sync.WaitGroup, i int) {
	defer wg.Done()

	log.Debugf("Provisioner worker %d starting", i)

	for {
		select {
		case host := <-work:
			log.Infof("Provisioning %s", host.Identity)

			err := provision(ctx, host)
			if err != nil {
				provErrCtr.WithLabelValues(conf.Site).Inc()
				log.Errorf("Could not provision %s: %s", host.Identity, err)
			}

			done <- host
		case <-ctx.Done():
			log.Infof("Worker %d exiting on context", i)
			return
		}
	}
}

func provision(ctx context.Context, target *host.Host) error {
	return target.Provision(ctx, fw)
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
