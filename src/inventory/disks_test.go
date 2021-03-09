package inventory

import (
	"fmt"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/stretchr/testify/mock"
	"github.com/thoas/go-funk"
	"os"
	"strings"

	"github.com/jaypipes/ghw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func createFakeModelDisk(num int) *models.Disk {
	return &models.Disk{
		ByPath:    fmt.Sprintf("/dev/disk/by-path/bus-path%d", num),
		ID:        fmt.Sprintf("/dev/disk/by-path/bus-path%d", num),
		DriveType: "HDD",
		Hctl:      "0.2.0.0",
		Model:     fmt.Sprintf("disk%d-model", num),
		Name:      fmt.Sprintf("disk%d", num),
		Path:      fmt.Sprintf("/dev/foo/disk%d", num),
		Serial:    fmt.Sprintf("disk%d-serial", num),
		SizeBytes: 5555,
		Vendor:    fmt.Sprintf("disk%d-vendor", num),
		Wwn:       fmt.Sprintf("disk%d-wwn", num),
		Bootable:  false,
		Smart:     `{"some": "json"}`,
		InstallationEligibility: models.DiskInstallationEligibility{
			Eligible: true,
		},
	}
}

func createFakeGHWDisk(num int) *ghw.Disk {
	return &ghw.Disk{
		Name:                   fmt.Sprintf("disk%d", num),
		SizeBytes:              5555,
		DriveType:              ghw.DRIVE_TYPE_HDD,
		BusPath:                fmt.Sprintf("bus-path%d", num),
		Vendor:                 fmt.Sprintf("disk%d-vendor", num),
		Model:                  fmt.Sprintf("disk%d-model", num),
		SerialNumber:           fmt.Sprintf("disk%d-serial", num),
		WWN:                    fmt.Sprintf("disk%d-wwn", num),
		BusType:                ghw.BUS_TYPE_SCSI,
		IsRemovable:            false,
		NUMANodeID:             0,
		PhysicalBlockSizeBytes: 512,
		StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
	}
}

func createNVMEDisk() *ghw.Disk {
	return &ghw.Disk{
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
	}
}

func createAWSXenEBSDisk() *ghw.Disk {
	return &ghw.Disk{
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
	}
}

const sdaWwn = "wwn-0x6141877064533b0020adf3bb03167694"
const sdaPath = "/dev/sda"
const sdbWwn = "wwn-0x6141877064533b0020adf3bc0325d664"
const sdbPath = "/dev/sdb"

func createExpectedSDAModelDisk() *models.Disk {

	return &models.Disk{
		ID:        sdaWwn,
		Bootable:  true,
		ByID:      sdaWwn,
		ByPath:    "/dev/disk/by-path/pci-0000:02:00.0-scsi-0:2:0:0",
		DriveType: "HDD",
		Hctl:      "0.2.0.0",
		InstallationEligibility: models.DiskInstallationEligibility{
			Eligible:           true,
			NotEligibleReasons: nil,
		},
		IoPerf:              nil,
		IsInstallationMedia: false,
		Model:               "PERC_H330_Mini",
		Name:                "sda",
		Path:                sdaPath,
		Serial:              "6141877064533b0020adf3bb03167694",
		SizeBytes:           999653638144,
		Smart:               "{\"some\": \"json\"}",
		Vendor:              "DELL",
		Wwn:                 "0x6141877064533b0020adf3bb03167694",
	}
}

func createExpectedSDBModelDisk() *models.Disk {
	return &models.Disk{
		ID:        sdbWwn,
		Bootable:  true,
		ByID:      sdbWwn,
		ByPath:    "/dev/disk/by-path/pci-0000:02:00.0-scsi-0:2:1:0",
		DriveType: "HDD",
		Hctl:      "0.2.0.0",
		InstallationEligibility: models.DiskInstallationEligibility{
			Eligible:           true,
			NotEligibleReasons: nil,
		},
		IoPerf:              nil,
		IsInstallationMedia: false,
		Model:               "PERC_H330_Mini",
		Name:                "sdb",
		Path:                sdbPath,
		Serial:              "6141877064533b0020adf3bc0325d664",
		SizeBytes:           999653638144,
		Smart:               "{\"some\": \"json\"}",
		Vendor:              "DELL",
		Wwn:                 "0x6141877064533b0020adf3bc0325d664",
	}
}

/**
SDA disk is real disk data from a bare metal machine.
*/
func createSDADisk() *ghw.Disk {
	return &ghw.Disk{
		Name:      "sda",
		SizeBytes: 999653638144,
		DriveType: ghw.DRIVE_TYPE_HDD,
		// run $cat  /sys/block/diskName/dev and then cat /run/udev/data/b{previous command output}
		BusPath:                "pci-0000:02:00.0-scsi-0:2:0:0",
		Vendor:                 "DELL",
		Model:                  "PERC_H330_Mini",
		SerialNumber:           "6141877064533b0020adf3bb03167694",
		WWN:                    "0x6141877064533b0020adf3bb03167694",
		BusType:                ghw.BUS_TYPE_SCSI,
		IsRemovable:            false,
		NUMANodeID:             0,
		PhysicalBlockSizeBytes: 512,
		StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
	}
}

/**
SDB disk is real disk data from a bare metal machine.
*/
func createSDBDisk() *ghw.Disk {
	return &ghw.Disk{
		Name:                   "sdb",
		SizeBytes:              999653638144,
		DriveType:              ghw.DRIVE_TYPE_HDD,
		BusPath:                "pci-0000:02:00.0-scsi-0:2:1:0",
		Vendor:                 "DELL",
		Model:                  "PERC_H330_Mini",
		SerialNumber:           "6141877064533b0020adf3bc0325d664",
		WWN:                    "0x6141877064533b0020adf3bc0325d664",
		BusType:                ghw.BUS_TYPE_SCSI,
		IsRemovable:            false,
		NUMANodeID:             0,
		PhysicalBlockSizeBytes: 512,
		StorageController:      ghw.STORAGE_CONTROLLER_SCSI,
	}
}

func createWWNResults() map[string]string {
	byidmapping := make(map[string]string)
	byidmapping[sdbPath] = sdbWwn
	byidmapping["/dev/sda2"] = "wwn-0x6141877064533b0020adf3bb03167694-part2"
	byidmapping["/dev/sda1"] = "wwn-0x6141877064533b0020adf3bb03167694-part1"
	byidmapping["/dev/sda3"] = "wwn-0x6141877064533b0020adf3bb03167694-part3"
	byidmapping[sdaPath] = sdaWwn
	return byidmapping
}

func mockReadDir(dependencies *MockIDependencies, dir string, errMessage string, files ...os.FileInfo) *mock.Call {
	if errMessage != "" {
		return dependencies.On("ReadDir", dir).Return(nil, errors.New(errMessage)).Once()
	}

	return dependencies.On("ReadDir", dir).Return(files, nil).Once()
}

func mockExecuteDependencyCall(dependencies *MockIDependencies, command string, output string, err string, args ...string) *mock.Call {
	exitCode := 0

	if err != "" {
		exitCode = 1
	}

	interfacesArgs := make([]interface{}, len(args)+1)
	interfacesArgs[0] = command

	for i := range args {
		interfacesArgs[i+1] = args[i]
	}

	return dependencies.On("Execute", interfacesArgs...).Return(output, err, exitCode).Once()
}

func mockStatDependencyCall(dependencies *MockIDependencies, path string, errMessage string) *mock.Call {
	if errMessage != "" {
		return dependencies.On("Stat", path).Return(nil, errors.New(errMessage)).Once()
	} else {
		fileInfoMock := MockFileInfo{}
		fileInfoMock.On("Name").Return(path).Once()
		var info os.FileInfo = &fileInfoMock
		return dependencies.On("Stat", path).Return(info, nil).Once()
	}
}

func mockGetWWNReadDirForSuccess(dependencies *MockIDependencies, results map[string]string) *mock.Call {
	fileInfos := funk.Map(results, func(path string, id string) os.FileInfo {
		fileInfoMock := MockFileInfo{}
		fileInfoMock.On("Name").Return(id).Once()
		fileInfoMock.On("Mode").Return(os.ModeSymlink).Once()
		return &fileInfoMock
	})

	return mockReadDir(dependencies, "/dev/disk/by-id", "", fileInfos.([]os.FileInfo)...)
}

func mockGetWWNCallForSuccess(dependencies *MockIDependencies, results map[string]string) {
	mockGetWWNReadDirForSuccess(dependencies, results)

	funk.ForEach(results, func(path string, id string) {
		if !strings.HasPrefix(id, "wwn-") && !strings.HasPrefix(id, "nvme-eui") {
			return
		}

		incrementFileInfoNameCall(dependencies, id)
		dependencies.On("EvalSymlinks", fmt.Sprintf("/dev/disk/by-id/%s", id)).Return(path, nil).Once()
		dependencies.On("Abs", path).Return(path, nil).Once()
	})
}

func incrementFileInfoNameCall(dependencies *MockIDependencies, id string) {
	_, call := util.GetExpectedCall(&dependencies.Mock, "ReadDir", "/dev/disk/by-id")
	fileInfos := call.ReturnArguments.Get(0)

	for _, fileInfo := range fileInfos.([]os.FileInfo) {
		mockFileInfo := fileInfo.(*MockFileInfo)
		index, call := util.GetExpectedCall(&mockFileInfo.Mock, "Name")

		if index >= 0 && call.ReturnArguments[0].(string) == id {
			util.IncrementCall(&mockFileInfo.Mock, index)
		}
	}
}

func mockFetchDisks(dependencies *MockIDependencies, error error, disks ...*ghw.Disk) {
	dependencies.On("Block", ghw.WithChroot("/host")).Return(&ghw.BlockInfo{Disks: disks}, error).Once()
}

// mockGetPathFromDev Mocks the dependency call that try to locate the disk at /dev/diskName used by disks.getPath.
func mockGetPathFromDev(dependencies *MockIDependencies, diskName string, errMessage string) *mock.Call {
	return mockStatDependencyCall(dependencies, fmt.Sprintf("/dev/%s", diskName), errMessage)
}

// mockGetByPath Mocks the dependency call that try to find the by-path disk name used by disks.getPath.
// The by-path name is the shortest physical path to the device.
// Read this article for more details. https://wiki.archlinux.org/index.php/persistent_block_device_naming
func mockGetByPath(dependencies *MockIDependencies, busPath string, errMessage string) *mock.Call {
	return mockStatDependencyCall(dependencies, fmt.Sprintf("/dev/disk/by-path/%s", busPath), errMessage)
}

func remockGetByPath(dependencies *MockIDependencies, busPath string, errMessage string) *mock.Call {
	path := fmt.Sprintf("/dev/disk/by-path/%s", busPath)
	util.DeleteExpectedMethod(&dependencies.Mock, "Stat", path)
	return mockStatDependencyCall(dependencies, path, errMessage)
}

func mockGetHctl(dependencies *MockIDependencies, name string, err string) *mock.Call {
	fileInfoMock := MockFileInfo{}
	fileInfoMock.On("Name").Return("0.2.0.0").Once()
	var fileInfo os.FileInfo = &fileInfoMock
	return mockReadDir(dependencies, fmt.Sprintf("/sys/block/%s/device/scsi_device", name), err, fileInfo)
}

func mockGetBootable(dependencies *MockIDependencies, path string, bootable bool, err string) *mock.Call {
	result := "DOS/MBR boot sector"

	if !bootable {
		result = "Linux rev 1.0 ext4 filesystem data"
	}

	return mockExecuteDependencyCall(dependencies, "file", result, err, "-s", path)
}

func mockGetSMART(dependencies *MockIDependencies, path string, err string) *mock.Call {
	return mockExecuteDependencyCall(dependencies, "smartctl", `{"some": "json"}`, err, "--xall", "--json=c", path)
}

func mockAllForSuccess(dependencies *MockIDependencies, disks ...*ghw.Disk) {
	mockFetchDisks(dependencies, nil, disks...)

	for _, disk := range disks {
		name := disk.Name
		busPath := disk.BusPath

		mockGetPathFromDev(dependencies, name, "")
		mockGetByPath(dependencies, busPath, "")
		mockGetHctl(dependencies, name, "")
		mockGetBootable(dependencies, fmt.Sprintf("/dev/%s", name), true, "")
		mockGetSMART(dependencies, fmt.Sprintf("/dev/%s", name), "")
	}
}

func prepareDiskObjects(dependencies *MockIDependencies, diskNum int) {
	// Don't find it under /dev/disk1 to test the fallback of searching /dev/disk/by-path
	disk := createFakeGHWDisk(diskNum)
	name := disk.Name
	path := fmt.Sprintf("/dev/foo/disk%d", diskNum)

	mockGetPathFromDev(dependencies, name, "error")
	mockGetHctl(dependencies, name, "")
	mockGetByPath(dependencies, disk.BusPath, "")
	mockGetBootable(dependencies, path, false, "")
	mockGetSMART(dependencies, path, "")

	dependencies.On("EvalSymlinks", fmt.Sprintf("/dev/disk/by-path/bus-path%d", diskNum)).Return(fmt.Sprintf("/dev/disk/by-path/../../foo/disk%d", diskNum), nil).Once()
	dependencies.On("Abs", fmt.Sprintf("/dev/disk/by-path/../../foo/disk%d", diskNum)).Return(path, nil).Once()
}

func prepareDisksTest(dependencies *MockIDependencies, numDisks int) (*ghw.BlockInfo, []*models.Disk) {
	blockInfo := &ghw.BlockInfo{}
	var expectedDisks []*models.Disk
	mockGetWWNCallForSuccess(dependencies, make(map[string]string))

	for i := 1; i <= numDisks; i++ {
		prepareDiskObjects(dependencies, i)
		blockInfo.Disks = append(blockInfo.Disks, createFakeGHWDisk(i))
		expectedDisks = append(expectedDisks, createFakeModelDisk(i))
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
		mockFetchDisks(dependencies, fmt.Errorf("just an error"))
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})

	It("Empty", func() {
		mockFetchDisks(dependencies, nil)
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{}))
	})

	Describe("Single disk", func() {
		var expectation []*models.Disk

		BeforeEach(func() {
			var blockInfo *ghw.BlockInfo
			blockInfo, expectation = prepareDisksTest(dependencies, 1)
			mockFetchDisks(dependencies, nil, blockInfo.Disks...)
		})

		It("Bootable", func() {
			Expect(util.DeleteExpectedMethod(&dependencies.Mock, "Execute", "file", "-s", "/dev/foo/disk1")).To(BeTrue())
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
			Expect(util.DeleteExpectedMethod(&dependencies.Mock, "Execute", "smartctl", "--xall", "--json=c", "/dev/foo/disk1")).To(BeTrue())
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
		mockFetchDisks(dependencies, nil, blockInfo.Disks...)
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

		mockFetchDisks(dependencies, nil, blockInfo.Disks...)
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

		mockFetchDisks(dependencies, nil, blockInfo.Disks...)
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

		mockFetchDisks(dependencies, nil, blockInfo.Disks...)
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
		mockGetWWNCallForSuccess(dependencies, make(map[string]string))
		path := "/dev/xvda"
		disk := createAWSXenEBSDisk()
		mockFetchDisks(dependencies, nil, disk)
		mockGetPathFromDev(dependencies, disk.Name, "")
		mockGetHctl(dependencies, disk.Name, "error")
		mockGetBootable(dependencies, path, true, "")
		mockGetSMART(dependencies, path, "")
		ret := GetDisks(dependencies)

		Expect(ret).To(Equal([]*models.Disk{
			{
				ID:        "/dev/xvda",
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
		mockGetWWNCallForSuccess(dependencies, make(map[string]string))
		disk := createNVMEDisk()
		path := "/dev/nvme0n1"
		mockFetchDisks(dependencies, nil, disk)
		mockGetPathFromDev(dependencies, disk.Name, "")
		mockGetHctl(dependencies, disk.Name, "error")
		mockGetBootable(dependencies, path, false, "")
		mockGetByPath(dependencies, disk.BusPath, "")
		mockGetSMART(dependencies, path, "")
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal([]*models.Disk{
			{
				ID:        "/dev/disk/by-path/pci-0000:3d:00.0-nvme-1",
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

		mockFetchDisks(dependencies, nil, blockInfo.Disks...)
		ret := GetDisks(dependencies)
		Expect(ret).To(Equal(expectation))
	})

	Describe("By-Id", func() {
		Specify("GetDisk does not affect from failures while fetching the disk WWN - Read dir failed", func() {
			mockReadDir(dependencies, "/dev/disk/by-id", "fetching the by-id disk failed")
			mockAllForSuccess(dependencies, createSDADisk(), createSDBDisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(2))

			for _, disk := range ret {
				Ω(disk.ByID).Should(BeEmpty())
			}
		})

		Specify("GetDisk does not affect from failures while fetching the disk WWN - Eval symlink failed", func() {
			mockGetWWNReadDirForSuccess(dependencies, createWWNResults())

			funk.ForEach(createWWNResults(), func(path string, id string) {
				if !strings.HasPrefix(id, "wwn-") && !strings.HasPrefix(id, "nvme-eui") {
					return
				}
				incrementFileInfoNameCall(dependencies, id)
				name := fmt.Sprintf("/dev/disk/by-id/%s", id)

				if path == sdbPath {
					dependencies.On("EvalSymlinks", name).Return("", errors.New("Error")).Once()
				} else {
					dependencies.On("EvalSymlinks", name).Return(path, nil).Once()
					dependencies.On("Abs", path).Return(path, nil).Once()
				}
			})

			mockAllForSuccess(dependencies, createSDADisk(), createSDBDisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(2))

			for _, disk := range ret {
				if disk.Path == sdbPath {
					Ω(disk.ByID).Should(BeEmpty())
				} else {
					Ω(disk.ByID).ShouldNot(BeEmpty())
				}
			}
		})

		Specify("GetDisk does not affect from failures while fetching the disk WWN - Abs call failed", func() {
			mockGetWWNReadDirForSuccess(dependencies, createWWNResults())

			funk.ForEach(createWWNResults(), func(path string, id string) {
				if !strings.HasPrefix(id, "wwn-") && !strings.HasPrefix(id, "nvme-eui") {
					return
				}

				incrementFileInfoNameCall(dependencies, id)
				dependencies.On("EvalSymlinks", fmt.Sprintf("/dev/disk/by-id/%s", id)).Return(path, nil).Once()

				if path == sdbPath {
					dependencies.On("Abs", path).Return("", errors.New("Error")).Once()
				} else {
					dependencies.On("Abs", path).Return(path, nil).Once()
				}
			})

			mockAllForSuccess(dependencies, createSDADisk(), createSDBDisk())
			ret := GetDisks(dependencies)

			Ω(ret).Should(HaveLen(2))

			for _, disk := range ret {
				if disk.Path == sdbPath {
					Ω(disk.ByID).Should(BeEmpty())
				} else {
					Ω(disk.ByID).ShouldNot(BeEmpty())
				}
			}

		})

		Specify("GetDisk does not affect from empty by id information", func() {
			byidmapping := make(map[string]string)
			byidmapping["path"] = "id"
			mockGetWWNCallForSuccess(dependencies, byidmapping)
			mockAllForSuccess(dependencies, createSDADisk(), createSDBDisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(2))

			for _, disk := range ret {
				Ω(disk.ByID).Should(BeEmpty())
			}
		})

		It("Should have the by-id information", func() {
			mockGetWWNCallForSuccess(dependencies, createWWNResults())
			mockAllForSuccess(dependencies, createSDADisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(1))
			disk := ret[0]
			Ω(disk.ByID).Should(Equal(sdaWwn))
		})

		It("Should have the by-id information only for one disk", func() {
			results := createWWNResults()
			delete(results, sdbPath)
			mockGetWWNCallForSuccess(dependencies, results)
			mockAllForSuccess(dependencies, createSDADisk(), createSDBDisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(2))

			for _, disk := range ret {
				if disk.Name == createSDBDisk().Name {
					Ω(disk.ByID).Should(BeEmpty())
				} else {
					Ω(disk.ByID).Should(Equal(sdaWwn))
				}
			}
		})

		It("Should have the by-id information for all the disks", func() {
			mockGetWWNCallForSuccess(dependencies, createWWNResults())
			mockAllForSuccess(dependencies, createSDADisk(), createSDBDisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(2))

			for _, disk := range ret {
				if disk.Name == createSDBDisk().Name {
					Ω(disk.ByID).Should(Equal(sdbWwn))
				} else {
					Ω(disk.ByID).Should(Equal(sdaWwn))
				}
			}
		})

		It("Should have the by-id information for nvme disk", func() {
			byidmapping := make(map[string]string)
			byidmapping["/dev/nvme0n1"] = "nvme-eui-0x6141877064533b0020adf3bc0325d664"
			mockGetWWNCallForSuccess(dependencies, byidmapping)
			mockAllForSuccess(dependencies, createNVMEDisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(1))
			disk := ret[0]
			Ω(disk.ByID).Should(Equal("nvme-eui-0x6141877064533b0020adf3bc0325d664"))
		})

		It("All the other fields are the same", func() {
			mockGetWWNCallForSuccess(dependencies, createWWNResults())
			mockAllForSuccess(dependencies, createSDADisk(), createSDBDisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(2))
			Expect(ret).Should(ConsistOf(createExpectedSDAModelDisk(), createExpectedSDBModelDisk()))
		})

		It("Id equals to the disk path", func() {
			mockGetWWNCallForSuccess(dependencies, make(map[string]string))
			sdaDisk := createSDADisk()
			mockAllForSuccess(dependencies, sdaDisk)
			remockGetByPath(dependencies, sdaDisk.BusPath, "Error")

			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(1))
			disk := ret[0]
			Ω(disk.ByID).Should(BeEmpty())
			Ω(disk.ID).Should(Equal(disk.Path))
		})

		It("Id equals to the disk by-path field", func() {
			mockGetWWNCallForSuccess(dependencies, make(map[string]string))
			mockAllForSuccess(dependencies, createSDADisk())
			ret := GetDisks(dependencies)
			Ω(ret).Should(HaveLen(1))
			disk := ret[0]
			Ω(disk.ByID).Should(BeEmpty())
			Ω(disk.ID).Should(Equal(disk.ByPath))
		})
	})
})
