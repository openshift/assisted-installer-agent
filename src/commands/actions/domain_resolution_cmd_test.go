package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		action, err := New(models.StepTypeDomainResolution, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
		Expect(command).To(Equal("podman"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(strings.Join(args, " ")).To(ContainSubstring(param))
		Expect(strings.Join(args, " ")).To(ContainSubstring("domain_resolution"))
	})

	It("domain resolution bad input", func() {
		badParamsCommonTests(models.StepTypeDomainResolution, []string{param})
	})
})
