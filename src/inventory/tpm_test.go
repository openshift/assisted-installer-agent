package inventory

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
)

var _ = Describe("TPM test", func() {

	var dependencies *util.MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("TPM disabled in BIOS", func() {

		dependencies.On("Execute", "cat", "/sys/class/tpm/tpm0/tpm_version_major").Return("", "cat: /sys/class/tpm/tpm0/tpm_version_major: No such file or directory", 1).Once()
		ret := GetTPM(dependencies)
		Expect(ret).To(Equal("none"))
	})

	It("Execute error", func() {

		dependencies.On("Execute", "cat", "/sys/class/tpm/tpm0/tpm_version_major").Return("", "Any other error", 1).Once()
		ret := GetTPM(dependencies)
		Expect(ret).To(Equal(""))
	})

	It("Unsupported TPM version", func() {

		dependencies.On("Execute", "cat", "/sys/class/tpm/tpm0/tpm_version_major").Return("1", "", 0).Once()
		ret := GetTPM(dependencies)
		Expect(ret).To(Equal("1.2"))
	})

	It("Happy flow", func() {

		dependencies.On("Execute", "cat", "/sys/class/tpm/tpm0/tpm_version_major").Return("2", "", 0).Once()
		ret := GetTPM(dependencies)
		Expect(ret).To(Equal("2.0"))
	})
})
