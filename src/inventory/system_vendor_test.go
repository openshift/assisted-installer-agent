package inventory

import (
	"fmt"
	"github.com/jaypipes/ghw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("System vendor test", func() {
	var dependencies *MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("Product error", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(nil, fmt.Errorf("Just an error")).Once()
		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{}))
	})
	It("Product OK", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         "A Name",
			SerialNumber: "A Serial Number",
			Vendor:       "A Vendor",
		}, nil).Once()

		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{
			ProductName:  "A Name",
			SerialNumber: "A Serial Number",
			Manufacturer: "A Vendor",
		}))
	})
	It("Virtual machine detection", func() {
		for _, test := range []struct {
			Product string
			IsVm    bool
		}{
			{"KVM", true},
			{"VirtualBox ()", true},
			{"VMware Virtual Platform ()", true},
			{"Virtual Machine", true},
			{"20T1S39D3N (LENOVO_MT_20T1_BU_Think_FM_ThinkPad T14s Gen 1)", false},
		} {
			Expect(isVirtual(test.Product)).Should(Equal(test.IsVm))
		}
	})
})
