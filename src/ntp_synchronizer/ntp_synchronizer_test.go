package ntp_synchronizer

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var _ = Describe("NTP synchronizer", func() {
	var (
		ntpDependencies *MockNtpSynchronizerDependencies
		log             *logrus.Logger
	)

	BeforeEach(func() {
		ntpDependencies = &MockNtpSynchronizerDependencies{}
		log = logrus.New()
	})

	AfterEach(func() {
		ntpDependencies.AssertExpectations(GinkgoT())
	})

	Context("getNTPSources", func() {
		It("no_sources", func() {
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return("", "", 0)

			sources, err := getNTPSources(ntpDependencies)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(sources).Should(BeEmpty())
		})

		It("timeout", func() {
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return("", "", util.TimeoutExitCode)

			sources, err := getNTPSources(ntpDependencies)
			Expect(err).Should(HaveOccurred())
			Expect(sources).Should(BeNil())
		})

		It("error", func() {
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return("", "", -1)

			sources, err := getNTPSources(ntpDependencies)
			Expect(err).Should(HaveOccurred())
			Expect(sources).Should(BeNil())
		})

		It("one_source", func() {
			name := "162.159.200.1"
			state := "+"
			output := fmt.Sprintf("^%s %s     2  10   377   268    +12ms[  +12ms] +/-  132ms", state, name)

			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return(output, "", 0)

			sources, err := getNTPSources(ntpDependencies)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(sources).Should(HaveLen(1))
			Expect(sources[0].SourceName).Should(Equal(name))
			Expect(sources[0].SourceState).Should(Equal(convertSourceState(state)))
		})

		It("multiple_sources", func() {
			output := `
				MS Name/IP address         Stratum Poll Reach LastRx Last sample               
				===============================================================================
				^+ ntpool1.603.newcontinuum>     2  10   377   268    +12ms[  +12ms] +/-  132ms
				^* eterna.binary.net             2  10   377   290  +2641us[+2352us] +/-  118ms
				^? time.cloudflare.com           0   6     0     -     +0ns[   +0ns] +/-    0ns
				^? time.cloudflare.com           0   6     0     -     +0ns[   +0ns] +/-    0ns
			`

			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return(output, "", 0)

			sources, err := getNTPSources(ntpDependencies)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(sources).Should(HaveLen(4))
		})
	})

	Context("isServerConfigured", func() {
		BeforeEach(func() {
			output := `
			210 Number of sources = 11
			MS Name/IP address         Stratum Poll Reach LastRx Last sample               
			===============================================================================
			^? 64.22.253.155                 0  10     0     -     +0ns[   +0ns] +/-    0ns
			^? 50.205.244.20                 0  10     0     -     +0ns[   +0ns] +/-    0ns
			^? 69.10.161.7                   0  10     0     -     +0ns[   +0ns] +/-    0ns
			^? 209.50.63.74                  0  10     0     -     +0ns[   +0ns] +/-    0ns
			^- 10.5.26.10                    1   9   377  1147   +148us[ +150us] +/-   24ms
			^- 10.2.32.38                    1  10   377   317   +147us[ +150us] +/- 3754us
			^* 10.11.160.238                 1  10   377   261  +5240ns[+7909ns] +/-  492us
			^- 10.5.27.10                    1   9   377   704   +160us[ +163us] +/-   24ms
			^- 10.18.52.10                   2  10   377   123    +21us[  +21us] +/-   30ms
			^- 10.2.32.37                    1  10   377   684   +594us[ +596us] +/- 4295us
			^- 10.18.100.10                  2  10   377   557    +79us[  +81us] +/-   45ms
		`

			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return(output, "", 0)
		})

		It("unknown_server", func() {
			server := "unknown.server.com"
			ntpDependencies.On("LookupHost", server).Return([]string{}, errors.Errorf("Unknown server"))

			configured, err := isServerConfigured(ntpDependencies, server)
			Expect(err).Should(HaveOccurred())
			Expect(configured).Should(BeFalse())
		})

		It("not_configured", func() {
			server := "1.1.1.1"
			ntpDependencies.On("LookupHost", server).Return([]string{}, nil)

			configured, err := isServerConfigured(ntpDependencies, server)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(configured).Should(BeFalse())
		})

		It("configured_reverse_lookup", func() {
			server := "clock.redhat.com"
			ntpDependencies.On("LookupHost", server).Return([]string{"10.5.27.10"}, nil)

			configured, err := isServerConfigured(ntpDependencies, server)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(configured).Should(BeTrue())
		})
	})

	Context("Run", func() {
		It("add_new_server", func() {
			name := "162.159.200.1"
			state := "+"
			output := fmt.Sprintf("^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns", state, name)

			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return("", "", 0).Once()
			ntpDependencies.On("LookupHost", name).Return([]string{}, nil).Once()
			ntpDependencies.On("ExecutePrivileged", "chronyc", "add", "server", name, "iburst").Return("", "", 0).Once()
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return(output, "", 0).Once()
			ntpDependencies.On("LookupAddr", name).Return([]string{}, errors.Errorf("error")).Once()

			request := &models.NtpSynchronizationRequest{NtpSource: &name}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			stdout, stderr, exitCode := Run(string(b), ntpDependencies, log)

			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			var response models.NtpSynchronizationResponse

			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())
			Expect(response.NtpSources).Should(HaveLen(1))
			Expect(response.NtpSources[0].SourceName).Should(Equal(name))
			Expect(response.NtpSources[0].SourceState).Should(Equal(convertSourceState(state)))
		})

		It("add_new_server_with_reverse_lookup", func() {
			name := "162.159.200.1"
			resolved_name := "time.cloudflare.com"
			state := "+"
			output := fmt.Sprintf("^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns", state, name)

			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return("", "", 0).Once()
			ntpDependencies.On("LookupHost", name).Return([]string{}, nil).Once()
			ntpDependencies.On("ExecutePrivileged", "chronyc", "add", "server", name, "iburst").Return("", "", 0).Once()
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return(output, "", 0).Once()
			ntpDependencies.On("LookupAddr", name).Return([]string{fmt.Sprintf("%s.", resolved_name)}, nil).Once()

			request := &models.NtpSynchronizationRequest{NtpSource: &name}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			stdout, stderr, exitCode := Run(string(b), ntpDependencies, log)

			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			var response models.NtpSynchronizationResponse

			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())
			Expect(response.NtpSources).Should(HaveLen(1))
			Expect(response.NtpSources[0].SourceName).Should(Equal(resolved_name))
			Expect(response.NtpSources[0].SourceState).Should(Equal(convertSourceState(state)))
		})

		It("add_multiple_servers", func() {
			names := []string{"162.159.200.1", "162.159.200.2", "162.159.200.3"}
			name := strings.Join(names, ",")
			state := "+"
			output := fmt.Sprintf(`
			210 Number of sources = %d
			MS Name/IP address         Stratum Poll Reach LastRx Last sample
			===============================================================================
			^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns
			^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns
			^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns
		`, len(names), state, names[0], state, names[1], state, names[2])

			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-n", "sources").Return("", "", 0).Times(len(names))

			for _, currName := range names {
				ntpDependencies.On("LookupHost", currName).Return([]string{}, nil).Once()
				ntpDependencies.On("ExecutePrivileged", "chronyc", "add", "server", currName, "iburst").Return("", "", 0).Once()
				ntpDependencies.On("LookupAddr", currName).Return([]string{}, errors.Errorf("error")).Once()
			}

			// Last call
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-n", "sources").Return(output, "", 0).Once()

			request := &models.NtpSynchronizationRequest{NtpSource: &name}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			stdout, stderr, exitCode := Run(string(b), ntpDependencies, log)

			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			var response models.NtpSynchronizationResponse

			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())
			Expect(response.NtpSources).Should(HaveLen(len(names)))

			for _, currentNtpSource := range response.NtpSources {
				Expect(names).Should(ContainElement(currentNtpSource.SourceName))
				Expect(currentNtpSource.SourceState).Should(Equal(convertSourceState(state)))
			}
		})

		It("add_existing_server", func() {
			name := "162.159.200.1"
			state := "+"
			output := fmt.Sprintf("^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns", state, name)

			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return(output, "", 0).Times(2)
			ntpDependencies.On("LookupAddr", name).Return([]string{}, errors.Errorf("error")).Once()

			request := &models.NtpSynchronizationRequest{NtpSource: &name}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			stdout, stderr, exitCode := Run(string(b), ntpDependencies, log)

			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			var response models.NtpSynchronizationResponse

			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())
			Expect(response.NtpSources).Should(HaveLen(1))
			Expect(response.NtpSources[0].SourceName).Should(Equal(name))
			Expect(response.NtpSources[0].SourceState).Should(Equal(convertSourceState(state)))
		})

		It("add_existing_pool", func() {
			poolName := "pool.cloud.com"
			serverName := "server.cloud.com"
			state := "+"
			output := fmt.Sprintf("^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns", state, serverName)

			ntpDependencies.On("LookupHost", poolName).Return([]string{serverName}, nil).Once()
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources").Return(output, "", 0).Times(2)
			ntpDependencies.On("LookupAddr", serverName).Return([]string{}, errors.Errorf("error")).Once()

			request := &models.NtpSynchronizationRequest{NtpSource: &poolName}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			stdout, stderr, exitCode := Run(string(b), ntpDependencies, log)

			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			var response models.NtpSynchronizationResponse

			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())
			Expect(response.NtpSources).ShouldNot(BeEmpty())
			Expect(response.NtpSources[0].SourceName).Should(Equal(serverName))
			Expect(response.NtpSources[0].SourceState).Should(Equal(convertSourceState(state)))
		})

		It("no_dns_resolvement", func() {
			name := "162.159.200.1"
			state := "+"
			output := fmt.Sprintf("^%s %s           0   6     0     -     +0ns[   +0ns] +/-    0ns", state, name)

			ntpDependencies.On("ExecutePrivileged", "chronyc", "add", "server", name, "iburst").Return("", "", 0).Once()
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-n", "sources").Return("", "", 0).Once()
			ntpDependencies.On("LookupHost", name).Return([]string{}, errors.New("error")).Once()
			ntpDependencies.On("ExecutePrivileged", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-n", "sources").Return(output, "", 0).Once()
			ntpDependencies.On("LookupAddr", name).Return([]string{}, errors.Errorf("error")).Once()

			request := &models.NtpSynchronizationRequest{NtpSource: &name}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			stdout, stderr, exitCode := Run(string(b), ntpDependencies, log)

			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			var response models.NtpSynchronizationResponse

			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())
			Expect(response.NtpSources).ShouldNot(BeEmpty())
			Expect(response.NtpSources[0].SourceName).Should(Equal(name))
			Expect(response.NtpSources[0].SourceState).Should(Equal(convertSourceState(state)))
		})
	})
})

func TestUnitests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NTP unit tests")
}
