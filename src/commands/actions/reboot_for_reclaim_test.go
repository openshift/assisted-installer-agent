package actions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("reboot_for_reclaim", func() {
	It("succeeds when given correct args", func() {
		action, err := New(&config.AgentConfig{}, models.StepTypeRebootForReclaim, []string{"{\"host_fs_mount_dir\":\"/host\"}"})
		Expect(err).NotTo(HaveOccurred())

		args := action.Args()
		command := action.Command()
		Expect(command).To(Equal("systemctl"))
		Expect(args).To(ContainElement("reboot"))
	})

	It("fails when given bad input", func() {
		badParamsCommonTests(models.StepTypeRebootForReclaim, []string{})
	})
})
