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

	Context("extractCmdlineParams", func() {
		It("returns the expected parameters when valid cmdline output is given", func() {
			cmdlineOutput := "ip=192.168.1.1 nameserver=8.8.8.8 rd.znet=enabled zfcp.allow_lun_scan=0 rd.zfcp=xyz rd.dasd=abc"
			paramsToExtract := []string{"ip", "nameserver", "rd.znet", "zfcp.allow_lun_scan", "rd.zfcp", "rd.dasd"}

			expectedCmdline := "ip=192.168.1.1 nameserver=8.8.8.8 rd.znet=enabled zfcp.allow_lun_scan=0 rd.zfcp=xyz rd.dasd=abc "
			requiredCmdline := extractCmdlineParams(cmdlineOutput, paramsToExtract)

			Expect(requiredCmdline).To(Equal(expectedCmdline))
		})

		It("returns an empty string when no parameters match", func() {
			cmdlineOutput := "other_param=value"
			paramsToExtract := []string{"ip", "nameserver"}

			requiredCmdline := extractCmdlineParams(cmdlineOutput, paramsToExtract)

			Expect(requiredCmdline).To(Equal(""))
		})
	})
})
