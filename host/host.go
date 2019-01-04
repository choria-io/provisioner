package host

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/choria"
	provision "github.com/choria-io/provisioning-agent/agent"
	"github.com/choria-io/provisioning-agent/config"
	"github.com/sirupsen/logrus"
)

type Host struct {
	Identity    string              `json:"identity"`
	CSR         *provision.CSRReply `json:"csr"`
	Metadata    string              `json:"inventory"`
	config      map[string]string
	provisioned bool
	ca          string
	cert        string

	cfg       *config.Config
	token     string
	fw        *choria.Framework
	log       *logrus.Entry
	mu        *sync.Mutex
	replylock *sync.Mutex
}

func NewHost(identity string, conf *config.Config) *Host {
	return &Host{
		Identity:    identity,
		provisioned: false,
		mu:          &sync.Mutex{},
		replylock:   &sync.Mutex{},
		token:       conf.Token,
		cfg:         conf,
	}
}

func (h *Host) Provision(ctx context.Context, fw *choria.Framework) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.provisioned {
		return nil
	}

	h.fw = fw
	h.log = fw.Logger(h.Identity)

	err := h.fetchInventory(ctx)
	if err != nil {
		return fmt.Errorf("could not provision %s: %s", h.Identity, err)
	}

	if h.cfg.Features.PKI {
		err = h.fetchCSR(ctx)
		if err != nil {
			return fmt.Errorf("could not provision %s: %s", h.Identity, err)
		}

		err = h.validateCSR()
		if err != nil {
			return fmt.Errorf("could not provision %s: %s", h.Identity, err)
		}
	}

	config, err := h.getConfig(ctx)
	if err != nil {
		helperErrCtr.WithLabelValues(h.cfg.Site).Inc()
		return err
	}

	if config.Defer {
		return fmt.Errorf("configuration defered: %s", config.Msg)
	}

	h.config = config.Configuration
	h.ca = config.CA
	h.cert = config.Certificate

	err = h.configure(ctx)
	if err != nil {
		return fmt.Errorf("configuration failed: %s", err)
	}

	err = h.restart(ctx)
	if err != nil {
		return fmt.Errorf("restart failed: %s", err)
	}

	h.provisioned = true

	return nil
}

func (h *Host) String() string {
	return h.Identity
}

func (h *Host) validateCSR() error {
	if h.CSR.CSR == "" {
		return fmt.Errorf("no CSR received")
	}

	block, _ := pem.Decode([]byte(h.CSR.CSR))
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return fmt.Errorf("could not parse CSR: %s", err)
	}

	names := []string{csr.Subject.CommonName}
	for _, name := range csr.DNSNames {
		names = append(names, name)
	}

	if csr.Subject.CommonName != h.Identity {
		return fmt.Errorf("common name %s does not match identity %s", csr.Subject.CommonName, h.Identity)
	}

	for _, name := range names {
		if matchAnyRegex(name, h.cfg.CertDenyList) {
			h.log.Errorf("Denying CSR with name %s due to pattern %s", name, strings.Join(h.cfg.CertDenyList, ", "))

			return fmt.Errorf("%s matches denied certificate pattern", name)
		}
	}

	return nil
}

func matchAnyRegex(str string, regex []string) bool {
	for _, reg := range regex {
		if matched, _ := regexp.MatchString("^/.+/$", reg); matched {
			reg = strings.TrimLeft(reg, "/")
			reg = strings.TrimRight(reg, "/")
		}

		if matched, _ := regexp.MatchString(reg, str); matched {
			return true
		}
	}

	return false
}
