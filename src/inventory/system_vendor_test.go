package inventory

import (
	"fmt"

	"github.com/jaypipes/ghw"
	ghwutil "github.com/jaypipes/ghw/pkg/util"
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
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
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
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
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
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
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
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("", "an error", 1).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Virtual).ShouldNot(BeTrue())
	})
	It("oVirt product detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Family: "oVirt",
		}, nil).Once()
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("ovirt", "", 0).Once()

		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{
			ProductName: "oVirt",
			Virtual:     true,
		}))
	})
	It("Chassis error", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         "A Name",
			SerialNumber: "A Serial Number",
			Vendor:       "A Vendor",
		}, nil).Once()
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(nil, fmt.Errorf("Just an error")).Once()
		ret := GetVendor(dependencies)
		Expect(ret).To(Equal(&models.SystemVendor{}))
	})
	It("Oracle Cloud detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         "A Name",
			SerialNumber: "A Serial Number",
			Vendor:       "A Vendor",
		}, nil).Once()
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "OracleCloud.com",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("none", "", 0).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Manufacturer).Should(Equal("OracleCloud.com"))
	})
	It("s390x zVM node detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         ghwutil.UNKNOWN,
			SerialNumber: ghwutil.UNKNOWN,
			Vendor:       ghwutil.UNKNOWN,
		}, nil).Once()
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("zvm", "", 0).Once()
		dependencies.On("Execute", "grep", "-m1", VENDOR_ID, "/proc/cpuinfo").Return("vendor_id       : IBM/S390", "", 0).Once()
		dependencies.On("Execute", "grep", VM_CTRL_PRG, "/proc/sysinfo").Return("VM00 Control Program: z/VM    7.2.0", "", 0).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Manufacturer).Should(Equal(VENDOR_IBM_ID))
		Expect(systemVendor.ProductName).Should(Equal("z/VM    7.2.0"))
	})
	It("s390x KVM node detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         ghwutil.UNKNOWN,
			SerialNumber: ghwutil.UNKNOWN,
			Vendor:       ghwutil.UNKNOWN,
		}, nil).Once()
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("kvm", "", 0).Once()
		dependencies.On("Execute", "grep", "-m1", VENDOR_ID, "/proc/cpuinfo").Return("vendor_id       : IBM/S390", "", 0).Once()
		dependencies.On("Execute", "grep", VM_CTRL_PRG, "/proc/sysinfo").Return("VM00 Control Program: KVM/Linux", "", 0).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Manufacturer).Should(Equal(VENDOR_IBM_ID))
		Expect(systemVendor.ProductName).Should(Equal("KVM/Linux"))
	})
	It("s390x LPAR node detection", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         ghwutil.UNKNOWN,
			SerialNumber: ghwutil.UNKNOWN,
			Vendor:       ghwutil.UNKNOWN,
		}, nil).Once()
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("", "", 0).Once()
		dependencies.On("Execute", "grep", "-m1", VENDOR_ID, "/proc/cpuinfo").Return("vendor_id       : IBM/S390", "", 0).Once()
		dependencies.On("Execute", "grep", VM_CTRL_PRG, "/proc/sysinfo").Return("", "", 0).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Manufacturer).Should(Equal(VENDOR_IBM_ID))
		Expect(systemVendor.ProductName).Should(Equal("LPAR"))
	})
	It("s390x LPAR node detection (glitch during detect virt)", func() {
		dependencies.On("Product", ghw.WithChroot("/host")).Return(&ghw.ProductInfo{
			Name:         ghwutil.UNKNOWN,
			SerialNumber: ghwutil.UNKNOWN,
			Vendor:       ghwutil.UNKNOWN,
		}, nil).Once()
		dependencies.On("Chassis", ghw.WithChroot("/host")).Return(&ghw.ChassisInfo{
			AssetTag: "An Asset Tag",
		}, nil).Once()
		dependencies.On("Execute", "systemd-detect-virt", "--vm").Return("", "an error", 1).Once()
		dependencies.On("Execute", "grep", "-m1", VENDOR_ID, "/proc/cpuinfo").Return("vendor_id       : IBM/S390", "", 0).Once()
		dependencies.On("Execute", "grep", VM_CTRL_PRG, "/proc/sysinfo").Return("", "", 0).Once()
		systemVendor := GetVendor(dependencies)
		Expect(systemVendor.Manufacturer).Should(Equal(VENDOR_IBM_ID))
		Expect(systemVendor.ProductName).Should(Equal("LPAR"))
	})
})
