// Copyright (c) 2018-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package host

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/provision"
	"github.com/choria-io/provisioner/config"
	"github.com/choria-io/tokens"
	"github.com/sirupsen/logrus"
)

type Host struct {
	Identity             string                     `json:"identity"`
	CSR                  *provision.CSRReply        `json:"csr"`
	ED25519PubKey        *provision.ED25519Reply    `json:"ed25519_pubkey"`
	Metadata             string                     `json:"inventory"`
	JWT                  *tokens.ProvisioningClaims `json:"jwt"`
	rawJWT               string
	config               map[string]string
	provisioned          bool
	ca                   string
	cert                 string
	key                  string
	sslDir               string
	serverPubKey         string
	provisionPubKey      string
	actionPolicies       map[string]interface{}
	opaPolicies          map[string]interface{}
	nonce                string
	edPubK               string
	signedServerJWT      string
	version              string
	upgradable           bool
	upgradeTargetVersion string

	discovered time.Time
	cfg        *config.Config
	token      string
	fw         *choria.Framework
	log        *logrus.Entry
	mu         *sync.Mutex
	replylock  *sync.Mutex
}

func NewHost(identity string, conf *config.Config) *Host {
	return &Host{
		Identity:    identity,
		provisioned: false,
		discovered:  time.Now(),
		mu:          &sync.Mutex{},
		replylock:   &sync.Mutex{},
		token:       conf.Token,
		cfg:         conf,
	}
}

func (h *Host) Provision(ctx context.Context, fw *choria.Framework) (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.provisioned {
		return true, nil
	}

	h.fw = fw
	h.log = fw.Logger(h.Identity)

	if !h.discovered.IsZero() {
		since := time.Since(h.discovered)
		if since > 2*h.cfg.IntervalDuration {
			return false, fmt.Errorf("skipping node that's been waiting %v", since)
		}
	}

	if h.cfg.Features.JWT {
		err := h.fetchJWT(ctx)
		if err != nil {
			return false, fmt.Errorf("could not fetch and validate JWT: %s: %s", h.Identity, err)
		}

		err = h.validateJWT()
		if err != nil {
			return false, fmt.Errorf("could not validate JWT: %s: %s", h.Identity, err)
		}
	}

	if h.cfg.Features.ED25519 {
		err := h.fetchEd25519PubKey(ctx)
		if err != nil {
			return false, fmt.Errorf("could not fetch ed25519 public key: %s", err)
		}
	}

	err := h.fetchInventory(ctx)
	if err != nil {
		return false, fmt.Errorf("could not provision %s: %s", h.Identity, err)
	}

	if h.cfg.Features.PKI {
		err = h.fetchCSR(ctx)
		if err != nil {
			return false, fmt.Errorf("could not provision %s: %s", h.Identity, err)
		}

		err = h.validateCSR()
		if err != nil {
			return false, fmt.Errorf("could not provision %s: %s", h.Identity, err)
		}
	}

	config, err := h.getConfig(ctx)
	if err != nil {
		helperErrCtr.WithLabelValues(h.cfg.Site).Inc()
		return false, err
	}

	if config.Defer {
		return false, fmt.Errorf("configuration defered: %s", config.Msg)
	}

	if config.Shutdown {
		if config.Msg == "" {
			h.log.Warnf("Shutting down host based on helper output, no reason given")
		} else {
			h.log.Warnf("Shutting down host based on helper output: %v", config.Msg)
		}

		return true, h.shutdown(ctx)
	}

	h.config = config.Configuration
	h.ca = config.CA
	h.cert = config.Certificate
	h.key = config.Key
	h.sslDir = config.SSLDir
	h.actionPolicies = make(map[string]interface{})
	h.opaPolicies = make(map[string]interface{})
	h.upgradeTargetVersion = config.UpgradeVersion

	if h.cfg.Features.ED25519 {
		err = h.generateServerJWT(config)
		if err != nil {
			return false, err
		}
	}

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
			return false, err
		}
	}

	if h.cfg.Features.VersionUpgrades && h.upgradeTargetVersion != "" {
		skipped, err := h.handleHostUpgrade(ctx)
		switch {
		case err == nil:
			if skipped {
				break
			}

			// no delay so we reprov asap
			return false, nil
		case h.cfg.UpgradesOptional:
			h.log.Warnf("Could not upgrade to %v, continuing: %v", h.upgradeTargetVersion, err)
		default:
			return true, err
		}
	}

	err = h.configure(ctx)
	if err != nil {
		return false, fmt.Errorf("configuration failed: %s", err)
	}

	err = h.restart(ctx)
	if err != nil {
		return false, fmt.Errorf("restart failed: %s", err)
	}

	h.provisioned = true

	return true, nil
}

func (h *Host) handleHostUpgrade(ctx context.Context) (bool, error) {
	if h.cfg.Features.VersionUpgrades {
		if h.cfg.UpgradesRepo == "" && !h.cfg.UpgradesOptional {
			return false, fmt.Errorf("updates_repository not configured for upgrades feature")
		}
	}

	if h.version == "" {
		return false, fmt.Errorf("did not receive a version in inventory")
	}

	if !h.upgradable {
		return false, fmt.Errorf("does not support upgrades")
	}

	cv := NewVersion(h.version)
	tv := NewVersion(h.upgradeTargetVersion)

	if cv.LessThan(tv) {
		h.log.Warnf("Will attempt version upgrade from %s to %s", h.version, h.upgradeTargetVersion)

		err := h.upgrade(ctx)
		if err != nil {
			return false, fmt.Errorf("upgrading to %v failed: %v", h.upgradeTargetVersion, err)
		}

		return false, nil
	}

	return true, nil
}

func (h *Host) generateServerJWT(c *ConfigResponse) error {
	if h.edPubK == "" {
		return fmt.Errorf("no ed25519 public key set")
	}

	if h.cfg.JWTSigningKey == "" {
		return fmt.Errorf("no jwt signing key configured using jwt_signing_key")
	}

	if h.cfg.ServerJWTValidityDuration == 0 {
		return fmt.Errorf("no default validity configured")
	}

	var (
		org         = "choria"
		validity    = h.cfg.ServerJWTValidityDuration
		collectives = []string{"mcollective"}
		pubSubs     []string
		permissions *tokens.ServerPermissions
	)

	// choria switches to a different default when v2 protocol and security is in use
	if prov, ok := c.Configuration["plugin.security.provider"]; ok && prov == "choria" {
		collectives = []string{"choria"}
	}

	if cc, ok := c.Configuration["collectives"]; ok {
		collectives = strings.Split(cc, ",")
	}

	if c.ServerClaims != nil {
		if c.ServerClaims.OrganizationUnit != "" {
			org = c.ServerClaims.OrganizationUnit
		}
		if c.ServerClaims.ExpiresAt != nil && !c.ServerClaims.ExpiresAt.IsZero() {
			validity = time.Until(c.ServerClaims.ExpiresAt.Time)
			if validity == 0 {
				h.log.Warnf("Could not parse expires claim %v", c.ServerClaims.ExpiresAt)
			}
		}
		if len(c.ServerClaims.Collectives) > 0 {
			collectives = c.ServerClaims.Collectives
		}
		if len(c.ServerClaims.AdditionalPublishSubjects) > 0 {
			pubSubs = c.ServerClaims.AdditionalPublishSubjects
		}
		if c.ServerClaims.Permissions != nil {
			permissions = c.ServerClaims.Permissions
		}
	}
	if validity <= time.Hour {
		h.log.Warnf("Server validity %v is too short, setting to default %s", validity, h.cfg.ServerJWTValidityDuration)
		validity = h.cfg.ServerJWTValidityDuration
	}

	pk, err := hex.DecodeString(h.edPubK)
	if err != nil {
		return err
	}

	claims, err := tokens.NewServerClaims(h.Identity, collectives, org, permissions, pubSubs, pk, "Choria Provisioner", validity)
	if err != nil {
		return err
	}

	if h.cfg.JWTSigningToken != "" {
		// a bunch of redundant repeated reading happens here of the same files but I prefer
		// to do that so just updating the secrets will update the running instance
		t, err := os.ReadFile(h.cfg.JWTSigningToken)
		if err != nil {
			return err
		}

		_, prik, err := choria.Ed25519KeyPairFromSeedFile(h.cfg.JWTSigningKey)
		if err != nil {
			return err
		}

		token, err := tokens.ParseClientIDTokenUnverified(string(t))
		if err != nil {
			return err
		}

		if token.TrustChainSignature == "" {
			return fmt.Errorf("no Trust Chain Signature claim in JWT Signing Token")
		}

		err = claims.AddChainIssuerData(token, prik)
		if err != nil {
			return err
		}
	}

	signed, err := tokens.SignTokenWithKeyFile(claims, h.cfg.JWTSigningKey)
	if err != nil {
		return err
	}

	h.signedServerJWT = signed

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

	var claims *tokens.ProvisioningClaims
	var err error
	if _, err = os.Stat(h.cfg.JWTVerifyCert); os.IsNotExist(err) {
		var pk []byte
		pk, err = hex.DecodeString(h.cfg.JWTVerifyCert)
		if err != nil {
			return err
		}
		claims, err = tokens.ParseProvisioningToken(h.rawJWT, ed25519.PublicKey(pk))
	} else {
		claims, err = tokens.ParseProvisioningTokenWithKeyfile(h.rawJWT, h.cfg.JWTVerifyCert)
	}
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
	names = append(names, csr.DNSNames...)

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
	m := regexp.MustCompile("^/.+/$")

	for _, reg := range regex {
		if m.MatchString(reg) {
			reg = strings.TrimLeft(reg, "/")
			reg = strings.TrimRight(reg, "/")
		}

		if matched, _ := regexp.MatchString(reg, str); matched {
			return true
		}
	}

	return false
}
