// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package host

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/choria-io/go-choria/opa"
	"github.com/choria-io/go-choria/tokens"
	"github.com/choria-io/provisioner/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type ConfigResponse struct {
	Defer          bool                 `json:"defer"`
	Msg            string               `json:"msg"`
	Key            string               `json:"key"`
	Certificate    string               `json:"certificate"`
	CA             string               `json:"ca"`
	SSLDir         string               `json:"ssldir"`
	ServerClaims   *tokens.ServerClaims `json:"server_claims"`
	Configuration  map[string]string    `json:"configuration"`
	ActionPolicies map[string]string    `json:"action_policies"`
	OPAPolicies    map[string]string    `json:"opa_policies"`
}

func (h *Host) shouldConfigure(ctx context.Context) (should bool, err error) {
	if h.cfg.RegoPolicy == "" {
		return true, nil
	}

	inventory := map[string]interface{}{}
	if h.Metadata != "" {
		err = json.Unmarshal([]byte(h.Metadata), &inventory)
		if err != nil {
			h.log.Errorf("host inventory unmarshaling failed while setting up OPA query: %s", err)
		}
	}

	inputs := map[string]interface{}{
		"identity":  h.Identity,
		"inventory": inventory,
		"claims":    map[string]interface{}{},
		"csr":       "",
	}

	if h.JWT != nil {
		inputs["claims"] = map[string]interface{}{
			"secure":     h.JWT.Secure,
			"urls":       h.JWT.URLs,
			"token":      h.JWT.Token,
			"srv_domain": h.JWT.SRVDomain,
			"default":    h.JWT.ProvDefault,
			"issued_at":  h.JWT.IssuedAt,
			"expires_at": h.JWT.ExpiresAt,
		}
	}

	if h.CSR != nil {
		csrm, err := h.csrAsMap()
		if err != nil {
			h.log.Errorf("could not parse CSR: %s", err)
		} else {
			inputs["claims"].(map[string]interface{})["csr"] = csrm
		}
	}

	rego, err := opa.New("io.choria.provisioner", "data.io.choria.provisioner.allow", opa.Logger(h.log), opa.File(h.cfg.RegoPolicy))
	if err != nil {
		return false, err
	}

	return rego.Evaluate(ctx, inputs)
}

func (h *Host) csrAsMap() (csr map[string]interface{}, err error) {
	if h.CSR == nil {
		return nil, fmt.Errorf("no csr data set")
	}

	req, err := x509.ParseCertificateRequest([]byte(h.CSR.CSR))
	if err != nil {
		return nil, fmt.Errorf("invalid CSR: %s", err)
	}

	reqj, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("could not marshal CSR request: %s", err)
	}

	err = json.Unmarshal(reqj, &csr)
	return csr, err
}

func (h *Host) getConfig(ctx context.Context) (*ConfigResponse, error) {
	r := &ConfigResponse{}

	input, err := json.Marshal(h)
	if err != nil {
		return nil, fmt.Errorf("could not JSON encode host: %s", err)
	}

	err = runDecodedHelper(ctx, []string{}, string(input), r, h.cfg, h.log)
	if err != nil {
		return nil, fmt.Errorf("could not invoke configure helper: %s", err)
	}

	return r, nil
}

func runDecodedHelper(ctx context.Context, args []string, input string, output interface{}, cfg *config.Config, log *logrus.Entry) error {
	o, err := runHelper(ctx, args, input, cfg)
	if err != nil {
		return err
	}

	err = json.Unmarshal(o, output)
	if err != nil {
		return fmt.Errorf("cannot decode output from %s: %s", cfg.Helper, err)
	}

	return nil
}

func runHelper(ctx context.Context, args []string, input string, cfg *config.Config) ([]byte, error) {
	obs := prometheus.NewTimer(helperDuration.WithLabelValues(cfg.Site))
	defer obs.ObserveDuration()

	if cfg.Paused() {
		return nil, fmt.Errorf("Provisioning is paused, cannot perform %s", cfg.Helper)
	}

	tctx, cancel := context.WithTimeout(ctx, time.Duration(10*time.Second))
	defer cancel()

	execution := exec.CommandContext(tctx, cfg.Helper, args...)

	stdin, err := execution.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot create stdin for %s: %s", cfg.Helper, err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, input)
	}()

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot open STDOUT for %s: %s", cfg.Helper, err)
	}
	defer stdout.Close()

	err = execution.Start()
	if err != nil {
		return nil, fmt.Errorf("cannot start %s: %s", cfg.Helper, err)
	}

	buf := new(bytes.Buffer)

	n, err := buf.ReadFrom(stdout)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s output: %s", cfg.Helper, err)
	}
	if n == 0 {
		return nil, fmt.Errorf("cannot read %s output: zero bytes received", cfg.Helper)
	}

	err = execution.Wait()
	if err != nil {
		return nil, fmt.Errorf("could not wait for %s: %s", cfg.Helper, err)
	}

	if !execution.ProcessState.Success() {
		return nil, fmt.Errorf("could not run helper %s: exited with non 0 exitcode", cfg.Helper)
	}

	return buf.Bytes(), nil
}
