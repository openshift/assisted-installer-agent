package inventory

import (
	"fmt"
	"github.com/thoas/go-funk"
	"os"

	"github.com/jaypipes/ghw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/openshift/assisted-service/models"
)

// deleteExpectedCall allows deleting call expectations from a mock according to method name and arguments
func deleteExpectedCall(mockDeps *MockIDependencies, methodName string, arguments ...interface{}) {
	deleteIndex := -1
	for i, expectedCall := range mockDeps.ExpectedCalls {
		if expectedCall.Method == methodName && len(expectedCall.Arguments) == len(arguments) {
			sameArguments := true
			for _, tup := range funk.Zip(expectedCall.Arguments, arguments) {
				if tup.Element1 != tup.Element2 {
					sameArguments = false
					break
				}
			}

			if sameArguments {
				deleteIndex = i
				break
			}
		}
	}

	Expect(deleteIndex).ToNot(Equal(-1))

	// Remove
	mockDeps.ExpectedCalls = append(
		mockDeps.ExpectedCalls[:deleteIndex],
		mockDeps.ExpectedCalls[deleteIndex+1:]...,
	)
}

func prepareDiskObjects(dependencies *MockIDependencies, diskNum int) (*ghw.Disk, *models.Disk) {
	fileInfoMock := FileInfoMock{}
	fileInfoMock.On("Name").Return("scsi").Once()
	// Don't find it under /dev/disk1 to test the fallback of searching /dev/disk/by-path
	dependencies.On("Stat", fmt.Sprintf("/dev/disk%d", diskNum)).Return(nil, errors.New("error")).Once()
	dependencies.On("ReadDir", fmt.Sprintf("/sys/block/disk%d/device/scsi_device", diskNum)).Return([]os.FileInfo{&fileInfoMock}, nil).Once()
	dependencies.On("EvalSymlinks", fmt.Sprintf("/dev/disk/by-path/bus-path%d", diskNum)).Return(fmt.Sprintf("/dev/disk/by-path/../../foo/disk%d", diskNum), nil).Once()
	dependencies.On("Abs", fmt.Sprintf("/dev/disk/by-path/../../foo/disk%d", diskNum)).Return(fmt.Sprintf("/dev/foo/disk%d", diskNum), nil).Once()
	dependencies.On("Stat", fmt.Sprintf("/dev/disk/by-path/bus-path%d", diskNum)).Return(nil, nil).Once()
	dependencies.On("Execute", "file", "-s", fmt.Sprintf("/dev/foo/disk%d", diskNum)).Return("Linux rev 1.0 ext4 filesystem data", "", 0).Once()
	dependencies.On("Execute", "smartctl", "--xall", "--json=c", fmt.Sprintf("/dev/foo/disk%d", diskNum)).Return(`{"some": "json"}`, "", 0).Once()

	mockDisk := &ghw.Disk{
		Name:                   fmt.Sprintf("disk%d", diskNum),
		SizeBytes:              5555,
		DriveType:              ghw.DRIVE_TYPE_HDD,
		BusPath:                fmt.Sprintf("bus-path%d", diskNum),
		Vendor:                 fmt.Sprintf("disk%d-vendor", diskNum),
		Model:                  fmt.Sprintf("disk%d-model", diskNum),
		SerialNumber:           fmt.Sprintf("disk%d-serial", diskNum),
		WWN:                    fmt.Sprintf("disk%d-wwn", diskNum),
		BusType:                ghw.BUS_TYPE_SCSI,
		IsRemovable:            false,
		NUMANodeID:             0,
		PhysicalBlockSizeBytes: 512,
		StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
	}

	expectedDisk := &models.Disk{
		ByPath:    fmt.Sprintf("/dev/disk/by-path/bus-path%d", diskNum),
		DriveType: "HDD",
		Hctl:      "scsi",
		Model:     fmt.Sprintf("disk%d-model", diskNum),
		Name:      fmt.Sprintf("disk%d", diskNum),
		Path:      fmt.Sprintf("/dev/foo/disk%d", diskNum),
		Serial:    fmt.Sprintf("disk%d-serial", diskNum),
		SizeBytes: 5555,
		Vendor:    fmt.Sprintf("disk%d-vendor", diskNum),
		Wwn:       fmt.Sprintf("disk%d-wwn", diskNum),
		Bootable:  false,
		Smart:     `{"some": "json"}`,
		InstallationEligibility: models.DiskInstallationEligibility{
			Eligible: true,
		},
	}

	return mockDisk, expectedDisk
}

func prepareDisksTest(dependencies *MockIDependencies, numDisks int) (*ghw.BlockInfo, []*models.Disk) {
	blockInfo := &ghw.BlockInfo{}
	expectedDisks := []*models.Disk{}

	for i := 1; i <= numDisks; i++ {
		ghwDisk, modelsDisk := prepareDiskObjects(dependencies, i)
		blockInfo.Disks = append(blockInfo.Disks, ghwDisk)
		expectedDisks = append(expectedDisks, modelsDisk)
	}

	return blockInfo, expectedDisks
}

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
		var expectation []*models.Disk

		BeforeEach(func() {
			var blockInfo *ghw.BlockInfo
			blockInfo, expectation = prepareDisksTest(dependencies, 1)
			dependencies.On("Block", ghw.WithChroot("/host")).Return(blockInfo, nil).Once()
		})

		It("Bootable", func() {
			deleteExpectedCall(dependencies, "Execute", "file", "-s", "/dev/foo/disk1")
			dependencies.On("Execute", "file", "-s", "/dev/foo/disk1").Return(" DOS/MBR boot sector", "", 0).Once()
			expectation[0].Bootable = true
		})

		It("Non-bootable", func() {
			// No need to change anything, default test disk already tests this.
			// Just make sure that it's truly as we expect
			Expect(expectation[0].Bootable).To(BeFalse())
		})

		It("Without a smartctl error", func() {
			expectation[0].Smart = `{"some": "json"}`
		})

		It("With a smartctl error - make sure JSON is still transmitted", func() {
			deleteExpectedCall(dependencies, "Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1")
			dependencies.On("Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1").Return(`{"some": "json"}`, "", 1).Once()
			expectation[0].Smart = `{"some": "json"}`
		})

		AfterEach(func() {
			ret := GetDisks(dependencies)
			Expect(ret).To(Equal(expectation))
		})
	})

	It("Multiple disks", func() {
		blockInfo, expectedDisks := prepareDisksTest(dependencies, 2)
		dependencies.On("Block", ghw.WithChroot("/host")).Return(blockInfo, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal(expectedDisks))
	})

	It("filters removable disks", func() {
		blockInfo, expectedDisks := prepareDisksTest(dependencies, 1)

		blockInfo.Disks[0].IsRemovable = true
		expectedDisks[0].InstallationEligibility.Eligible = false
		expectedDisks[0].InstallationEligibility.NotEligibleReasons = []string{
			"Disk is removable",
		}

		dependencies.On("Block", ghw.WithChroot("/host")).Return(blockInfo, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal(expectedDisks))
	})

	It("filters ISO disks / marks them as installation media", func() {
		blockInfo, expectedDisks := prepareDisksTest(dependencies, 3)

		blockInfo.Disks[0].Partitions = []*ghw.Partition{
			{
				Disk:       nil,
				Name:       "partition1",
				Label:      "partition1-label",
				MountPoint: "/media/iso",
				SizeBytes:  5555,
				Type:       "ext4",
				IsReadOnly: false,
			},
		}
		expectedDisks[0].InstallationEligibility.Eligible = false
		expectedDisks[0].InstallationEligibility.NotEligibleReasons = []string{
			"Disk appears to be an ISO installation media (has partition with mountpoint suffix iso)",
		}
		expectedDisks[0].IsInstallationMedia = true

		blockInfo.Disks[1].Partitions = []*ghw.Partition{
			{
				Disk:       nil,
				Name:       "partition2",
				Label:      "partition2-label",
				MountPoint: "/some/mount/point",
				SizeBytes:  5555,
				Type:       "iso9660",
				IsReadOnly: false,
			},
		}

		expectedDisks[1].InstallationEligibility.Eligible = false
		expectedDisks[1].InstallationEligibility.NotEligibleReasons = []string{
			"Disk appears to be an ISO installation media (has partition with type iso9660)",
		}
		expectedDisks[1].IsInstallationMedia = true

		// Make sure regular disks don't get marked as installation media
		expectedDisks[2].InstallationEligibility.Eligible = true
		expectedDisks[2].IsInstallationMedia = false

		dependencies.On("Block", ghw.WithChroot("/host")).Return(blockInfo, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal(expectedDisks))
	})

	It("ODD marked as installation media, HDD is not", func() {
		blockInfo, expectedDisks := prepareDisksTest(dependencies, 2)

		blockInfo.Disks[0].DriveType = ghw.DRIVE_TYPE_ODD
		expectedDisks[0].InstallationEligibility.Eligible = true
		expectedDisks[0].IsInstallationMedia = true
		expectedDisks[0].DriveType = "ODD"

		blockInfo.Disks[1].DriveType = ghw.DRIVE_TYPE_HDD
		expectedDisks[1].InstallationEligibility.Eligible = true
		expectedDisks[1].IsInstallationMedia = false
		expectedDisks[1].DriveType = "HDD"

		dependencies.On("Block", ghw.WithChroot("/host")).Return(blockInfo, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal(expectedDisks))
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
				InstallationEligibility: models.DiskInstallationEligibility{
					Eligible: true,
				},
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
				InstallationEligibility: models.DiskInstallationEligibility{
					Eligible: true,
				},
			},
		}))
	})
	It("Fedora 32 DM filter", func() {
		blockInfo, expectation := prepareDisksTest(dependencies, 1)

		blockInfo.Disks[0].StorageController = ghw.STORAGE_CONTROLLER_UNKNOWN
		blockInfo.Disks[0].BusType = ghw.BUS_TYPE_UNKNOWN

		expectation[0].InstallationEligibility.Eligible = false
		expectation[0].InstallationEligibility.NotEligibleReasons = []string{
			"Disk has unknown bus type and storage controller",
		}

		dependencies.On("Block", ghw.WithChroot("/host")).Return(blockInfo, nil).Once()
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal(expectation))
	})
})
