package inventory

import (
	"fmt"

	"github.com/jaypipes/ghw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("System vendor test", func() {
	var dependencies *util.MockIDependencies

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
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("none", "", 0).Once()

		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{
			ProductName:  "A Name",
			SerialNumber: "A Serial Number",
			Manufacturer: "A Vendor",
		}))
	})
	It("Bare metal virtualization detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         "A Name",
			SerialNumber: "A Serial Number",
			Vendor:       "A Vendor",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("none", "", 0).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Virtual).ShouldNot(BeTrue())
	})
	It("Virtual machine detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         "A Name",
			SerialNumber: "A Serial Number",
			Vendor:       "A Vendor",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("anyvirt", "", 0).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Virtual).Should(BeTrue())
	})
	It("Virtual machine error on detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         "A Name",
			SerialNumber: "A Serial Number",
			Vendor:       "A Vendor",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("", "an error", 1).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Virtual).ShouldNot(BeTrue())
	})
	It("oVirt product detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Family: "oVirt",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("ovirt", "", 0).Once()

		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{
			ProductName: "oVirt",
			Virtual:     true,
		}))
	})
})
