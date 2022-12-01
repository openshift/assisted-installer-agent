package tang_connectivity_check

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/go-openapi/swag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ClientMock struct {}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), "127.0.0.1") {
		client := &http.Client{}
		return client.Do(req)
	}

	if strings.Contains(req.URL.String(), "www.example.com") {
		return &http.Response{StatusCode: 404}, nil
	}
	return nil, errors.New("test client error, unexpected URL")
}

var _ = Describe("Tang connectivity check test", func() {
	var log logrus.FieldLogger
	var srv *httptest.Server
	testClient := &ClientMock{}

	BeforeEach(func() {
		log = logrus.New()
	})

	AfterEach(func() {
		srv.Close()
	})

	Context("Tang connectivity", func() {
		It("successful connection", func() {
			srv = serverMock(tangServerMock)
			tServers := []types.Tang{{URL: srv.URL, Thumbprint: swag.String("fake_thumbprint1")}}

			stdout, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)
			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(BeEmpty())
			checkTangResponse(stdout, true)
		})

		It("multiple tang servers", func() {
			srv1 := serverMock(tangServerMock)
			srv2 := serverMock(tangServerMock)

			tServers := []types.Tang{
				{URL: srv1.URL, Thumbprint: swag.String("fake_thumbprint1")},
				{URL: srv2.URL, Thumbprint: swag.String("fake_thumbprint2")},
			}

			stdout, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)

			Expect(exitCode).Should(Equal(0))
			Expect(stderr).Should(BeEmpty())
			checkTangResponse(stdout, true)
		})

		It("missing thumbprint", func() {
			srv = serverMock(tangServerMock)
			tServers := []types.Tang{{URL: srv.URL, Thumbprint: swag.String("")}}

			_, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(ContainSubstring("Tang thumbprint isn't set for server"))
		})

		It("missing tang URL", func() {
			srv = serverMock(tangServerMock)
			tServers := []types.Tang{{URL: "", Thumbprint: swag.String("fake_thumbprint1")}}

			_, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(ContainSubstring("empty url"))
		})

		It("invalid tang URL", func() {
			srv = serverMock(tangServerMock)
			tServers := []types.Tang{{URL: "foo", Thumbprint: swag.String("fake_thumbprint1")}}

			_, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(ContainSubstring("invalid URI for request"))
		})

		It("tang server not available", func() {
			srv = serverMock(tangServerMock)
			tServers := []types.Tang{{URL: "http://www.example.com", Thumbprint: swag.String("fake_thumbprint1")}}

			_, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(ContainSubstring("HTTP GET failure. Status Code: 404"))
		})

		It("multiple tang servers - one not available", func() {
			srv = serverMock(tangServerMock)
			tServers := []types.Tang{
				{URL: srv.URL, Thumbprint: swag.String("fake_thumbprint1")},
				{URL: "http://www.example.com", Thumbprint: swag.String("fake_thumbprint2")},
			}

			_, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(ContainSubstring("HTTP GET failure. Status Code: 404"))
		})

		It("multiple tang servers - one not valid URL", func() {
			srv = serverMock(tangServerMock)
			tServers := []types.Tang{
				{URL: srv.URL, Thumbprint: swag.String("fake_thumbprint1")},
				{URL: "foo", Thumbprint: swag.String("fake_thumbprint2")},
			}

			_, stderr, exitCode := CheckTangConnectivity(getRequestStr(tServers), log, testClient)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(ContainSubstring("invalid URI for request"))
		})

		It("invalid request format", func() {
			srv = serverMock(tangServerMock)
			_, stderr, exitCode := CheckTangConnectivity("some invalid request", log, testClient)
			Expect(exitCode).Should(Equal(-1))
			Expect(stderr).Should(ContainSubstring("Error unmarshaling TangConnectivityRequest"))
		})
	})
})

func getRequestStr(tServers []types.Tang) string {
	res, err := json.Marshal(tServers)
	if err != nil {
		return ""
	}
	resStr := string(res)
	request := models.TangConnectivityRequest{
		TangServers: &resStr,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return ""
	}
	return string(requestBytes)
}

func serverMock(mock func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc(TangKeysPath, mock)
	srv := httptest.NewServer(handler)
	return srv
}

func tangServerMock(w http.ResponseWriter, r *http.Request) {
	url := "http://127.0.0.1:7500"
	configBytes, err := json.Marshal(getTangResponse(url))
	if err != nil {
		logrus.Error("failed to marshal key to json")
		return
	}
	_, _ = w.Write(configBytes)
}

func getTangResponse(url string) models.TangServerResponse {
	return models.TangServerResponse{
		TangURL: url,
		Payload: "some_fake_payload",
		Signatures: []*models.TangServerSignatures{
			{
				Signature: "some_fake_signature1",
				Protected: "foobar1",
			},
			{
				Signature: "some_fake_signature2",
				Protected: "foobar2",
			},
		},
	}
}

func checkTangResponse(stdout string, success bool) {
	var response models.TangConnectivityResponse

	Expect(json.Unmarshal([]byte(stdout), &response)).ToNot(HaveOccurred())
	Expect(success).To(Equal(response.IsSuccess))
	if success {
		Expect(len(response.TangServerResponse)).ToNot(Equal(0))
		for _, res := range response.TangServerResponse {
			Expect(res.Payload).NotTo(BeEmpty())
			Expect(res.TangURL).NotTo(BeEmpty())
			Expect(res.Signatures).NotTo(BeEmpty())
			Expect(len(res.Signatures)).ToNot(Equal(0))
			for _, sig := range res.Signatures {
				Expect(sig.Signature).NotTo(BeEmpty())
				Expect(sig.Protected).NotTo(BeEmpty())
			}
		}
	}
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tang connectivity check unit tests")
}
