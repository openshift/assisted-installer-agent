package apivip_check

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	TestWorkerIgnitionPath = "/config/worker"
	AcceptHeader           = "application/vnd.coreos.ignition+json; version=3.2.0"
	IgnitionSource         = "http://127.0.0.1:1234"
)

var _ = Describe("API connectivity check test", func() {
	var log logrus.FieldLogger
	var srv *httptest.Server

	BeforeEach(func() {
		log = logrus.New()
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("Ignition file", func() {
		It("Download ignition file successfully", func() {
			srv = serverMock(ignitionMock)
			stdout, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(BeEmpty())
			checkResponse(stdout, true)
		})

		It("Invalid ignition file format", func() {
			srv = serverMock(ignitionMockInvalid)
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(ContainSubstring("Error unmarshaling Ignition string"))
		})

		It("Empty ignition", func() {
			srv = serverMock(ignitionMockEmpty)
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(Equal("Failed to download worker.ign file: Empty Ignition file"))
		})
	})

	Context("API URL", func() {
		It("Invalid API URL", func() {
			url := "http://127.0.0.1:2345"
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&url, false, nil, nil), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(ContainSubstring("HTTP download failure"))
		})

		It("Missing API URL", func() {
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(nil, false, nil, nil), log)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(Equal("Missing URL in checkAPIRequest"))
		})

		It("Bearer Token", func() {
			ignitionToken := "secrettoken"
			srv = serverMock(bearerIgnitionMock(ignitionToken))
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, &ignitionToken), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(BeEmpty())
		})

		It("Wrong Bearer Token", func() {
			ignitionToken := "secrettoken"
			srv = serverMock(bearerIgnitionMock("anothertoken"))
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, &ignitionToken), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(ContainSubstring("HTTP download failure"))
		})
	})

	Context("CA Cert", func() {
		It("Valid Cert", func() {
			servConfig, caPEM, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			srv, err = httpsServerMock(servConfig, ignitionMock)
			Expect(err).NotTo(HaveOccurred())
			encodedCaCert := b64.StdEncoding.EncodeToString(caPEM)
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, &encodedCaCert, nil), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(BeEmpty())
		})

		It("Invalid Cert", func() {
			servConfig, _, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			srv, err = httpsServerMock(servConfig, ignitionMock)
			Expect(err).NotTo(HaveOccurred())
			caCert := "somecert"
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, &caCert, nil), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(ContainSubstring("unable to parse cert"))
		})

		It("Wrong Cert", func() {
			_, cert, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			servConfig, _, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			srv, err = httpsServerMock(servConfig, ignitionMock)
			Expect(err).NotTo(HaveOccurred())
			wrongCert := b64.StdEncoding.EncodeToString(cert)
			_, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false, &wrongCert, nil), log)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(ContainSubstring("unknown authority"))
		})
	})
})

func getRequestStr(url *string, verifyCidr bool, caCert *string, token *string) string {
	if url != nil {
		ignitionURL := *url + TestWorkerIgnitionPath
		url = &ignitionURL
	}
	request := models.APIVipConnectivityRequest{
		URL:                   url,
		VerifyCidr:            verifyCidr,
		CaCertificate:         caCert,
		IgnitionEndpointToken: token,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return ""
	}
	return string(requestBytes)
}

func serverMock(mock func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc(TestWorkerIgnitionPath, mock)
	srv := httptest.NewServer(handler)
	return srv
}

func bearerIgnitionMock(token string) func(w http.ResponseWriter, r *http.Request) {
	bearerToken := fmt.Sprintf("Bearer %s", token)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != bearerToken {
			logrus.Error("missing Auth header in request")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ignitionMock(w, r)
	}
}

func ignitionMock(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") != AcceptHeader {
		logrus.Error("missing Accept header in request")
		return
	}

	ignitionConfig, err := FormatNodeIgnitionFile(IgnitionSource)
	if err != nil {
		return
	}
	_, _ = w.Write(ignitionConfig)
}

func ignitionMockInvalid(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("invalid"))
}

func ignitionMockEmpty(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte{})
}

func checkResponse(stdout string, success bool) {
	var response models.APIVipConnectivityResponse
	Expect(json.Unmarshal([]byte(stdout), &response)).ToNot(HaveOccurred())
	Expect(success).To(Equal(response.IsSuccess))

	if success {
		ignition, err := FormatNodeIgnitionFile(IgnitionSource)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(ignition)).To(Equal(response.Ignition))
	}
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API connectivity check unit tests")
}

func httpsServerMock(servConfig *tls.Config, mock func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, error) {
	handler := http.NewServeMux()
	handler.HandleFunc(TestWorkerIgnitionPath, mock)
	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = servConfig
	srv.StartTLS()
	return srv, nil
}

func certsetup() (*tls.Config, []byte, error) {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, nil, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, nil, err
	}

	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return nil, nil, err
	}

	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, nil, err
	}

	serverTLSConf := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}

	return serverTLSConf, certPEM.Bytes(), nil
}
