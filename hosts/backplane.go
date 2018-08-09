package hosts

import (
	"context"
	"sync"

	"github.com/choria-io/go-backplane/backplane"
)

func startBackplane(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	log.Info("Starting choria management backplane")

	opts := []backplane.Option{
		backplane.ManageInfoSource(conf),
		backplane.ManagePausable(conf),
	}

	_, err := backplane.Run(ctx, wg, conf.Management, opts...)
	if err != nil {
		log.Errorf("Could not start backplane: %s", err)
	}

	return nil
}
