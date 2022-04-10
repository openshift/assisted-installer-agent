package actions

import (
	"strings"

	"github.com/jinzhu/copier"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("Logs gather test", func() {
	var param string
	var oldConfig config.ConnectivityConfig
	var agentConfig *config.AgentConfig

	BeforeEach(func() {
		param = "{\"bootstrap\":true,\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"192.168.127.12\"]}"
		agentConfig = &config.AgentConfig{}
	})

	It("Logs gather bootstrap", func() {
		Expect(copier.Copy(&oldConfig, &agentConfig.ConnectivityConfig)).To(BeNil())
		agentConfig.InsecureConnection = true
		action, err := New(agentConfig, models.StepTypeLogsGather, []string{param})
		Expect(err).NotTo(HaveOccurred())

		args := action.Args()
		command := action.Command()
		Expect(command).To(Equal("timeout"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
			"/etc/pki",
			"/root/.ssh",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(strings.Join(args, " ")).To(ContainSubstring("--env PULL_SECRET_TOKEN --name logs-sender --pid=host"))
		Expect(strings.Join(args, " ")).To(ContainSubstring("-masters-ips=192.168.127.10,192.168.127.12"))
		Expect(strings.Join(args, " ")).To(ContainSubstring("-bootstrap=true -with-installer-gather-logging=true"))
	})
	AfterEach(func() {
		Expect(copier.Copy(&agentConfig.ConnectivityConfig, &oldConfig)).To(BeNil())
	})

	It("Logs gather ca cert", func() {
		param = "{\"bootstrap\":true,\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"192.168.127.12\"]}"

		agentConfig.CACertificatePath = "/ca_cert"
		action, err := New(agentConfig, models.StepTypeLogsGather, []string{param})
		Expect(err).NotTo(HaveOccurred())

		args := action.Args()
		command := action.Command()
		Expect(command).To(Equal("timeout"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
			"/etc/pki",
			"/root/.ssh",
			"/ca_cert",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(strings.Join(args, " ")).To(ContainSubstring("--cacert /ca_cert"))
	})

	It("Logs gather", func() {
		badParamsCommonTests(models.StepTypeLogsGather, []string{param})

		By("Bad Uuid")
		param = "{\"bootstrap\":true,\"cluster_id\":\"bad\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"192.168.127.12\"]}"
		_, err := New(agentConfig, models.StepTypeLogsGather, []string{param})
		Expect(err).To(HaveOccurred())

		By("Bad Ip")
		param = "{\"bootstrap\":true,\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"echo\"]}"
		_, err = New(agentConfig, models.StepTypeLogsGather, []string{param})
		Expect(err).To(HaveOccurred())

		By("Bad boolean")
		param = "{\"bootstrap\":\"echo\",\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\"]}"
		_, err = New(agentConfig, models.StepTypeLogsGather, []string{param})
		Expect(err).To(HaveOccurred())
	})
})
