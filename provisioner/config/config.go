package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ghodss/yaml"
)

type Config struct {
	Workers     int    `json:"workers"`
	Interval    string `json:"interval"`
	Logfile     string `json:"logfile"`
	Loglevel    string `json:"loglevel"`
	Helper      string `json:"helper"`
	Token       string `json:"token"`
	TLS         bool   `json:"tls"`
	Site        string `json:"site"`
	MonitorPort int    `json:"monitor_port"`
	Features    struct {
		PKI bool `json:"pki"`
	} `json:"features"`

	IntervalDuration time.Duration `json:"-"`
	File             string        `json:"-"`
}

// Load reads configuration from a YAML file
func Load(file string) (*Config, error) {
	config := &Config{}

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

	config.IntervalDuration, err = time.ParseDuration(config.Interval)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %s", err)
	}

	return config, nil
}
