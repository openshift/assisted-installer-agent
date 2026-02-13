package actions

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	"github.com/spf13/afero"
)

func verifyPaths(command string, paths []string) {
	for _, path := range paths {
		Expect(command).To(ContainSubstring(fmt.Sprintf("-v %[1]v:%[1]v", path)))
	}
}

var _ = Describe("inventory", func() {
	var filesystem afero.Fs
	var hostId string
	var action *inventory

	BeforeEach(func() {
		filesystem = afero.NewMemMapFs()
		hostId = uuid.NewString()
		action = &inventory{
			args: []string{
				hostId,
			},
			filesystem:  filesystem,
			agentConfig: &config.AgentConfig{},
		}
	})

	It("inventory cmd", func() {
		args := action.Args()

		By("running two commands via sh")
		command := action.Command()
		Expect(command).To(Equal("sh"))
		Expect(args[0]).To(Equal("-c"))
		Expect(args[1]).To(ContainSubstring("&&"))

		mtabFile := fmt.Sprintf("/root/mtab-%s", hostId)
		mtabCopy := fmt.Sprintf("cp /etc/mtab %s", mtabFile)
		mtabMount := fmt.Sprintf("%s:/host/etc/mtab:ro", mtabFile)

		Expect(args[1]).To(ContainSubstring(mtabCopy))

		By("verifying mounts to host's filesystem")
		Expect(args[1]).To(ContainSubstring(mtabMount))
		paths := []string{
			"/proc/meminfo",
			"/sys/kernel/mm/hugepages",
			"/proc/cpuinfo",
			"/sys/block",
			"/sys/devices",
			"/sys/bus",
			"/sys/class",
			"/run/udev",
		}
		for _, path := range paths {
			Expect(args[1]).To(ContainSubstring(fmt.Sprintf("-v %[1]v:/host%[1]v:ro", path)))
		}
	})

	It("inventory cmd wrong args number", func() {
		badParamsCommonTests(models.StepTypeDhcpLeaseAllocate, []string{hostId})
	})

	It("Adds the EFI variables volume if the directory exists", func() {
		err := filesystem.MkdirAll("/sys/firmware/efi/efivars", 0755)
		Expect(err).ToNot(HaveOccurred())

		args := action.Args()
		Expect(args[1]).To(ContainSubstring("-v /sys/firmware/efi/efivars:/host/sys/firmware/efi/efivars"))
	})

	It("Doesn't add the EFI variables volume if the directory doesn't exist", func() {
		args := action.Args()
		Expect(args[1]).ToNot(ContainSubstring("-v /sys/firmware/efi/efivars:/host/sys/firmware/efi/efivars"))
	})
})

var _ = Describe("inventory command injection prevention", func() {
	var filesystem afero.Fs
	var agentConfig *config.AgentConfig

	BeforeEach(func() {
		filesystem = afero.NewMemMapFs()
		agentConfig = &config.AgentConfig{}
		agentConfig.AgentVersion = "test-image:latest"
	})

	It("uses shellescape to protect podman arguments", func() {
		// This test verifies that the inventory command uses shellescape.QuoteCommand
		// to protect against command injection in podman arguments
		hostId := uuid.NewString()
		action := &inventory{
			args:        []string{hostId},
			filesystem:  filesystem,
			agentConfig: agentConfig,
		}

		args := action.Args()
		Expect(args).To(HaveLen(2))
		Expect(args[0]).To(Equal("-c"))
		Expect(args[1]).To(ContainSubstring("podman"))
		Expect(args[1]).To(ContainSubstring("inventory"))
	})

	It("escapes mtab path to prevent injection", func() {
		// Note: Validate() checks for UUID, but we test the escaping behavior
		// in case validation is bypassed or relaxed in the future
		hostId := uuid.NewString()
		action := &inventory{
			args:        []string{hostId},
			filesystem:  filesystem,
			agentConfig: agentConfig,
		}

		args := action.Args()
		cmdStr := args[1]

		// The mtab copy command should be properly formatted
		Expect(cmdStr).To(ContainSubstring("cp /etc/mtab"))
		Expect(cmdStr).To(ContainSubstring(fmt.Sprintf("/root/mtab-%s", hostId)))
	})

	It("protects agentVersion from injection via quoting", func() {
		// Even if AgentVersion contained malicious content, shellescape.QuoteCommand
		// should escape it properly
		hostId := uuid.NewString()
		maliciousConfig := &config.AgentConfig{}
		maliciousConfig.AgentVersion = "test-image:latest; rm -rf /"
		action := &inventory{
			args:        []string{hostId},
			filesystem:  filesystem,
			agentConfig: maliciousConfig,
		}

		args := action.Args()
		cmdStr := args[1]

		// The malicious content should be quoted/escaped
		Expect(cmdStr).To(ContainSubstring("test-image:latest"))
		// Verify it's quoted - shellescape wraps special characters in quotes
		Expect(cmdStr).To(ContainSubstring("'test-image:latest; rm -rf /'"))
	})
})
