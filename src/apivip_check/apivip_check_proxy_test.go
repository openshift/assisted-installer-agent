package apivip_check

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

const proxyTestScenarioEnv = "ASSISTED_INSTALLER_AGENT_PROXY_TEST_SCENARIO"
const proxyTestHostname = "ignition.invalid"

var _ = Describe("HTTPS proxy with custom CA", func() {
	for _, test := range []struct {
		name     string
		scenario string
	}{
		{"downloads ignition through HTTPS_PROXY", "proxy"},
		{"bypasses HTTPS_PROXY when NO_PROXY matches", "no_proxy"},
	} {
		test := test
		It(test.name, func() {
			runProxyTestScenario(test.scenario, test.name)
		})
	}
})

func runProxyTestScenario(scenario, specName string) {
	if os.Getenv(proxyTestScenarioEnv) == "" {
		// ProxyFromEnvironment reads the environment once per process. Run each
		// scenario in a fresh process with only its Ginkgo spec selected.
		//nolint:gosec // G204: re-execute the running test binary with fixed scenario names.
		cmd := exec.Command(os.Args[0], "-test.run=^TestSubsystem$", "-test.timeout=1m", "-ginkgo.focus="+regexp.QuoteMeta(specName)+"$")
		cmd.Env = append(os.Environ(), proxyTestScenarioEnv+"="+scenario)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "proxy scenario %q failed:\n%s", scenario, output)
		return
	}
	Expect(os.Getenv(proxyTestScenarioEnv)).To(Equal(scenario))

	serverConfig, caPEM, err := certsetup()
	Expect(err).NotTo(HaveOccurred())
	server, err := httpsServerMock(serverConfig, ignitionMock)
	Expect(err).NotTo(HaveOccurred())
	defer server.Close()

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	Expect(err).NotTo(HaveOccurred())
	targetHost := net.JoinHostPort(proxyTestHostname, port)
	targetURL := "https://" + targetHost

	var connectCount atomic.Int32
	var proxyError atomic.Value
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectCount.Add(1)
		if r.Method != http.MethodConnect || r.Host != targetHost {
			proxyError.Store(fmt.Sprintf("unexpected proxy request: %s %s", r.Method, r.Host))
			http.Error(w, "unexpected proxy request", http.StatusBadRequest)
			return
		}

		upstream, err := net.Dial("tcp", server.Listener.Addr().String())
		if err != nil {
			proxyError.Store(fmt.Sprintf("connect to ignition server: %v", err))
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		client, buffered, err := w.(http.Hijacker).Hijack()
		if err != nil {
			upstream.Close()
			proxyError.Store(fmt.Sprintf("hijack proxy connection: %v", err))
			return
		}
		if _, err := fmt.Fprint(buffered, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			client.Close()
			upstream.Close()
			return
		}
		if err := buffered.Flush(); err != nil {
			client.Close()
			upstream.Close()
			return
		}

		go func() {
			_, _ = io.Copy(upstream, buffered)
			upstream.Close()
			client.Close()
		}()
		_, _ = io.Copy(client, upstream)
		client.Close()
		upstream.Close()
	}))
	defer proxy.Close()

	// Ginkgo v1's GinkgoT().Setenv is a no-op; these changes stay in this child process.
	Expect(os.Setenv("HTTPS_PROXY", proxy.URL)).To(Succeed())
	Expect(os.Setenv("https_proxy", "")).To(Succeed())
	Expect(os.Setenv("NO_PROXY", "")).To(Succeed())
	Expect(os.Setenv("no_proxy", "")).To(Succeed())
	if scenario == "no_proxy" {
		Expect(os.Setenv("NO_PROXY", proxyTestHostname)).To(Succeed())
	}

	encodedCA := base64.StdEncoding.EncodeToString(caPEM)
	stdout, stderr, exitCode := CheckAPIConnectivity(getRequestStr(&targetURL, false, &encodedCA, nil), logrus.New())
	Expect(proxyError.Load()).To(BeNil())
	var response models.APIVipConnectivityResponse
	Expect(json.Unmarshal([]byte(stdout), &response)).To(Succeed(), "invalid response %q", stdout)
	Expect(exitCode).To(BeZero())

	switch scenario {
	case "proxy":
		Expect(response.IsSuccess).To(BeTrue(), "ignition download through proxy failed: %s", response.DownloadError)
		Expect(stderr).To(BeEmpty())
		Expect(connectCount.Load()).To(Equal(int32(1)))
	case "no_proxy":
		Expect(response.IsSuccess).To(BeFalse())
		Expect(response.DownloadError).To(ContainSubstring("request failed"))
		Expect(connectCount.Load()).To(BeZero(), "NO_PROXY should bypass the proxy")
	}
}
