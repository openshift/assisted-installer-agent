package actions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("connectivity check", func() {
	var param string

	BeforeEach(func() {
		param = "[{\"host_id\":\"0d2c4624-70a8-4412-86d5-1562419f4a43\"," +
			"\"nics\":[{\"ip_addresses\":[\"192.168.127.13\"]," +
			"\"mac\":\"02:00:00:df:1c:e8\",\"name\":\"ens3\"}," +
			"{\"ip_addresses\":[\"192.168.145.13\"],\"mac\":\"02:00:00:47:da:a0\",\"name\":\"ens4\"}]}, " +
			"{\"host_id\":\"0478cd23-a7f0-4f6f-9dc7-1ebb74a6547e\",\"nics\":[{\"ip_addresses\":[\"192.168.127.12\"]," +
			"\"mac\":\"02:00:00:ff:17:be\",\"name\":\"ens3\"}," +
			"{\"ip_addresses\":[\"192.168.145.12\"],\"mac\":\"02:00:00:51:bc:a0\",\"name\":\"ens4\"}]}]"
	})

	It("connectivity cmd", func() {
		_, err := New(&config.AgentConfig{}, models.StepTypeConnectivityCheck, []string{param})
		Expect(err).NotTo(HaveOccurred())
	})

	It("connectivity check cmd wrong args number", func() {
		badParamsCommonTests(models.StepTypeConnectivityCheck, []string{param})
	})
})
