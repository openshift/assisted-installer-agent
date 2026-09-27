package actions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("vips verifier", func() {
	var param string

	BeforeEach(func() {
		param = `[
			{
				"vip": "1.2.3.4",
				"vip_type": "api"
			},
			{
				"vip": "1.2.3.5",
				"vip_type": "ingress"
			},
			{
				"vip": "ff::1",
				"vip_type": "api"
			},
			{
				"vip": "ff::2",
				"vip_type": "ingress"
			}
        ]`
	})

	It("vip verifier cmd", func() {
		_, err := New(&config.AgentConfig{}, models.StepTypeVerifyVips, []string{param})
		Expect(err).NotTo(HaveOccurred())
	})

	It("vip verifier cmd wrong args number", func() {
		badParamsCommonTests(models.StepTypeVerifyVips, []string{param})
	})
})
