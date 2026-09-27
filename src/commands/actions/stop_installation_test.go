package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("stop", func() {
	BeforeEach(func() {
	})

	It("stop", func() {
		action, err := New(&config.AgentConfig{}, models.StepTypeStopInstallation, []string{})
		Expect(err).NotTo(HaveOccurred())

		args := action.Args()
		command := action.Command()
		Expect(command).To(Equal("podman"))
		Expect(strings.Join(args, " ")).To(ContainSubstring("stop"))
		Expect(strings.Join(args, " ")).To(ContainSubstring("assisted-installer"))
	})

	It("stop", func() {
		badParamsCommonTests(models.StepTypeStopInstallation, []string{})
	})
})
