package inventory

import (
	"errors"
	"os"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

var (
	card1 = ghw.PCIDevice{
		Address: "0000:00:02.0",
		Class: &pcidb.Class{
			ID:   "03",
			Name: "Display Controller",
		},
		Subclass: &pcidb.Subclass{
			ID:   "00",
			Name: "VGA compatible controller",
		},
		Product: &pcidb.Product{
			VendorID: "8086",
			ID:       "3ea0",
			Name:     "UHD Graphics 620 (Whiskey Lake)",
		},
		Vendor: &pcidb.Vendor{
			Name: "Intel Corporation",
			ID:   "8086",
		},
	}
	card2 = ghw.PCIDevice{
		Address: "0000:00:03.0",
		Class: &pcidb.Class{
			ID:   "03",
			Name: "Display Controller",
		},
		Subclass: &pcidb.Subclass{
			ID:   "02",
			Name: "3D controller",
		},
		Product: &pcidb.Product{
			VendorID: "10de",
			ID:       "20f1",
			Name:     "GA100 [A100 PCIe 40GB]",
		},
		Vendor: &pcidb.Vendor{
			Name: "NVIDIA Corporation",
			ID:   "10de",
		},
	}
	card3 = ghw.PCIDevice{
		Address: "0000:00:04.0",
		Class: &pcidb.Class{
			ID:   "12",
			Name: "Processing accelerators",
		},
		Subclass: &pcidb.Subclass{
			ID:   "00",
			Name: "Processing accelerators",
		},
		Product: &pcidb.Product{
			VendorID: "1da3",
			ID:       "1020",
			Name:     "Gaudi2 AI Training Accelerator",
		},
		Vendor: &pcidb.Vendor{
			Name: "Habana Labs Ltd.",
			ID:   "1da3",
		},
	}
	card3a = ghw.PCIDevice{
		Address: "0000:00:04.1",
		Class: &pcidb.Class{
			ID:   "12",
			Name: "Processing accelerators",
		},
		Subclass: &pcidb.Subclass{
			ID:   "00",
			Name: "Processing accelerators",
		},
		Product: &pcidb.Product{
			VendorID: "1da3",
			ID:       "1020",
			Name:     "Gaudi2 AI Training Accelerator",
		},
		Vendor: &pcidb.Vendor{
			Name: "Habana Labs Ltd.",
			ID:   "1da3",
		},
	}
	card4 = ghw.PCIDevice{
		Address: "0000:00:05.0",
		Class: &pcidb.Class{
			ID:   "03",
			Name: "Display Controller",
		},
		Subclass: &pcidb.Subclass{
			ID:   "80",
			Name: "Display Controller",
		},
		Product: &pcidb.Product{
			VendorID: "1002",
			ID:       "740f",
			Name:     "Aldebaran/MI200 [Instinct MI210]",
		},
		Vendor: &pcidb.Vendor{
			Name: "Advanced Micro Devices, Inc. [AMD/ATI]",
			ID:   "1002",
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
		Name:     "GA100 [A100 PCIe 40GB]",
		DeviceID: "20f1",
		Vendor:   "NVIDIA Corporation",
		VendorID: "10de",
	}
	gpu3 = models.Gpu{
		Address:  "0000:00:04.0",
		Name:     "Gaudi2 AI Training Accelerator",
		DeviceID: "1020",
		Vendor:   "Habana Labs Ltd.",
		VendorID: "1da3",
	}
	gpu3a = models.Gpu{
		Address:  "0000:00:04.1",
		Name:     "Gaudi2 AI Training Accelerator",
		DeviceID: "1020",
		Vendor:   "Habana Labs Ltd.",
		VendorID: "1da3",
	}
	gpu4 = models.Gpu{
		Address:  "0000:00:05.0",
		Name:     "Aldebaran/MI200 [Instinct MI210]",
		DeviceID: "740f",
		Vendor:   "Advanced Micro Devices, Inc. [AMD/ATI]",
		VendorID: "1002",
	}
)

var _ = Describe("GPUs information discovery", func() {

	var dependencies *util.MockIDependencies
	var subprocessConfig *config.SubprocessConfig

	BeforeEach(func() {
		dependencies = newDependenciesMock()
		subprocessConfig = &config.SubprocessConfig{}
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("should load information about one GPU", func() {
		dependencies.On("PCI").Return(&ghw.PCIInfo{Devices: []*ghw.PCIDevice{&card1}}, nil).Once()

		gpus := GetGPUs(subprocessConfig, dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu1))
	})

	It("should load information about multiple GPUs", func() {
		dependencies.On("PCI").Return(&ghw.PCIInfo{Devices: []*ghw.PCIDevice{&card1, &card2, &card3, &card4}}, nil).Once()

		gpus := GetGPUs(subprocessConfig, dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu1, &gpu2, &gpu3, &gpu4))
	})

	It("should load information about Gaudi GPUs only", func() {
		dependencies.On("PCI").Return(&ghw.PCIInfo{Devices: []*ghw.PCIDevice{&card1, &card2, &card3, &card4}}, nil).Once()

		var yamlData = "---\nvendors:\n  - '1200 1da3'"

		tmpFile, err := os.CreateTemp("", "gpus*.yaml")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(yamlData)
		Expect(err).NotTo(HaveOccurred())
		tmpFile.Close()

		subprocessConfig.GPUConfigFile = tmpFile.Name()
		gpus := GetGPUs(subprocessConfig, dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu3))
	})

	It("should work with a configuration with extra whitespaces", func() {
		dependencies.On("PCI").Return(&ghw.PCIInfo{Devices: []*ghw.PCIDevice{&card1, &card2, &card3, &card4}}, nil).Once()

		var yamlData = "---\nvendors:\n  - '0302  10de '"

		tmpFile, err := os.CreateTemp("", "gpus*.yaml")
		Expect(err).NotTo(HaveOccurred())
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(yamlData)
		Expect(err).NotTo(HaveOccurred())
		tmpFile.Close()

		subprocessConfig.GPUConfigFile = tmpFile.Name()
		gpus := GetGPUs(subprocessConfig, dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu2))
	})

	// This is important for some OCP functionalities
	// Each GPU, even within the same PCI card, must be detected as a different GPU
	It("should detect PCI cards con several functions", func() {
		dependencies.On("PCI").Return(&ghw.PCIInfo{Devices: []*ghw.PCIDevice{&card1, &card2, &card3, &card3a}}, nil).Once()

		gpus := GetGPUs(subprocessConfig, dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(ConsistOf(&gpu1, &gpu2, &gpu3, &gpu3a))
	})

	It("should handle error gracefully", func() {
		dependencies.On("PCI").Return(nil, errors.New("boom")).Once()

		gpus := GetGPUs(subprocessConfig, dependencies)

		Expect(gpus).ToNot(BeNil())
		Expect(gpus).To(BeEmpty())
	})
})
