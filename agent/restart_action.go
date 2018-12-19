package provision

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"syscall"
	"time"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/config"
	lifecycle "github.com/choria-io/go-lifecycle"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/sirupsen/logrus"
)

type RestartRequest struct {
	Token string `json:"token"`
	Splay int    `json:"splay"`
}

func restartAction(ctx context.Context, req *mcorpc.Request, reply *mcorpc.Reply, agent *mcorpc.Agent, conn choria.ConnectorInfo) {
	mu.Lock()
	defer mu.Unlock()

	if !agent.Choria.ProvisionMode() && build.ProvisionToken == "" {
		abort("Cannot restart a server that is not in provisioning mode or with no token set", reply)
		return
	}

	args := &RestartRequest{}
	if !mcorpc.ParseRequestData(args, req, reply) {
		return
	}

	if !checkToken(args.Token, reply) {
		return
	}

	cfg, err := config.NewConfig(agent.Config.ConfigFile)
	if err != nil {
		abort(fmt.Sprintf("Configuration %s could not be parsed, restart cannot continue: %s", agent.Config.ConfigFile, err), reply)
		return
	}

	if cfg.Choria.Provision {
		abort(fmt.Sprintf("Configuration %s enables provisioning, restart cannot continue", agent.Config.ConfigFile), reply)
		return
	}

	if args.Splay == 0 {
		args.Splay = 10
	}

	splay := time.Duration(rand.Intn(args.Splay) + 2)

	agent.Log.Warnf("Restarting server via request %s from %s (%s) with splay %d", req.RequestID, req.CallerID, req.SenderID, splay)

	go restart(splay, agent.Log)

	err = agent.ServerInfoSource.NewEvent(lifecycle.Shutdown)
	if err != nil {
		agent.Log.Errorf("Could not publish shutdown event: %s", err)
	}

	reply.Data = Reply{fmt.Sprintf("Restarting Choria Server after %ds", splay)}
}

func restart(splay time.Duration, log *logrus.Entry) {
	if !allowRestart {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	log.Warnf("Restarting Choria Server after %ds splay time", splay)
	time.Sleep(splay * time.Second)

	err := syscall.Exec(os.Args[0], os.Args, os.Environ())
	if err != nil {
		log.Errorf("Could not restart server: %s", err)
	}
}
