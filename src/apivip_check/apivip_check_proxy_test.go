package apivip_check

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

const proxyTestHostname = "ignition.invalid"

var _ = Describe("HTTPS proxy with custom CA", func() {
	for _, test := range []struct {
		name     string
		scenario string
	}{
		{"downloads ignition through HTTPS_PROXY", "proxy"},
		{"bypasses HTTPS_PROXY when NO_PROXY matches", "no_proxy"},
	} {
		It(test.name, func() {
			runProxyTestScenario(test.scenario)
		})
	}
})

func runProxyTestScenario(scenario string) {
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

	noProxy := ""
	if scenario == "no_proxy" {
		noProxy = proxyTestHostname
	}
	restoreEnv := useProxyEnv(proxy.URL, noProxy)
	defer restoreEnv()

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

// useProxyEnv restores the environment after each spec. Ginkgo v1's Setenv is a no-op.
func useProxyEnv(httpsProxy, noProxy string) func() {
	keys := []string{"HTTPS_PROXY", "https_proxy", "NO_PROXY", "no_proxy"}
	type previousValue struct {
		value string
		set   bool
	}
	previous := make(map[string]previousValue, len(keys))
	for _, key := range keys {
		value, set := os.LookupEnv(key)
		previous[key] = previousValue{value: value, set: set}
	}

	Expect(os.Setenv("HTTPS_PROXY", httpsProxy)).To(Succeed())
	Expect(os.Setenv("https_proxy", "")).To(Succeed())
	Expect(os.Setenv("NO_PROXY", noProxy)).To(Succeed())
	Expect(os.Setenv("no_proxy", "")).To(Succeed())

	return func() {
		var restoreErrors []error
		for _, key := range keys {
			var err error
			if value := previous[key]; value.set {
				err = os.Setenv(key, value.value)
			} else {
				err = os.Unsetenv(key)
			}
			if err != nil {
				restoreErrors = append(restoreErrors, fmt.Errorf("restore %s: %w", key, err))
			}
		}
		Expect(errors.Join(restoreErrors...)).To(Succeed())
	}
}
