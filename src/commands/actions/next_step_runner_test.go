package actions

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/copier"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

func runNextRunner(params string, expectedError bool) (string, []string) {
	action := nextStepRunner{args: []string{params}}
	err := action.Validate()
	if expectedError {
		Expect(err).To(HaveOccurred())
		return "", nil
	}
	Expect(err).NotTo(HaveOccurred())
	return action.CreateCmd()
}

var _ = Describe("next step runner", func() {
	var params string
	var runnerArgs models.NextStepCmdRequest
	var oldConfig config.ConnectivityConfig

	BeforeEach(func() {
		Expect(copier.Copy(&oldConfig, &config.GlobalAgentConfig.ConnectivityConfig)).To(BeNil())
		config.GlobalAgentConfig.InsecureConnection = true
		config.GlobalAgentConfig.TargetURL = "http://10.1.178.26:6000"
		hostId := strfmt.UUID("f7ac1860-92cf-4ed8-aeec-2d9f20b35bab")
		infraEnvId := strfmt.UUID("456eecf6-7aec-402d-b453-f609b19783cb")
		runnerArgs = models.NextStepCmdRequest{
			HostID:       &hostId,
			InfraEnvID:   &infraEnvId,
			AgentVersion: swag.String("quay.io/edge-infrastructure/assisted-installer-controller:latest"),
		}
		b, err := json.Marshal(&runnerArgs)
		Expect(err).NotTo(HaveOccurred())
		params = string(b)
	})
	AfterEach(func() {
		Expect(copier.Copy(&config.GlobalAgentConfig.ConnectivityConfig, &oldConfig)).To(BeNil())
	})

	It("next step runner", func() {
		command, args := runNextRunner(params, false)
		Expect(command).To(Equal("podman"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
			"/dev",
			"/etc/pki",
			"/run/media",
		}
		argsAsString := strings.Join(args, " ")
		verifyPaths(argsAsString, paths)
		Expect(argsAsString).To(ContainSubstring("--env PULL_SECRET_TOKEN"))
		Expect(argsAsString).To(ContainSubstring("--env CONTAINERS_CONF"))
		Expect(argsAsString).To(ContainSubstring("--env CONTAINERS_STORAGE_CONF"))
		Expect(argsAsString).To(ContainSubstring("--env HTTP_PROXY"))
		Expect(argsAsString).To(ContainSubstring("--env HTTPS_PROXY"))
		Expect(argsAsString).To(ContainSubstring("--env http_proxy"))
		Expect(argsAsString).To(ContainSubstring("--env https_proxy"))
		Expect(argsAsString).To(ContainSubstring("--env NO_PROXY"))
		Expect(argsAsString).To(ContainSubstring("--env no_proxy"))
		Expect(argsAsString).To(ContainSubstring("--name next-step-runner"))
		Expect(argsAsString).To(ContainSubstring(fmt.Sprintf("--insecure=%s", strconv.FormatBool(config.GlobalAgentConfig.InsecureConnection))))
		Expect(argsAsString).To(ContainSubstring(fmt.Sprintf("--agent-version %s", swag.StringValue(runnerArgs.AgentVersion))))
		Expect(argsAsString).To(ContainSubstring(fmt.Sprintf("--url %s", config.GlobalAgentConfig.TargetURL)))
		Expect(argsAsString).To(ContainSubstring(fmt.Sprintf("--host-id %s", runnerArgs.HostID.String())))
		Expect(argsAsString).To(ContainSubstring(fmt.Sprintf("--infra-env-id %s", runnerArgs.InfraEnvID.String())))
	})

	It("next step runner ca cert", func() {
		caPath := "/ca_cert"
		config.GlobalAgentConfig.CACertificatePath = caPath
		b, err := json.Marshal(&runnerArgs)
		Expect(err).NotTo(HaveOccurred())
		_, args := runNextRunner(string(b), false)
		paths := []string{
			"/ca_cert",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(strings.Join(args, " ")).To(ContainSubstring("--cacert /ca_cert"))
	})

	It("next step runner insecure false", func() {
		config.GlobalAgentConfig.InsecureConnection = false
		b, err := json.Marshal(&runnerArgs)
		Expect(err).NotTo(HaveOccurred())
		_, args := runNextRunner(string(b), false)
		Expect(strings.Join(args, " ")).To(ContainSubstring("--insecure=false"))
	})
	It("bad commands", func() {
		By("bad command")
		_, _ = runNextRunner("echo aaaa", true)

		if len(params) > 0 {
			_, _ = runNextRunner("echo aaaa", true)
			By("Less then 1")
			action := nextStepRunner{args: []string{}}
			err := action.Validate()
			Expect(err).To(HaveOccurred())
		}

		By("More then expected")
		action := nextStepRunner{args: []string{params, "echo aaaa"}}
		err := action.Validate()
		Expect(err).To(HaveOccurred())
	})
})