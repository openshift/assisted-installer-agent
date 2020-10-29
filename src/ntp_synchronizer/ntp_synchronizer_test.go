package ntp_synchronizer

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
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

	Context("GetNTPSources", func() {
		It("no_sources", func() {
			ntpDependencies.On("Execute", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-c", "sources").Return("", "", 0)

			sources, err := GetNTPSources(ntpDependencies)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*sources).Should(BeEmpty())
		})

		It("timeout", func() {
			ntpDependencies.On("Execute", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-c", "sources").Return("", "", util.TimeoutExitCode)

			sources, err := GetNTPSources(ntpDependencies)
			Expect(err).Should(HaveOccurred())
			Expect(sources).Should(BeNil())
		})

		It("error", func() {
			ntpDependencies.On("Execute", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-c", "sources").Return("", "", -1)

			sources, err := GetNTPSources(ntpDependencies)
			Expect(err).Should(HaveOccurred())
			Expect(sources).Should(BeNil())
		})

		It("one_source", func() {
			name := "162.159.200.1"
			state := "+"
			output := fmt.Sprintf("^,%s,%s,3,10,377,963,0.002488636,0.004916504,0.041325551", state, name)

			ntpDependencies.On("Execute", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-c", "sources").Return(output, "", 0)

			sources, err := GetNTPSources(ntpDependencies)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*sources).Should(HaveLen(1))
			Expect((*sources)[0].SourceName).Should(Equal(name))
			Expect((*sources)[0].SourceState).Should(Equal(convertSourceState(state)))
		})

		It("multiple_sources", func() {
			output := `
				^,+,162.159.200.1,3,10,377,586,-0.009501813,-0.012108720,0.042861138
				^,*,162.159.200.123,3,10,377,444,-0.006190482,-0.008834993,0.039687645
				^,?,2606:4700:f1::1,0,6,0,4294967295,0.000000000,0.000000000,0.000000000
				^,?,2606:4700:f1::123,0,6,0,4294967295,0.000000000,0.000000000,0.000000000
			`

			ntpDependencies.On("Execute", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-c", "sources").Return(output, "", 0)

			sources, err := GetNTPSources(ntpDependencies)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*sources).Should(HaveLen(4))
		})
	})

	Context("NtpSync", func() {
		It("add_server", func() {
			name := "162.159.200.1"
			state := "+"
			output := fmt.Sprintf("^,%s,%s,3,10,377,963,0.002488636,0.004916504,0.041325551", state, name)

			ntpDependencies.On("Execute", "timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-c", "sources").Return(output, "", 0)
			ntpDependencies.On("Execute", "chronyc", "add", "server", name).Return("", "", 0)

			request := &models.NtpSynchronizationRequest{NtpSource: &name}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			stdout, stderr, exitCode := NtpSync(string(b), ntpDependencies, log)

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
