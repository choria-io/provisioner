// Copyright (c) 2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package hosts

import (
	"context"
	"sync"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/inter"
	election "github.com/choria-io/go-choria/providers/election/streams"
	"github.com/sirupsen/logrus"
)

func startElection(ctx context.Context, wg *sync.WaitGroup, conn inter.Connector, fw inter.Framework, trigger chan struct{}, logger *logrus.Entry) error {
	defer wg.Done()

	log := logger.WithField("election", "provisioner")
	log.Infof("Starting leader election against 'provisioner'")
	conf.Pause()

	won := func() {
		conf.Resume()
		log.Warn("Became leader after winning election")

		trigger <- struct{}{}
		log.Info("Triggered a discovery after becoming leader")
	}

	lost := func() {
		conf.Pause()
		log.Warn("Lost leadership")
		removeAllHosts()
	}

	elect, err := fw.NewElection(ctx, conn, "provisioner", true, election.OnWon(won), election.OnLost(lost), election.WithBackoff(backoff.TwentySec), election.WithDebug(log.Debugf))
	if err != nil {
		return err
	}

	go func() {
		err := elect.Start(ctx)
		if err != nil {
			log.Fatalf("Leader election failed to start: %s", err)
		}
	}()

	return nil
}
