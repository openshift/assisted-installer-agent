package apivip_check

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			_, _, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false), log)
			Expect(exitCode).Should(Equal(0))
		})

		It("Invalid ignition file format", func() {
			srv = serverMock(ignitionMockInvalid)
			_, _, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false), log)
			Expect(exitCode).Should(Equal(0))
		})

		It("Empty ignition", func() {
			srv = serverMock(ignitionMockEmpty)
			_, _, exitCode := CheckAPIConnectivity(getRequestStr(&srv.URL, false), log)
			Expect(exitCode).Should(Equal(0))
		})
	})

	Context("API URL", func() {
		It("Invalid API URL", func() {
			url := "http://127.0.0.1:2345"
			_, _, exitCode := CheckAPIConnectivity(getRequestStr(&url, false), log)
			Expect(exitCode).Should(Equal(0))
		})

		It("Missing API URL", func() {
			_, _, exitCode := CheckAPIConnectivity(getRequestStr(nil, false), log)
			Expect(exitCode).Should(Equal(-1))
		})
	})

	Context("Verify CIDR", func() {
		It("Verification success - hostname", func() {
			srv = serverMock(ignitionMock)
			err := verifyCIDR("http://localhost:1234")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Verification success - IP address", func() {
			srv = serverMock(ignitionMock)
			err := verifyCIDR("http://127.0.0.1:1234")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Invalid URL", func() {
			err := verifyCIDR("http://invalid:1234")
			Expect(err).To(HaveOccurred())
		})

		It("CIDR not suitable", func() {
			err := verifyCIDR("http://example.com:1234")
			Expect(err).To(HaveOccurred())
		})
	})
})

func getRequestStr(url *string, verifyCidr bool) string {
	request := models.APIVipConnectivityRequest{
		URL:        url,
		VerifyCidr: verifyCidr,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return ""
	}
	return string(requestBytes)
}

func serverMock(mock func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc(WorkerIgnitionPath, mock)
	srv := httptest.NewServer(handler)
	return srv
}

func ignitionMock(w http.ResponseWriter, r *http.Request) {
	ignitionConfig, err := FormatNodeIgnitionFile("http://127.0.0.1:1234")
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

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API connectivity check unit tests")
}
