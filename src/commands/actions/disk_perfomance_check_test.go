package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("dhcp leases", func() {
	var param string
	var timeout string

	BeforeEach(func() {
		param = "{\"path\":\"/dev/disk/by-path/pci-0000:00:06.0\"}"
		timeout = "5.25"
	})

	It("disk performance", func() {
		action, err := New(models.StepTypeInstallationDiskSpeedCheck, []string{param, timeout})
		Expect(err).NotTo(HaveOccurred())

		args := action.Args()
		command := action.Command()
		Expect(command).To(Equal("sh"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
			"/dev",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(args[len(args)-1]).To(ContainSubstring(param))
		Expect(args[len(args)-1]).To(ContainSubstring(timeout))

	})

	It("disk performance input failures", func() {
		By("bad model")
		_, err := New(models.StepTypeInstallationDiskSpeedCheck, []string{"echo aaaa", timeout})
		Expect(err).To(HaveOccurred())

		By("bad timeout")
		_, err = New(models.StepTypeInstallationDiskSpeedCheck, []string{param, "aaaaa"})
		Expect(err).To(HaveOccurred())

		By("One arg")
		_, err = New(models.StepTypeInstallationDiskSpeedCheck, []string{param})
		Expect(err).To(HaveOccurred())

		By("Three args")
		_, err = New(models.StepTypeInstallationDiskSpeedCheck, []string{param, timeout, "aaa"})
		Expect(err).To(HaveOccurred())
	})
})
