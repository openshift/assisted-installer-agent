package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("free addresses", func() {
	var param string

	BeforeEach(func() {
		param = "[\"192.168.127.0/24\",\"192.168.145.0/24\"]"
	})

	It("free addresses", func() {
		action, err := New(models.StepTypeFreeNetworkAddresses, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
		Expect(command).To(Equal("sh"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(args[len(args)-1]).To(ContainSubstring(param))
	})

	It("free addresses wrong args number", func() {
		badParamsCommonTests(models.StepTypeFreeNetworkAddresses, []string{param})

		By("Bad model object")
		param = "[\"192.168.127.0/24\",\"rm -f\"]"
		_, err := New(models.StepTypeFreeNetworkAddresses, []string{param})
		Expect(err).To(HaveOccurred())

	})
})