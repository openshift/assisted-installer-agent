package actions

import (
	"strings"

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
		action, err := New(&config.AgentConfig{}, models.StepTypeNtpSynchronizer, []string{param})
		Expect(err).NotTo(HaveOccurred())

		args := action.Args()
		command := action.Command()
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

	It("bad ntp sync commands", func() {
		badParamsCommonTests(models.StepTypeNtpSynchronizer, []string{param})
	})

	It("bad ntp sync ntp source", func() {
		param = "{\"ntp_source\":\"echo aaa\"}"
		_, err := New(&config.AgentConfig{}, models.StepTypeNtpSynchronizer, []string{param})
		Expect(err).To(HaveOccurred())
	})
})
