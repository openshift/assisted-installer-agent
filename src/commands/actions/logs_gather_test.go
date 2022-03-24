package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("Logs gather test", func() {
	var param string

	BeforeEach(func() {
		param = "{\"base_url\":\"http://10.1.178.26:6000\",\"bootstrap\":true,\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"insecure\":true,\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"192.168.127.12\"]}"
	})

	It("Logs gather bootstrap", func() {
		action, err := New(models.StepTypeLogsGather, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
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

	It("Logs gather ca cert", func() {
		param = "{\"base_url\":\"http://10.1.178.26:6000\",\"bootstrap\":true,\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"insecure\":true,\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"192.168.127.12\"], \"ca_cert_path\": \"/ca_cert\"}"

		action, err := New(models.StepTypeLogsGather, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
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
		param = "{\"base_url\":\"http://10.1.178.26:6000\",\"bootstrap\":true,\"cluster_id\":\"bad\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"insecure\":true,\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"192.168.127.12\"]}"
		_, err := New(models.StepTypeLogsGather, []string{param})
		Expect(err).To(HaveOccurred())

		By("Bad Ip")
		param = "{\"base_url\":\"http://10.1.178.26:6000\",\"bootstrap\":true,\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"insecure\":true,\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\",\"echo\"]}"
		_, err = New(models.StepTypeLogsGather, []string{param})
		Expect(err).To(HaveOccurred())

		By("Bad boolean")
		param = "{\"base_url\":\"http://10.1.178.26:6000\",\"bootstrap\":\"echo\",\"cluster_id\":\"57a0830c-0d5f-45ad-8513-7d0060c33615\"," +
			"\"host_id\":\"9f45b240-73d5-4390-a04e-7f5a09da44f7\",\"infra_env_id\":\"ea123507-1875-4da2-968a-15bb2d4b1e91\"," +
			"\"insecure\":true,\"installer_gather\":true,\"master_ips\":[\"192.168.127.10\"]}"
		_, err = New(models.StepTypeLogsGather, []string{param})
		Expect(err).To(HaveOccurred())
	})
})