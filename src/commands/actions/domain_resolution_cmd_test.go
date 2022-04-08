package actions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("domain resolution", func() {
	var param string

	BeforeEach(func() {
		param = "{\"domains\":[{\"domain_name\":\"api.test.test.com\"},{\"domain_name\":\"api-int.test.test.com\"}," +
			"{\"domain_name\":\"console-openshift-console.apps.test.test.com\"}," +
			"{\"domain_name\":\"validateNoWildcardDNS.test.test.com\"}]}"
	})

	It("domain resolution", func() {
		_, err := New(&config.AgentConfig{}, models.StepTypeDomainResolution, []string{param})
		Expect(err).NotTo(HaveOccurred())
	})

	It("domain resolution - bad domain name", func() {
		param = "{\"domains\":[{\"domain_name\":\"aaaaaa\"}]}"
		_, err := New(&config.AgentConfig{}, models.StepTypeDomainResolution, []string{param})
		Expect(err).To(HaveOccurred())

	})

	It("domain resolution - bad domain name with subcommand", func() {
		param = "{\"domains\":[{\"domain_name\":\"api.test.test.com;echo\"}]}"
		_, err := New(&config.AgentConfig{}, models.StepTypeDomainResolution, []string{param})
		Expect(err).To(HaveOccurred())
	})

	It("domain resolution bad input", func() {
		badParamsCommonTests(models.StepTypeDomainResolution, []string{param})
	})
})
