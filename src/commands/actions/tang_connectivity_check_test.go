package actions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("tang connectivity check", func() {
	var param string

	BeforeEach(func() {
		param = "{\"tang_servers\":\"[{\\\"thumbprint\\\":\\\"fake_thumbprint1\\\",\\\"url\\\":\\\"http://www.example.com\\\"}]\"}"
	})

	It("tang connectivity cmd", func() {
		_, err := New(&config.AgentConfig{}, models.StepTypeTangConnectivityCheck, []string{param})
		Expect(err).NotTo(HaveOccurred())
	})

	It("tang connectivity wrong args", func() {
		badParamsCommonTests(models.StepTypeTangConnectivityCheck, []string{param})
	})
})
