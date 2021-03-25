package inventory

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

const (
	singleDisplayLshw = ` {
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:0000:00:02.0",
    "description" : "VGA compatible controller",
    "product" : "UHD Graphics 620 (Whiskey Lake) [8086:3ea0]",
    "vendor" : "Intel Corporation [8086]",
    "physid" : "2",
    "businfo" : "pci@0000:00:02.0",
    "version" : "02",
    "width" : 64,
    "clock" : 33000000,
    "configuration" : {
      "driver" : "i915",
      "latency" : "0"
    },
    "capabilities" : {
      "pciexpress" : "PCI Express",
      "msi" : "Message Signalled Interrupts",
      "pm" : "Power Management",
      "vga_controller" : true,
      "bus_master" : "bus mastering",
      "cap_list" : "PCI capabilities listing",
      "rom" : "extension ROM"
    }
  },`
	multipleDisplayLshw = `{
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:0000:00:02.0",
    "description" : "VGA compatible controller",
    "product" : "UHD Graphics 620 (Whiskey Lake) [8086:3ea0]",
    "vendor" : "Intel Corporation [8086]",
    "physid" : "2",
    "businfo" : "pci@0000:00:02.0",
    "version" : "02",
    "width" : 64,
    "clock" : 33000000,
    "configuration" : {
      "driver" : "i915",
      "latency" : "0"
    },
    "capabilities" : {
      "pciexpress" : "PCI Express",
      "msi" : "Message Signalled Interrupts",
      "pm" : "Power Management",
      "vga_controller" : true,
      "bus_master" : "bus mastering",
      "cap_list" : "PCI capabilities listing",
      "rom" : "extension ROM"
    }
  },  			 {
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:0000:00:03.0",
    "description" : "VGA compatible controller",
    "product" : "Some GPU [1111:0000]",
    "vendor" : "Other Vendor [1111]",
    "physid" : "3",
    "businfo" : "pci@0000:00:03.0",
    "version" : "02",
    "width" : 64,
    "clock" : 30000000,
    "configuration" : {
      "driver" : "i915",
      "latency" : "0"
    },
    "capabilities" : {
      "pciexpress" : "PCI Express",
      "msi" : "Message Signalled Interrupts",
      "pm" : "Power Management",
      "vga_controller" : true,
      "bus_master" : "bus mastering",
      "cap_list" : "PCI capabilities listing",
      "rom" : "extension ROM"
    }
  }, `
	malformedLshw = `Boom{
    "id" : "display",
    "class" : "display",
    "claimed" : true,
    "handle" : "PCI:0000:00:02.0"
	}`
)

var (
	gpu1 = models.Gpu{
		BusInfo:  "pci@0000:00:02.0",
		ClockHz:  33000000,
		Name:     "UHD Graphics 620 (Whiskey Lake)",
		DeviceID: "3ea0",
		Vendor:   "Intel Corporation",
		VendorID: "8086",
	}
	gpu1NoIDs = models.Gpu{
		BusInfo: "pci@0000:00:02.0",
		ClockHz: 33000000,
		Name:    "UHD Graphics 620 (Whiskey Lake)",
		Vendor:  "Intel Corporation",
	}
	gpu2 = models.Gpu{
		BusInfo:  "pci@0000:00:03.0",
		ClockHz:  30000000,
		Name:     "Some GPU",
		DeviceID: "0000",
		Vendor:   "Other Vendor",
		VendorID: "1111",
	}
)

var _ = Describe("GPUs information discovery", func() {

	var dependencies *util.MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("should load information about one GPU", func() {
		dependencies.On("Execute", "lshw", "-class", "display", "-json", "-numeric").Return(singleDisplayLshw, "", 0).Once()
		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu1))
	})

	It("should load information about multiple GPUs", func() {
		dependencies.On("Execute", "lshw", "-class", "display", "-json", "-numeric").Return(multipleDisplayLshw, "", 0).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeEmpty())
		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu1, &gpu2))
	})

	It("should handle lshw error gracefully", func() {
		dependencies.On("Execute", "lshw", "-class", "display", "-json", "-numeric").Return("", "Execute error", -1).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(BeEmpty())
	})

	It("should handle JSON error gracefully", func() {
		dependencies.On("Execute", "lshw", "-class", "display", "-json", "-numeric").Return(malformedLshw, "", 0).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(BeEmpty())
	})

	It("should handle empty lshw output", func() {
		dependencies.On("Execute", "lshw", "-class", "display", "-json", "-numeric").Return("", "", 0).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(BeEmpty())
	})

})
