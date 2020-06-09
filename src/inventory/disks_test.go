package inventory

import (
	"fmt"
	"os"

	"github.com/filanov/bm-inventory/models"
	"github.com/jaypipes/ghw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disks test", func() {
	var dependencies *MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	It("Execute error", func() {
		dependencies.On("Block").Return(nil, fmt.Errorf("Just an error")).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})
	It("Empty", func() {
		dependencies.On("Block").Return(&ghw.BlockInfo{}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})
	It("Single disk", func() {
		fileInfoMock := FileInfoMock{}
		fileInfoMock.On("Name").Return("scsi").Once()
		dependencies.On("ReadDir", "/sys/block/disk1/device/scsi_device").Return([]os.FileInfo{&fileInfoMock}, nil).Once()
		dependencies.On("EvalSymlinks", "/dev/disk/by-path/bus-path").Return("/dev/disk/by-path/../../disk1", nil).Once()
		dependencies.On("Abs", "/dev/disk/by-path/../../disk1").Return("/dev/disk1", nil).Once()
		dependencies.On("Block").Return(&ghw.BlockInfo{
			Disks: []*ghw.Disk{
				{
					Name:         "disk1",
					SizeBytes:    5555,
					DriveType:    ghw.DRIVE_TYPE_HDD,
					BusPath:      "bus-path",
					Vendor:       "disk1-vendor",
					Model:        "disk1-model",
					SerialNumber: "disk1-serial",
					WWN:          "disk1-wwn",
				},
			},
		}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{
			{
				ByPath:    "/dev/disk/by-path/bus-path",
				DriveType: "HDD",
				Hctl:      "scsi",
				Model:     "disk1-model",
				Name:      "disk1",
				Path:      "/dev/disk1",
				Serial:    "disk1-serial",
				SizeBytes: 5555,
				Vendor:    "disk1-vendor",
				Wwn:       "disk1-wwn",
			},
		}))
	})
})
