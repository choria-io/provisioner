package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/provision"
	"github.com/choria-io/provisioner/config"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
)

type provClaims struct {
	Secure      bool   `json:"chs"`
	URLs        string `json:"chu"`
	Token       string `json:"cht"`
	SRVDomain   string `json:"chsrv"`
	ProvDefault bool   `json:"chpd"`

	jwt.StandardClaims
}

type Host struct {
	Identity        string              `json:"identity"`
	CSR             *provision.CSRReply `json:"csr"`
	Metadata        string              `json:"inventory"`
	JWT             *provClaims         `json:"jwt"`
	rawJWT          string
	config          map[string]string
	provisioned     bool
	ca              string
	cert            string
	key             string
	sslDir          string
	serverPubKey    string
	provisionPubKey string
	actionPolicies  map[string]interface{}
	opaPolicies     map[string]interface{}

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

	if h.cfg.Features.JWT {
		err := h.fetchJWT(ctx)
		if err != nil {
			return fmt.Errorf("could not fetch and validate JWT: %s: %s", h.Identity, err)
		}

		err = h.validateJWT()
		if err != nil {
			return fmt.Errorf("could not validate JWT: %s: %s", h.Identity, err)
		}
	}

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
	h.key = config.Key
	h.sslDir = config.SSLDir
	h.actionPolicies = make(map[string]interface{})
	h.opaPolicies = make(map[string]interface{})

	if len(config.OPAPolicies) > 0 {
		for k, v := range config.OPAPolicies {
			h.opaPolicies[k] = v
		}
	}

	if len(config.ActionPolicies) > 0 {
		for k, v := range config.ActionPolicies {
			h.actionPolicies[k] = v
		}
	}

	if h.key != "" {
		err = h.encryptPrivateKey()
		if err != nil {
			return err
		}
	}

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

func (h *Host) encryptPrivateKey() error {
	if h.key == "" {
		return fmt.Errorf("no key to encrypt")
	}

	if h.serverPubKey == "" {
		return fmt.Errorf("private key received from helper but server did not start Diffie-Hellman exchange")
	}

	block, _ := pem.Decode([]byte(h.key))
	if block == nil {
		return fmt.Errorf("bad key received")
	}

	serverPubKey, err := hex.DecodeString(h.serverPubKey)
	if err != nil {
		return err
	}

	provPrivate, provPublic, err := choria.ECDHKeyPair()
	if err != nil {
		return err
	}
	h.provisionPubKey = fmt.Sprintf("%x", provPublic)

	sharedSecret, err := choria.ECDHSharedSecret(provPrivate, serverPubKey)
	if err != nil {
		return err
	}

	//lint:ignore SA1019 there is no alternative
	epb, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", block.Bytes, sharedSecret, x509.PEMCipherAES256)
	if err != nil {
		return err
	}

	out := &bytes.Buffer{}
	err = pem.Encode(out, epb)
	if err != nil {
		return err
	}

	h.key = out.String()

	return nil
}

func (h *Host) String() string {
	return h.Identity
}

func (h *Host) validateJWT() error {
	if h.rawJWT == "" {
		return fmt.Errorf("no JWT received")
	}

	if h.cfg.JWTVerifyCert == "" {
		return fmt.Errorf("no JWT verification certificate configured, cannot validate JWT")
	}

	claims := &provClaims{}
	_, err := jwt.ParseWithClaims(h.rawJWT, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unsupported signing method in token")
		}

		pem, err := ioutil.ReadFile(h.cfg.JWTVerifyCert)
		if err != nil {
			return nil, fmt.Errorf("could not read JWT verification certificate: %s", err)
		}

		return jwt.ParseRSAPublicKeyFromPEM(pem)
	})
	if err != nil {
		return err
	}

	h.JWT = claims

	return nil
}

func (h *Host) validateCSR() error {
	if h.CSR == nil {
		return fmt.Errorf("no CSR received")
	}

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
