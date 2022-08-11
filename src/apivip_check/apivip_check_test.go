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

	v31_types "github.com/coreos/ignition/v2/config/v3_1/types"
	v32_types "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gomega_format "github.com/onsi/gomega/format"
)

const (
	TestWorkerIgnitionPath = "/config/worker"
	AcceptHeader           = "application/vnd.coreos.ignition+json; version=3.2.0"
	IgnitionSource         = "http://127.0.0.1:1234"
)

var _ = Describe("API connectivity check test", func() {
	gomega_format.CharactersAroundMismatchToInclude = 800

	var log logrus.FieldLogger
	var srv *httptest.Server

	BeforeEach(func() {
		log = logrus.New()
	})

	AfterEach(func() {
		if srv != nil {
			srv.Close()
		}
	})

	Context("Ignition file", func() {
		It("Download ignition file successfully", func() {
			srv = serverMock(ignitionMock)
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)).
				withExpectedIgnition(getIgnitionConfig()).
				withLuks().
				checkResponse()
		})

		It("Download old ignition file successfully", func() {
			srv = serverMock(ignitionMock31)
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)).
				withExpectedIgnition(getIgnitionConfigV31Upgraded()).
				checkResponse()
		})

		It("ignition not is json format", func() {
			srv = serverMock(ignitionMockInvalid)
			errorMessage := `ignition file download failed: response is not valid json:
invalid
parse error is: invalid character 'i' looking for beginning of value`
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)).
				withExpectedError(errorMessage).
				withExpectedFailure().
				checkResponse()
		})

		It("Invalid ignition format", func() {
			srv = serverMock(ignitionMockInvalidFormat)
			errorMessage := `response is not a valid ignition file:
{"ignition": {}}
parse error is: invalid config version (couldn't parse)`

			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)).
				withExpectedError(errorMessage).
				withExpectedFailure().
				checkResponse()
		})

		It("Empty ignition", func() {
			srv = serverMock(ignitionMockEmpty)
			errorMessage := "ignition file download failed: server responsed with status code 200 but the response was empty"
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, nil), log)).
				withExpectedError(errorMessage).
				withExpectedFailure().
				checkResponse()
		})
	})

	Context("API URL", func() {
		It("Invalid API URL", func() {
			url := "http://127.0.0.1:2345"
			errorMessage := `ignition file download failed: request failed: Get "http://127.0.0.1:2345/config/worker": dial tcp 127.0.0.1:2345: connect: connection refused`
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&url, false, nil, nil), log)).
				withExpectedError(errorMessage).
				withExpectedFailure().
				checkResponse()
		})

		It("Missing API URL", func() {
			newResponseChecker(CheckAPIConnectivity(getRequestStr(nil, false, nil, nil), log)).
				withExpectedError("internal error - service request is missing URL").
				withExpectedExitCode(-1).
				withExpectedURL("<unknown URL due to internal error>").
				withExpectedFailure().
				checkResponse()
		})

		It("Bearer Token", func() {
			ignitionToken := "secrettoken"
			srv = serverMock(bearerIgnitionMock(ignitionToken))
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, &ignitionToken), log)).
				withExpectedIgnition(getIgnitionConfig()).
				withLuks().
				checkResponse()
		})

		It("Wrong Bearer Token", func() {
			ignitionToken := "secrettoken"
			srv = serverMock(bearerIgnitionMock("anothertoken"))
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, nil, &ignitionToken), log)).
				withExpectedFailure().
				withExpectedError("ignition file download failed: bad status code: 401. server response: Invalid token").
				checkResponse()
		})
	})

	Context("CA Cert", func() {
		It("Valid Cert", func() {
			servConfig, caPEM, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			srv, err = httpsServerMock(servConfig, ignitionMock)
			Expect(err).NotTo(HaveOccurred())
			encodedCaCert := b64.StdEncoding.EncodeToString(caPEM)
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, &encodedCaCert, nil), log)).
				withExpectedIgnition(getIgnitionConfig()).
				withLuks().
				checkResponse()
		})

		It("Invalid Cert", func() {
			servConfig, _, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			srv, err = httpsServerMock(servConfig, ignitionMock)
			Expect(err).NotTo(HaveOccurred())
			caCert := "somecert"
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, &caCert, nil), log)).
				withExpectedFailure().
				withExpectedError("ignition file download failed: unable to parse cert").
				checkResponse()
		})

		It("Wrong Cert", func() {
			_, cert, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			servConfig, _, err := certsetup()
			Expect(err).NotTo(HaveOccurred())
			srv, err = httpsServerMock(servConfig, ignitionMock)
			Expect(err).NotTo(HaveOccurred())
			wrongCert := b64.StdEncoding.EncodeToString(cert)
			newResponseChecker(CheckAPIConnectivity(getRequestStr(&srv.URL, false, &wrongCert, nil), log)).
				withExpectedFailure().
				withExpectedErrorRegex(`ignition file download failed: request failed: Get "https://127.0.0.1:[0-9]*/config/worker": x509: certificate signed by unknown authority \(possibly because of "x509: invalid signature: parent certificate cannot sign this kind of certificate" while trying to verify candidate authority certificate "Company, INC."\)`).
				checkResponse()
		})
	})
})

func getIgnitionConfig() v32_types.Config {
	tpm2Enabled := true
	device := "/dev/disk"
	return v32_types.Config{
		Ignition: v32_types.Ignition{Version: "3.2.0"},
		Storage: v32_types.Storage{
			Luks: []v32_types.Luks{
				{
					Clevis: &v32_types.Clevis{Tpm2: &tpm2Enabled},
					Device: &device,
				},
			},
		},
	}
}

func getIgnitionConfigV31() v31_types.Config {
	return v31_types.Config{
		Ignition: v31_types.Ignition{Version: "3.1.0"},
	}
}

func getIgnitionConfigV31Upgraded() v32_types.Config {
	return v32_types.Config{
		Ignition: v32_types.Ignition{Version: "3.2.0"},
	}
}

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
			w.WriteHeader(http.StatusUnauthorized)
			response := []byte("Invalid token")
			n, err := w.Write(response)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(len(response)))
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

	ignitionConfig := getIgnitionConfig()
	configBytes, err := json.Marshal(ignitionConfig)
	if err != nil {
		logrus.Error("failed to marshal config to json")
		return
	}
	_, _ = w.Write(configBytes)
}

func ignitionMock31(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") != AcceptHeader {
		logrus.Error("missing Accept header in request")
		return
	}

	ignitionConfig := getIgnitionConfigV31()
	configBytes, err := json.Marshal(ignitionConfig)
	if err != nil {
		logrus.Error("failed to marshal config to json")
		return
	}
	_, _ = w.Write(configBytes)
}

func ignitionMockInvalid(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("invalid"))
}

func ignitionMockInvalidFormat(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`{"ignition": {}}`))
}

func ignitionMockEmpty(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte{})
}

type responseChecker struct {
	stdout              string
	stderr              string
	exitCode            int
	expectedStdout      *string
	expectedStderr      string
	expectedExitCode    int
	success             bool
	luks                bool
	expectedURL         string
	expectedError       *string
	expectedErrorRegex  *string
	expectedIgnition    *v32_types.Config
	expectedIgnitionV31 *v31_types.Config
}

func newResponseChecker(stdout string, stderr string, exitCode int) *responseChecker {
	return &responseChecker{
		stdout:              stdout,
		stderr:              stderr,
		exitCode:            exitCode,
		expectedStdout:      nil,
		expectedStderr:      "",
		expectedExitCode:    0,
		success:             true,
		luks:                false,
		expectedURL:         "http://127.0.0.1:40313/config/worker",
		expectedError:       nil,
		expectedErrorRegex:  nil,
		expectedIgnition:    nil,
		expectedIgnitionV31: nil,
	}
}

func (r *responseChecker) withExpectedFailure() *responseChecker {
	r.success = false
	r.expectedStderr = "ignition download error"
	return r
}

func (r *responseChecker) withExpectedExitCode(expectedExitCode int) *responseChecker {
	r.expectedExitCode = expectedExitCode
	return r
}

func (r *responseChecker) withLuks() *responseChecker {
	r.luks = true
	return r
}

func (r *responseChecker) withExpectedURL(expectedURL string) *responseChecker {
	r.expectedURL = expectedURL
	return r
}

func (r *responseChecker) withExpectedError(expectedError string) *responseChecker {
	r.expectedError = &expectedError
	return r
}

func (r *responseChecker) withExpectedErrorRegex(expectedErrorRegex string) *responseChecker {
	r.expectedErrorRegex = &expectedErrorRegex
	return r
}

func (r *responseChecker) withExpectedIgnition(expectedIgnition v32_types.Config) *responseChecker {
	r.expectedIgnition = &expectedIgnition
	return r
}

func (r *responseChecker) checkResponse() {
	if r.expectedStdout != nil {
		Expect(r.stdout).To(Equal(r.expectedStdout))
	}
	Expect(r.stderr).To(Equal(r.expectedStderr))
	Expect(r.exitCode).To(Equal(r.expectedExitCode))

	var response models.APIVipConnectivityResponse
	Expect(json.Unmarshal([]byte(r.stdout), &response)).To(Succeed())
	Expect(response.IsSuccess).To(Equal(r.success))

	if r.expectedError != nil {
		Expect(response.DownloadError).To(Equal(*r.expectedError))
	}

	if r.expectedErrorRegex != nil {
		Expect(response.DownloadError).To(MatchRegexp(*r.expectedErrorRegex))
	}

	if r.success {
		if r.expectedIgnition != nil {
			var responseConfig v32_types.Config
			err := json.Unmarshal([]byte(response.Ignition), &responseConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(responseConfig).To(Equal(*r.expectedIgnition))

			if r.luks {
				ignitionConfig := getIgnitionConfig()
				Expect(responseConfig.Storage.Luks).To(Equal(ignitionConfig.Storage.Luks))
			} else {
				Expect(responseConfig.Storage.Luks).To(Equal([]v32_types.Luks(nil)))
			}
		}

		if r.expectedIgnitionV31 != nil {
			var responseConfig v31_types.Config
			err := json.Unmarshal([]byte(response.Ignition), &responseConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(responseConfig).To(Equal(*r.expectedIgnitionV31))
		}

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
