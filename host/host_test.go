package host

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/provision"
	"github.com/choria-io/provisioning-agent/config"

	. "github.com/onsi/ginkgo"
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
