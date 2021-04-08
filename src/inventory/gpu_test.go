package inventory

import (
	"errors"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

var (
	card1 = ghw.GraphicsCard{
		Address: "0000:00:02.0",
		DeviceInfo: &ghw.PCIDevice{
			Product: &pcidb.Product{
				VendorID: "8086",
				ID:       "3ea0",
				Name:     "UHD Graphics 620 (Whiskey Lake)",
			},
			Vendor: &pcidb.Vendor{Name: "Intel Corporation", ID: "8086"},
		},
	}
	card2 = ghw.GraphicsCard{
		Address: "0000:00:03.0",
		DeviceInfo: &ghw.PCIDevice{
			Product: &pcidb.Product{
				VendorID: "1111",
				ID:       "0000",
				Name:     "Some GPU",
			},
			Vendor: &pcidb.Vendor{Name: "Other Vendor", ID: "1111"},
		},
	}

	gpu1 = models.Gpu{
		Address:  "0000:00:02.0",
		Name:     "UHD Graphics 620 (Whiskey Lake)",
		DeviceID: "3ea0",
		Vendor:   "Intel Corporation",
		VendorID: "8086",
	}
	gpu2 = models.Gpu{
		Address:  "0000:00:03.0",
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
		dependencies.On("GPU").Return(&ghw.GPUInfo{GraphicsCards: []*ghw.GraphicsCard{&card1}}, nil).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu1))
	})

	It("should load information about multiple GPUs", func() {
		dependencies.On("GPU").Return(&ghw.GPUInfo{GraphicsCards: []*ghw.GraphicsCard{&card1, &card2}}, nil).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu1, &gpu2))
	})

	It("should handle error gracefully", func() {
		dependencies.On("GPU").Return(nil, errors.New("boom")).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(BeEmpty())
	})

	table.DescribeTable("should handle incomplete data", func(card ghw.GraphicsCard, expectedGpu models.Gpu) {
		dependencies.On("GPU").Return(&ghw.GPUInfo{GraphicsCards: []*ghw.GraphicsCard{&card}}, nil).Once()

		gpus := GetGPUs(dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&expectedGpu))
	},
		table.Entry("Missing DeviceInfo", ghw.GraphicsCard{Address: "0000:00:02.0"}, models.Gpu{Address: "0000:00:02.0"}),
		table.Entry("Missing Product Info",
			ghw.GraphicsCard{Address: "0000:00:02.0",
				DeviceInfo: &ghw.PCIDevice{
					Vendor: &pcidb.Vendor{Name: "Intel Corporation", ID: "8086"},
				},
			}, models.Gpu{Address: "0000:00:02.0", Vendor: "Intel Corporation", VendorID: "8086"}),
		table.Entry("Missing Vendor Info",
			ghw.GraphicsCard{Address: "0000:00:02.0",
				DeviceInfo: &ghw.PCIDevice{
					Product: &pcidb.Product{Name: "UHD Graphics 620 (Whiskey Lake)", ID: "3ea0", VendorID: "8086"},
				},
			},
			models.Gpu{Address: "0000:00:02.0", VendorID: "8086", DeviceID: "3ea0", Name: "UHD Graphics 620 (Whiskey Lake)"}),
	)
})
