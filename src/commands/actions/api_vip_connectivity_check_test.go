package actions

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

func TestActions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "actions tests")
}

func badParamsCommonTests(stepType models.StepType, params []string) {
	By("bad command")
	_, err := New(stepType, []string{"echo aaaa"})
	Expect(err).To(HaveOccurred())

	if len(params) > 0 {
		By("Less then 1")
		_, err = New(stepType, []string{})
		Expect(err).To(HaveOccurred())
	}

	By("More then expected")
	_, err = New(stepType, append(params, "aaaa"))
	Expect(err).To(HaveOccurred())
}

var _ = Describe("api connectivity check", func() {
	var param string

	BeforeEach(func() {
		param = "{\"url\":\"http://test.com:22624/config/worker\"}"
	})

	It("api connectivity cmd", func() {
		action, err := New(models.StepTypeAPIVipConnectivityCheck, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
		Expect(command).To(Equal(podman))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(args[len(args)-1]).To(ContainSubstring(param))
	})

	It("api connectivity wrong args", func() {
		badParamsCommonTests(models.StepTypeAPIVipConnectivityCheck, []string{param})
	})
})