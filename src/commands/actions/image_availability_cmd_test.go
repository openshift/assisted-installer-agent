package actions

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("image availability", func() {
	var param string

	BeforeEach(func() {
		param = "{\"images\":[\"quay.io/openshift-release-dev/ocp-release:4.9.19-x86_64\"," +
			"\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:7eff99c5449bf8a6d6223f1a644caf36bbfaa4f2589e7bcac74b165c43b7bffe\"," +
			"\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:1f3994a75464c01f1953aaeda23c2a02c477e1b5ea36eb3434123ecccd141b0c\"," +
			"\"registry.redhat.io/rhai-tech-preview/assisted-installer-rhel8:v1.0.0-125\"],\"timeout\":960}"
	})

	It("image availability", func() {
		action, err := New(models.StepTypeContainerImageAvailability, []string{param})
		Expect(err).NotTo(HaveOccurred())

		command, args := action.Run()
		Expect(command).To(Equal("sh"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(args[len(args)-1]).To(ContainSubstring(param))
		Expect(args[len(args)-1]).To(ContainSubstring("container_image_availability"))
	})

	It("image availability bad input", func() {
		badParamsCommonTests(models.StepTypeContainerImageAvailability, []string{param})
	})
})