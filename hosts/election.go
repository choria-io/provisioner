package hosts

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/provisioner/config"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/election"
	"github.com/sirupsen/logrus"
)

func startElection(ctx context.Context, wg *sync.WaitGroup, conn choria.Connector, conf *config.Config, trigger chan struct{}, logger *logrus.Entry) error {
	defer wg.Done()

	log := logger.WithField("election", conf.LeaderElectionName)

	log.Infof("Starting leader election against %s", conf.LeaderElectionName)
	conf.Pause()

	mgr, err := jsm.New(conn.Nats())
	if err != nil {
		return err
	}

	won := func() {
		conf.Resume()
		log.Warnf("Became leader after winning election of %s", conf.LeaderElectionName)
		select {
		case trigger <- struct{}{}:
			log.Info("Triggered a discovery after becoming leader")
		}
	}

	lost := func() {
		conf.Paused()
		log.Warnf("Lost leadership of %s", conf.LeaderElectionName)
	}

	elect, err := election.NewElection(fmt.Sprintf("%s-%d", conf.Site, os.Getpid()), won, lost, conf.LeaderElectionName, mgr, election.WithBackoff(backoff.TwentySec), election.WithDebug(logger.Debugf), election.WithHeartBeatInterval(time.Second))
	if err != nil {
		return err
	}

	go func() { elect.Start(ctx) }()

	return nil
}
