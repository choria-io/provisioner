// Copyright (c) 2019-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package host

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/choria-io/go-choria/choria"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/provision"
	"github.com/choria-io/go-choria/tokens"
	"github.com/choria-io/provisioner/config"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChoria(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Host")
}

var _ = Describe("Host", func() {
	var h *Host

	BeforeEach(func() {
		log := logrus.NewEntry(logrus.New())
		log.Logger.Out = ioutil.Discard

		h = &Host{
			Identity: "ginkgo.example.net",
			CSR:      &provision.CSRReply{},
			log:      log,
			cfg: &config.Config{
				CertDenyList: []string{
					"\\.privileged.mcollective$",
					"\\.privileged.choria$",
					"\\.mcollective$",
					"\\.choria$",
				},
			},
		}
	})

	Describe("generateServerJWT", func() {
		var (
			pubK ed25519.PublicKey
			err  error
		)

		BeforeEach(func() {
			pubK, _, err = choria.Ed25519KeyPair()
			Expect(err).ToNot(HaveOccurred())

			h.edPubK = hex.EncodeToString(pubK)
			h.cfg.JWTSigningKey = "testdata/jwt-signer.pem"
			h.cfg.ServerJWTValidityDuration = time.Hour
		})

		It("Should handle no pub key", func() {
			h.edPubK = ""
			Expect(h.generateServerJWT(nil)).To(MatchError("no ed25519 public key set"))
		})

		It("Should handle no signing key", func() {
			h.edPubK = "x"
			h.cfg.JWTSigningKey = ""
			Expect(h.generateServerJWT(nil)).To(MatchError("no jwt signing key configured using jwt_signing_key"))
		})

		It("Should use the collectives from the server if known", func() {
			config := &ConfigResponse{
				Configuration: map[string]string{
					"collectives": "one,two",
				},
			}

			err = h.generateServerJWT(config)
			Expect(err).ToNot(HaveOccurred())
			Expect(h.signedServerJWT).ToNot(Equal(""))

			claims, err := tokens.ParseServerTokenWithKeyfile(h.signedServerJWT, "testdata/jwt-signer-public.pem")
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Collectives).To(Equal([]string{"one", "two"}))
			Expect(claims.OrganizationUnit).To(Equal("choria"))
			Expect(claims.Permissions).To(BeNil())
			Expect(claims.AdditionalPublishSubjects).To(HaveLen(0))
			Expect(claims.Issuer).To(Equal("Choria Provisioner"))
			Expect(claims.ChoriaIdentity).To(Equal(h.Identity))
			until := time.Until(claims.ExpiresAt.Time)
			Expect(until).To(BeNumerically("~", time.Hour, time.Second))
			Expect(claims.PublicKey).To(Equal(h.edPubK))
		})

		It("Should use the configured claims", func() {})
	})

	Describe("validateCSR", func() {
		It("Should handle no CSR", func() {
			Expect(h.validateCSR()).To(MatchError("no CSR received"))
		})

		It("Should ensure the names match identity", func() {
			csr, _, err := gencsr("notme.example.net", []string{})
			Expect(err).ToNot(HaveOccurred())
			h.CSR.CSR = string(csr)
			Expect(h.validateCSR()).To(MatchError("common name notme.example.net does not match identity ginkgo.example.net"))
		})

		It("Should catch bad common names", func() {
			for _, name := range strings.Fields("bob.choria bob.mcollective bob.privileged.choria bob.privileged.mcollective") {
				h.Identity = name
				csr, _, err := gencsr(name, []string{})
				Expect(err).ToNot(HaveOccurred())
				h.CSR.CSR = string(csr)
				Expect(h.validateCSR()).To(MatchError(fmt.Sprintf("%s matches denied certificate pattern", name)))
			}
		})

		It("Should catch bad DNS names", func() {
			for _, name := range strings.Fields("bob.choria bob.mcollective bob.privileged.choria bob.privileged.mcollective") {
				csr, _, err := gencsr("ginkgo.example.net", []string{"something.something", name, "something.else"})
				Expect(err).ToNot(HaveOccurred())
				h.CSR.CSR = string(csr)
				Expect(h.validateCSR()).To(MatchError(fmt.Sprintf("%s matches denied certificate pattern", name)))
			}
		})

		It("Should handle valid names", func() {
			csr, _, err := gencsr("ginkgo.example.net", []string{"something.something", "something.else"})
			Expect(err).ToNot(HaveOccurred())
			h.CSR.CSR = string(csr)
			Expect(h.validateCSR()).To(BeNil())
		})
	})

	Describe("encryptPrivateKey", func() {
		It("Should correctly encrypt the private key", func() {
			Expect(h.encryptPrivateKey()).To(MatchError("no key to encrypt"))

			// create an unencrypted key
			pk, err := rsa.GenerateKey(rand.Reader, 1024)
			Expect(err).ToNot(HaveOccurred())
			pkBytes := x509.MarshalPKCS1PrivateKey(pk)
			pkPem := &bytes.Buffer{}
			err = pem.Encode(pkPem, &pem.Block{Bytes: pkBytes, Type: "RSA PRIVATE KEY"})
			Expect(err).ToNot(HaveOccurred())
			h.key = pkPem.String()

			// pubkey received from choria
			h.serverPubKey = "88a9a0ed27dc93c29466ea2bef99e078342b27e7a1d789fc35a9131f86c3a022"

			// encrypt it with the calculated shared secret based on serverPubKey
			h.encryptPrivateKey()

			blk, _ := pem.Decode([]byte(h.key))
			Expect(blk).ToNot(BeNil())
			//lint:ignore SA1019 there is no alternative
			Expect(x509.IsEncryptedPEMBlock(blk)).To(BeTrue())

			// make sure the server can decode
			srvPri, err := hex.DecodeString("67e4a9b3934a3030470ed7a30f89eeaf7dab7b492aa9ee02fb864d690b7e6eeb")
			Expect(err).ToNot(HaveOccurred())

			provPub, err := hex.DecodeString(h.provisionPubKey)
			Expect(err).ToNot(HaveOccurred())

			shared, err := choria.ECDHSharedSecret(srvPri, provPub)
			Expect(err).ToNot(HaveOccurred())

			//lint:ignore SA1019 there is no alternative
			clearBytes, err := x509.DecryptPEMBlock(blk, shared)
			Expect(err).ToNot(HaveOccurred())

			Expect(clearBytes).To(Equal(pkBytes))
		})
	})
})

func gencsr(cn string, altnames []string) (csr []byte, key []byte, err error) {
	if cn == "" {
		return csr, key, fmt.Errorf("common name is required")
	}

	subj := pkix.Name{
		CommonName: cn,
	}.ToRDNSequence()

	asn1Subj, err := asn1.Marshal(subj)
	if err != nil {
		return csr, key, err
	}

	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	if len(altnames) > 0 {
		template.DNSNames = altnames
	}

	keyBytes, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return csr, key, err
	}

	key = pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(keyBytes),
		},
	)

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)
	if err != nil {
		return csr, key, err
	}

	csr = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})

	return csr, key, nil
}
