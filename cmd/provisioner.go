// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/choria-io/go-choria/choria"
	cconf "github.com/choria-io/go-choria/config"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/provisioner/config"
	"github.com/choria-io/provisioner/hosts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/fisk"
)

var (
	pidFile string
	cfile   string
	ccfile  string
	debug   bool
	ctx     context.Context
	cancel  func()
	log     *logrus.Entry
)

func Run() {
	app := fisk.New("choria-provisioner", "The Choria Provisioning Framework")
	app.Version(config.Version)
	app.Author("R.I.Pienaar <rip@devco.net>")

	app.Flag("debug", "Enables debug logging").BoolVar(&debug)

	cmd := app.Command("run", "Runs the provisioner").Default()
	cmd.Flag("config", "Configuration file").Required().ExistingFileVar(&cfile)
	cmd.Flag("choria-config", "Choria configuration file").Default(choria.UserConfig()).ExistingFileVar(&ccfile)
	cmd.Flag("pid", "Write running PID to a file").StringVar(&pidFile)

	command := fisk.MustParse(app.Parse(os.Args[1:]))

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	switch command {
	case cmd.FullCommand():
		run()
	}
}

func run() {
	cfg, err := config.Load(cfile)
	fisk.FatalIfError(err, "Provisioning could not be configured: %s", err)

	ccfg, err := cconf.NewConfig(ccfile)
	fisk.FatalIfError(err, "Could not load Choria Configuration: %s", err)

	ccfg.LogLevel = cfg.Loglevel
	ccfg.LogFile = cfg.Logfile
	ccfg.Collectives = []string{"provisioning"}
	ccfg.MainCollective = "provisioning"

	if debug {
		ccfg.LogLevel = "debug"
	}

	if cfg.Insecure {
		ccfg.DisableTLS = true
		protocol.Secure = "false"
		ccfg.Choria.SecurityProvider = "file"
	}

	if cfg.BrokerProvisionPassword != "" {
		ccfg.Choria.NatsUser = "provisioner"
		ccfg.Choria.NatsPass = cfg.BrokerProvisionPassword
	}

	fw, err := choria.NewWithConfig(ccfg)
	fisk.FatalIfError(err, "Provisioning could not configure Choria: %s", err)

	log = fw.Logger("provisioner")

	if cfg.MonitorPort > 0 {
		go setupPrometheus(cfg.MonitorPort)
	}

	go interruptHandler(ctx, cancel)

	if pidFile != "" {
		writePID(pidFile)
		defer os.Remove(pidFile)
	}

	err = hosts.Process(ctx, cfg, fw)
	fisk.FatalIfError(err, "Provisioning could not start: %s", err)
}

func writePID(pidfile string) {
	if pidfile == "" {
		return
	}

	err := os.WriteFile(pidfile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	fisk.FatalIfError(err, "Could not write PID: %s", err)
}

func interruptHandler(ctx context.Context, cancel func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func setupPrometheus(port int) {
	log.Infof("Listening for /metrics on %d", port)
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
