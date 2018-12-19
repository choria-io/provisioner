package provision

import (
	"sync"

	"github.com/choria-io/go-choria/build"
	"github.com/choria-io/go-choria/server"
	"github.com/choria-io/go-choria/server/agents"
	"github.com/choria-io/mcorpc-agent-provider/mcorpc"
	"github.com/sirupsen/logrus"
)

type Reply struct {
	Message string `json:"message"`
}

var mu = &sync.Mutex{}
var allowRestart = true
var log *logrus.Entry

func New(mgr server.AgentManager) (*mcorpc.Agent, error) {
	metadata := &agents.Metadata{
		Name:        "choria_provision",
		Description: "Choria Provisioner",
		Author:      "R.I.Pienaar <rip@devco.net>",
		Version:     build.Version,
		License:     build.License,
		Timeout:     20,
		URL:         "http://choria.io",
	}

	log = mgr.Logger()

	agent := mcorpc.New("choria_provision", metadata, mgr.Choria(), log)

	agent.MustRegisterAction("gencsr", csrAction)
	agent.MustRegisterAction("configure", configureAction)
	agent.MustRegisterAction("restart", restartAction)
	agent.MustRegisterAction("reprovision", reprovisionAction)
	agent.MustRegisterAction("release_update", releaseUpdateAction)

	return agent, nil
}
