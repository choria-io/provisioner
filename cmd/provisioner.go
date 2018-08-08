package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/choria-io/go-choria/choria"
	cconf "github.com/choria-io/go-choria/config"
	"github.com/choria-io/provisioning-agent/config"
	"github.com/choria-io/provisioning-agent/hosts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	pidFile string
	cfile   string
	ccfile  string
	debug   bool
	ctx     context.Context
	cancel  func()
	log     *logrus.Entry
	sha     string
)

func Run() {
	app := kingpin.New("choria-provisioner", "The Choria Provisioning Framework")
	app.Version(config.Version)
	app.Author("R.I.Pienaar <rip@devco.net>")

	app.Flag("debug", "Enables debug logging").BoolVar(&debug)

	cmd := app.Command("run", "Runs the provisioner").Default()
	cmd.Flag("config", "Configuration file").Required().ExistingFileVar(&cfile)
	cmd.Flag("choria-config", "Choria configuration file").Default(choria.UserConfig()).ExistingFileVar(&ccfile)
	cmd.Flag("pid", "Write running PID to a file").StringVar(&pidFile)

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	switch command {
	case cmd.FullCommand():
		run()
	}
}

func run() {
	cfg, err := config.Load(cfile)
	kingpin.FatalIfError(err, "Provisioning could not be configured: %s", err)

	ccfg, err := cconf.NewConfig(ccfile)

	ccfg.LogLevel = cfg.Loglevel
	ccfg.LogFile = cfg.Logfile
	ccfg.Collectives = []string{"provisioning"}
	ccfg.MainCollective = "provisioning"

	if debug {
		ccfg.LogLevel = "debug"
	}

	if cfg.Insecure {
		ccfg.DisableTLS = true
	}

	if pidFile != "" {
		writePID(pidFile)
	}

	fw, err := choria.NewWithConfig(ccfg)
	kingpin.FatalIfError(err, "Provisioning could not configure Choria: %s", err)

	log = fw.Logger("provisioner")

	if cfg.MonitorPort > 0 {
		go setupPrometheus(cfg.MonitorPort)
	}

	go interruptHandler(ctx, cancel)

	err = hosts.Process(ctx, cfg, fw)
	kingpin.FatalIfError(err, "Provisioning could not start: %s", err)
}

func writePID(pidfile string) {
	if pidfile == "" {
		return
	}

	err := ioutil.WriteFile(pidfile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	kingpin.FatalIfError(err, "Could not write PID: %s", err)
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
