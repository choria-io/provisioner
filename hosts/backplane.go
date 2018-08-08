package hosts

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/go-backplane/backplane"
)

func startBackplane(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	opts := []backplane.Option{
		backplane.ManageInfoSource(conf),
		backplane.ManagePausable(conf),
	}

	_, err := backplane.Run(ctx, wg, conf.Management, opts...)
	if err != nil {
		return (fmt.Errorf("Could not start backplane: %s", err))
	}

	return nil
}
