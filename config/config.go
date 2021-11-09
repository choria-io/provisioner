package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ghodss/yaml"
)

// Version is the version of the app
var Version = "0.0.0"

// Config is the configuration structure
type Config struct {
	Workers                 int      `json:"workers"`
	Interval                string   `json:"interval"`
	Logfile                 string   `json:"logfile"`
	Loglevel                string   `json:"loglevel"`
	Helper                  string   `json:"helper"`
	Token                   string   `json:"token"`
	LifecycleComponent      string   `json:"lifecycle_component"`
	Insecure                bool     `json:"choria_insecure"`
	Site                    string   `json:"site"`
	MonitorPort             int      `json:"monitor_port"`
	BrokerPort              int      `json:"broker_port"`
	BrokerProvisionPassword string   `json:"broker_provisioning_password"`
	CertDenyList            []string `json:"cert_deny_list"`
	JWTVerifyCert           string   `json:"jwt_verify_cert"`
	RegoPolicy              string   `json:"rego_policy"`
	LeaderElection          bool     `json:"leader_election"`

	Features struct {
		PKI bool `json:"pki"`
		JWT bool `json:"jwt"`
	} `json:"features"`

	IntervalDuration time.Duration `json:"-"`
	File             string        `json:"-"`

	paused bool
	sync.Mutex
}

// Load reads configuration from a YAML file
func Load(file string) (*Config, error) {
	config := &Config{
		LifecycleComponent: "provision_mode_server",
		File:               file,
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s not found", file)
	}

	c, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("config file could not be read: %s", err)
	}

	j, err := yaml.YAMLToJSON(c)
	if err != nil {
		return nil, fmt.Errorf("file %s could not be parsed: %s", file, err)
	}

	err = json.Unmarshal(j, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse config file %s as YAML: %s", file, err)
	}

	if config.LifecycleComponent == "" {
		config.LifecycleComponent = "provision_mode_server"
	}

	if strings.Contains(config.LifecycleComponent, ".") || strings.Contains(config.LifecycleComponent, ">") || strings.Contains(config.LifecycleComponent, "*") {
		return nil, fmt.Errorf("invalid lifecycle component: %s", config.LifecycleComponent)
	}

	if len(config.CertDenyList) == 0 {
		config.CertDenyList = []string{
			"\\.privileged.mcollective$",
			"\\.privileged.choria$",
			"\\.mcollective$",
			"\\.choria$",
		}
	}

	config.IntervalDuration, err = time.ParseDuration(config.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %s", err)
	}

	if config.IntervalDuration < time.Minute {
		return nil, errors.New("interval is too small, minmum is 1 minute.  Valid example values are 10m or 10h")
	}

	pausedGauge.WithLabelValues(config.Site).Set(0)

	return config, nil
}
