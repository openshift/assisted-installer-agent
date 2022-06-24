package actions

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("Upgrade agent action", func() {
	It("Succeeds when given the right parameters", func() {
		// Create the action:
		action, err := New(
			&config.AgentConfig{},
			models.StepTypeUpgradeAgent,
			[]string{`{
				"agent_image": "quay.io/my/image/v1.2.3"
			}`},
		)
		Expect(err).ToNot(HaveOccurred())

		// Verify the result:
		command := action.Command()
		args := action.Args()
		Expect(command).To(Equal("upgrade_agent"))
		Expect(args).To(HaveLen(1))
		arg := args[0]
		Expect(arg).To(MatchJSON(`{
			"agent_image": "quay.io/my/image/v1.2.3"
		}`))
	})

	It("Fails if given wrong parameters", func() {
		badParamsCommonTests(models.StepTypeUpgradeAgent, []string{})
	})
})
