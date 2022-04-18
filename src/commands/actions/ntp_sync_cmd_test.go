package actions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("ntp sync", func() {
	var param string

	BeforeEach(func() {
		param = "{\"ntp_source\":\"clock.redhat.com\"}"
	})

	It("ntp sync", func() {
		_, err := New(&config.AgentConfig{}, models.StepTypeNtpSynchronizer, []string{param})
		Expect(err).NotTo(HaveOccurred())
	})

	It("bad ntp sync commands", func() {
		badParamsCommonTests(models.StepTypeNtpSynchronizer, []string{param})
	})

	It("bad ntp sync ntp source", func() {
		param = "{\"ntp_source\":\"echo aaa\"}"
		_, err := New(&config.AgentConfig{}, models.StepTypeNtpSynchronizer, []string{param})
		Expect(err).To(HaveOccurred())
	})
})
