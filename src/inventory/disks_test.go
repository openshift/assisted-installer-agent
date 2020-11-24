package inventory

import (
	"fmt"
	"os"

	"github.com/jaypipes/ghw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
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
		dependencies.On("Block", ghw.WithChroot("/host")).Return(nil, fmt.Errorf("Just an error")).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})

	It("Empty", func() {
		dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})

	Describe("Single disk", func() {
		var fileInfoMock FileInfoMock
		var expectation []*models.Disk

		BeforeEach(func() {
			fileInfoMock = FileInfoMock{}
			fileInfoMock.On("Name").Return("scsi").Once()
			// Don't find it under /dev/disk1 to test the fallback of searching /dev/disk/by-path
			dependencies.On("Stat", "/dev/disk1").Return(nil, errors.New("error")).Once()
			dependencies.On("ReadDir", "/sys/block/disk1/device/scsi_device").Return([]os.FileInfo{&fileInfoMock}, nil).Once()
			dependencies.On("EvalSymlinks", "/dev/disk/by-path/bus-path").Return("/dev/disk/by-path/../../foo/disk1", nil).Once()
			dependencies.On("Abs", "/dev/disk/by-path/../../foo/disk1").Return("/dev/foo/disk1", nil).Once()
			dependencies.On("Stat", "/dev/disk/by-path/bus-path").Return(nil, nil).Once()
			dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{
				Disks: []*ghw.Disk{
					{
						Name:                   "disk1",
						SizeBytes:              5555,
						DriveType:              ghw.DRIVE_TYPE_HDD,
						BusPath:                "bus-path",
						Vendor:                 "disk1-vendor",
						Model:                  "disk1-model",
						SerialNumber:           "disk1-serial",
						WWN:                    "disk1-wwn",
						BusType:                ghw.BUS_TYPE_SCSI,
						IsRemovable:            false,
						NUMANodeID:             0,
						PhysicalBlockSizeBytes: 512,
						StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
					},
				},
			}, nil).Once()

			expectation = []*models.Disk{
				{
					ByPath:    "/dev/disk/by-path/bus-path",
					DriveType: "HDD",
					Hctl:      "scsi",
					Model:     "disk1-model",
					Name:      "disk1",
					Path:      "/dev/foo/disk1",
					Serial:    "disk1-serial",
					SizeBytes: 5555,
					Vendor:    "disk1-vendor",
					Wwn:       "disk1-wwn",
					Bootable:  true,
					Smart:     `{"some": "json"}`,
				},
			}
		})

		It("Bootable", func() {
			dependencies.On("Execute", "file", "-s", "/dev/foo/disk1").Return(" DOS/MBR boot sector", "", 0).Once()
			dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1").Return(`{"some": "json"}`, "", 0).Once()
		})

		It("Non-bootable", func() {
			dependencies.On("Execute", "file", "-s", "/dev/foo/disk1").Return("Linux rev 1.0 ext4 filesystem data", "", 0).Once()
			dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1").Return(`{"some": "json"}`, "", 0).Once()
			expectation[0].Bootable = false
		})

		It("Without a smartctl error", func() {
			dependencies.On("Execute", "file", "-s", "/dev/foo/disk1").Return(" DOS/MBR boot sector", "", 0).Once()
			dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1").Return(`{"some": "json"}`, "", 0).Once()
		})

		It("With a smartctl error", func() {
			dependencies.On("Execute", "file", "-s", "/dev/foo/disk1").Return(" DOS/MBR boot sector", "", 0).Once()
			dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1").Return(`{"some": "error"}`, "", 2).Once()
			expectation[0].Smart = ""
		})

		AfterEach(func() {
			ret := GetDisks(dependencies)
			Expect(ret).To(Equal(expectation))
		})
	})

	It("filters ISO disks", func() {
		dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{
			Disks: []*ghw.Disk{
				{
					Name:                   "disk1",
					SizeBytes:              5555,
					DriveType:              ghw.DRIVE_TYPE_HDD,
					BusPath:                "bus-path1",
					Vendor:                 "disk1-vendor",
					Model:                  "disk1-model",
					SerialNumber:           "disk1-serial",
					WWN:                    "disk1-wwn",
					BusType:                ghw.BUS_TYPE_SCSI,
					IsRemovable:            false,
					NUMANodeID:             0,
					PhysicalBlockSizeBytes: 512,
					StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
					Partitions: []*ghw.Partition{
						{
							Disk:       nil,
							Name:       "partition1",
							Label:      "partition1-label",
							MountPoint: "/media/iso",
							SizeBytes:  5555,
							Type:       "ext4",
							IsReadOnly: false,
						},
					},
				},
				{
					Name:                   "disk2",
					SizeBytes:              5555,
					DriveType:              ghw.DRIVE_TYPE_HDD,
					BusPath:                "bus-path2",
					Vendor:                 "disk1-vendor",
					Model:                  "disk1-model",
					SerialNumber:           "disk1-serial",
					WWN:                    "disk1-wwn",
					BusType:                ghw.BUS_TYPE_SCSI,
					IsRemovable:            false,
					NUMANodeID:             0,
					PhysicalBlockSizeBytes: 512,
					StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
					Partitions: []*ghw.Partition{
						{
							Disk:       nil,
							Name:       "partition2",
							Label:      "partition2-label",
							MountPoint: "/some/mount/point",
							SizeBytes:  5555,
							Type:       "iso9660",
							IsReadOnly: false,
						},
					},
				},
			},
		}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})
	It("Multiple disks", func() {
		fileInfoMock := FileInfoMock{}
		fileInfoMock.On("Name").Return("scsi").Times(2)
		// Don't find it under /dev/disk1 to test the fallback of searching /dev/disk/by-path
		dependencies.On("Stat", "/dev/disk1").Return(nil, errors.New("error")).Once()
		dependencies.On("Stat", "/dev/disk2").Return(nil, errors.New("error")).Once()
		dependencies.On("ReadDir", "/sys/block/disk1/device/scsi_device").Return([]os.FileInfo{&fileInfoMock}, nil).Once()
		dependencies.On("ReadDir", "/sys/block/disk2/device/scsi_device").Return([]os.FileInfo{&fileInfoMock}, nil).Once()
		dependencies.On("EvalSymlinks", "/dev/disk/by-path/bus-path1").Return("/dev/disk/by-path/../../foo/disk1", nil).Once()
		dependencies.On("EvalSymlinks", "/dev/disk/by-path/bus-path2").Return("/dev/disk/by-path/../../foo/disk2", nil).Once()
		dependencies.On("Abs", "/dev/disk/by-path/../../foo/disk1").Return("/dev/foo/disk1", nil).Once()
		dependencies.On("Abs", "/dev/disk/by-path/../../foo/disk2").Return("/dev/foo/disk2", nil).Once()
		dependencies.On("Stat", "/dev/disk/by-path/bus-path1").Return(nil, nil).Once()
		dependencies.On("Stat", "/dev/disk/by-path/bus-path2").Return(nil, nil).Once()
		dependencies.On("Execute", "file", "-s", "/dev/foo/disk1").Return("Linux rev 1.0 ext4 filesystem data", "", 0).Once()
		dependencies.On("Execute", "file", "-s", "/dev/foo/disk2").Return(" DOS/MBR boot sector", "", 0).Once()
		dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1").Return(`{"some": "json"}`, "", 0).Once()
		dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk2").Return(`{"some": "json"}`, "", 0).Once()
		dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{
			Disks: []*ghw.Disk{
				{
					Name:                   "disk1",
					SizeBytes:              5555,
					DriveType:              ghw.DRIVE_TYPE_HDD,
					BusPath:                "bus-path1",
					Vendor:                 "disk1-vendor",
					Model:                  "disk1-model",
					SerialNumber:           "disk1-serial",
					WWN:                    "disk1-wwn",
					BusType:                ghw.BUS_TYPE_SCSI,
					IsRemovable:            false,
					NUMANodeID:             0,
					PhysicalBlockSizeBytes: 512,
					StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
				},
				{
					Name:                   "disk2",
					SizeBytes:              5555,
					DriveType:              ghw.DRIVE_TYPE_HDD,
					BusPath:                "bus-path2",
					Vendor:                 "disk1-vendor",
					Model:                  "disk1-model",
					SerialNumber:           "disk1-serial",
					WWN:                    "disk1-wwn",
					BusType:                ghw.BUS_TYPE_SCSI,
					IsRemovable:            false,
					NUMANodeID:             0,
					PhysicalBlockSizeBytes: 512,
					StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
				},
			},
		}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{
			{
				ByPath:    "/dev/disk/by-path/bus-path1",
				DriveType: "HDD",
				Hctl:      "scsi",
				Model:     "disk1-model",
				Name:      "disk1",
				Path:      "/dev/foo/disk1",
				Serial:    "disk1-serial",
				SizeBytes: 5555,
				Vendor:    "disk1-vendor",
				Wwn:       "disk1-wwn",
				Bootable:  false,
				Smart:     `{"some": "json"}`,
			},
			{
				ByPath:    "/dev/disk/by-path/bus-path2",
				DriveType: "HDD",
				Hctl:      "scsi",
				Model:     "disk1-model",
				Name:      "disk2",
				Path:      "/dev/foo/disk2",
				Serial:    "disk1-serial",
				SizeBytes: 5555,
				Vendor:    "disk1-vendor",
				Wwn:       "disk1-wwn",
				Bootable:  true,
				Smart:     `{"some": "json"}`,
			},
		}))
	})
	It("AWS Xen EBS disk", func() {
		/*
			# ls -l /sys/block/xvda/device/
			total 0
			drwxr-xr-x. 3 root root    0 Aug  6 07:30 block
			-r--r--r--. 1 root root 4096 Aug  6 07:40 devtype
			lrwxrwxrwx. 1 root root    0 Aug  6 07:40 driver -> ../../bus/xen/drivers/vbd
			-r--r--r--. 1 root root 4096 Aug  6 07:40 modalias
			-r--r--r--. 1 root root 4096 Aug  6 07:40 nodename
			drwxr-xr-x. 2 root root    0 Aug  6 07:40 power
			lrwxrwxrwx. 1 root root    0 Aug  6 07:30 subsystem -> ../../bus/xen
			-rw-r--r--. 1 root root 4096 Aug  6 07:30 uevent

			# ls /dev/disk/
				by-label  by-partlabel  by-partuuid  by-uuid
		*/
		dependencies.On("Stat", "/dev/xvda").Return(nil, nil).Once()
		dependencies.On("ReadDir", "/sys/block/xvda/device/scsi_device").Return(nil, errors.New("error")).Once()
		dependencies.On("Execute", "file", "-s", "/dev/xvda").Return(" DOS/MBR boot sector", "", 0).Once()
		dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/xvda").Return(`{"some": "json"}`, "", 0).Once()
		dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{
			Disks: []*ghw.Disk{
				{
					Name:                   "xvda",
					SizeBytes:              21474836480,
					DriveType:              ghw.DRIVE_TYPE_SSD,
					BusPath:                "unknown",
					Vendor:                 "unknown",
					Model:                  "unknown",
					SerialNumber:           "unknown",
					WWN:                    "unknown",
					BusType:                ghw.BUS_TYPE_SCSI,
					IsRemovable:            false,
					NUMANodeID:             0,
					PhysicalBlockSizeBytes: 512,
					StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
				},
			},
		}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{
			{
				ByPath:    "",
				DriveType: "SSD",
				Hctl:      "",
				Model:     "",
				Name:      "xvda",
				Path:      "/dev/xvda",
				Serial:    "",
				SizeBytes: 21474836480,
				Vendor:    "",
				Wwn:       "",
				Bootable:  true,
				Smart:     `{"some": "json"}`,
			},
		}))
	})
	It("Fedora 32 NVME", func() {
		/*
			# ls -l /sys/block/nvme0n1/device/
			total 0
			-r--r--r--.  1 root root 4096 Aug  6 10:48 address
			-r--r--r--.  1 root root 4096 Aug  6 10:48 cntlid
			-r--r--r--.  1 root root 4096 Aug  6 10:48 dev
			lrwxrwxrwx.  1 root root    0 Aug  6 10:48 device -> ../../../0000:3d:00.0
			-r--r--r--.  1 root root 4096 Aug  6 10:48 firmware_rev
			-r--r--r--.  1 root root 4096 Aug  4 18:14 model
			-r--r--r--.  1 root root 4096 Aug  6 10:48 numa_node
			drwxr-xr-x. 12 root root    0 Aug  4 15:51 nvme0n1
			drwxr-xr-x.  2 root root    0 Aug  6 10:48 power
			-r--r--r--.  1 root root 4096 Aug  6 10:48 queue_count
			--w-------.  1 root root 4096 Aug  6 10:48 rescan_controller
			--w-------.  1 root root 4096 Aug  6 10:48 reset_controller
			-r--r--r--.  1 root root 4096 Aug  6 10:48 serial
			-r--r--r--.  1 root root 4096 Aug  6 10:48 sqsize
			-r--r--r--.  1 root root 4096 Aug  6 10:48 state
			-r--r--r--.  1 root root 4096 Aug  6 10:48 subsysnqn
			lrwxrwxrwx.  1 root root    0 Aug  4 17:08 subsystem -> ../../../../../../class/nvme
			-r--r--r--.  1 root root 4096 Aug  6 10:48 transport
			-rw-r--r--.  1 root root 4096 Aug  4 17:08 uevent

			# ls -l /dev/disk/by-path/pci-0000\:3d\:00.0-nvme-1
			lrwxrwxrwx. 1 root root 13 Aug  2 09:18 /dev/disk/by-path/pci-0000:3d:00.0-nvme-1 -> ../../nvme0n1
		*/
		fileInfoMock := FileInfoMock{}
		fileInfoMock.On("Name").Return("scsi").Once()
		// Don't find it under /dev/disk1 to test the fallback of searching /dev/disk/by-path
		dependencies.On("Stat", "/dev/nvme0n1").Return(nil, nil).Once()
		dependencies.On("ReadDir", "/sys/block/nvme0n1/device/scsi_device").Return(nil, errors.New("error")).Once()
		dependencies.On("Stat", "/dev/disk/by-path/pci-0000:3d:00.0-nvme-1").Return(nil, nil).Once()
		dependencies.On("Execute", "file", "-s", "/dev/nvme0n1").Return(" kuku", "", 0).Once()
		dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/nvme0n1").Return(`{"some": "json"}`, "", 0).Once()
		dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{
			Disks: []*ghw.Disk{
				{
					Name:                   "nvme0n1",
					SizeBytes:              256060514304,
					DriveType:              ghw.DRIVE_TYPE_SSD,
					BusPath:                "pci-0000:3d:00.0-nvme-1",
					Vendor:                 "unknown",
					Model:                  "INTEL SSDPEKKF256G8L",
					SerialNumber:           "PHHP942200RN256B",
					WWN:                    "eui.5cd2e42a91419c24",
					BusType:                ghw.BUS_TYPE_NVME,
					IsRemovable:            false,
					NUMANodeID:             0,
					PhysicalBlockSizeBytes: 512,
					StorageController:      ghw.STORAGE_CONTROLLER_NVME,
				},
			},
		}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{
			{
				ByPath:    "/dev/disk/by-path/pci-0000:3d:00.0-nvme-1",
				DriveType: "SSD",
				Hctl:      "",
				Model:     "INTEL SSDPEKKF256G8L",
				Name:      "nvme0n1",
				Path:      "/dev/nvme0n1",
				Serial:    "PHHP942200RN256B",
				SizeBytes: 256060514304,
				Vendor:    "",
				Wwn:       "eui.5cd2e42a91419c24",
				Bootable:  false,
				Smart:     `{"some": "json"}`,
			},
		}))
	})
	It("Fedora 32 DM filter", func() {
		dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{
			Disks: []*ghw.Disk{
				{
					Name:                   "dm-0",
					SizeBytes:              237561184256,
					DriveType:              ghw.DRIVE_TYPE_SSD,
					BusPath:                ghw.UNKNOWN,
					Vendor:                 ghw.UNKNOWN,
					Model:                  ghw.UNKNOWN,
					SerialNumber:           ghw.UNKNOWN,
					WWN:                    ghw.UNKNOWN,
					BusType:                ghw.BUS_TYPE_UNKNOWN,
					IsRemovable:            false,
					NUMANodeID:             0,
					PhysicalBlockSizeBytes: 512,
					StorageController:      ghw.STORAGE_CONTROLLER_UNKNOWN,
				},
			},
		}, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})
})
