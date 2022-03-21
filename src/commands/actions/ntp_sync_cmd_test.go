package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("ntp sync", func() {
	var param string

	BeforeEach(func() {
		param = "{\"ntp_source\":\"clock.redhat.com\"}"
	})

	It("ntp sync", func() {
		action, err := New(models.StepTypeNtpSynchronizer, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.Run()
		Expect(command).To(Equal("podman"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
			"/usr/bin/chronyc",
			"/var/run/chrony",
		}

		verifyPaths(strings.Join(args, " "), paths)
		Expect(args[len(args)-1]).To(ContainSubstring(param))
	})

	It("ntp sync", func() {
		badParamsCommonTests(models.StepTypeNtpSynchronizer, []string{param})
	})
})