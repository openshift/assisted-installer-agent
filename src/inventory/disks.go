package inventory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	ghwutil "github.com/jaypipes/ghw/pkg/util"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

const (
	// Arbitrary SMART result to use in dry-run mode, the content doesn't really matter
	dryRunSmart = `{"json_format_version":[1,0],"smartctl":{"version":[7,1],"svn_revision":"5022","platform_info":"x86_64-linux-5.14.0-60.fc35.x86_64","build_info":"(local build)","argv":["smartctl","--xall","--json=c","/dev/vda"],"messages":[{"string":"/dev/vda: Unable to detect device type","severity":"error"}],"exit_status":1}}`
)

type disks struct {
	dependencies     util.IDependencies
	subprocessConfig *config.SubprocessConfig
}

func newDisks(subprocessConfig *config.SubprocessConfig, dependencies util.IDependencies) *disks {
	return &disks{dependencies: dependencies, subprocessConfig: subprocessConfig}
}

func (d *disks) getDisksWWNs() map[string]string {
	const ByIdLocation = "/dev/disk/by-id"
	filesInfo, err := d.dependencies.ReadDir(ByIdLocation)

	if err != nil {
		logrus.Warnf("Cannot get disk/by-id information: %s", err)
		return make(map[string]string)
	}

	matchingFiles := funk.Filter(filesInfo, func(fileInfo os.FileInfo) bool {
		basename := filepath.Base(fileInfo.Name())

		if !strings.HasPrefix(basename, "wwn-") && !strings.HasPrefix(basename, "nvme-eui") {
			return false
		}

		return fileInfo.Mode()&os.ModeSymlink != 0
	})

	// Finding the disk path (/dev/sda) from the by-id symlink.
	// For example: wwn-0x6141877064533b0020adf3bc0325d664	-> /dev/sdb
	// "wwn-0x6141877064533b0020adf3bc0325d664" is the disk id and the path is: /dev/sdb
	return funk.Map(matchingFiles, func(fileInfo os.FileInfo) (string, string) {
		diskId := filepath.Join(ByIdLocation, fileInfo.Name())
		diskPath, err := d.dependencies.EvalSymlinks(diskId)

		if err != nil {
			logrus.WithError(err).Warnf("Cannot resolve disk path from the disk by-id information (disk id is [%s]) - skipping:", diskId)
			return "", ""
		}

		diskPath, err = d.dependencies.Abs(diskPath)

		if err != nil {
			logrus.WithError(err).Warnf("Cannot resolve disk path from the disk by-id information (disk id is [%s]) - skipping:", diskId)
			return "", ""
		}

		return diskPath, diskId
	}).(map[string]string)
}

func (d *disks) getPath(busPath string, diskName string) string {
	path := filepath.Join("/dev", diskName)
	_, err := d.dependencies.Stat(path)
	if err == nil {
		return path
	}
	path = filepath.Join("/dev/disk/by-path", busPath)
	evaledPath, err := d.dependencies.EvalSymlinks(path)
	if err != nil {
		logrus.WithError(err).Warn("EvalSymlink")
		return ""
	}
	ret, err := d.dependencies.Abs(evaledPath)
	if err != nil {
		logrus.WithError(err).Warn("Abs")
		return ""
	}
	return ret
}

func (d *disks) getByPath(busPath string) string {
	if busPath != ghwutil.UNKNOWN {
		path := filepath.Join("/dev/disk/by-path", busPath)
		_, err := d.dependencies.Stat(path)
		if err == nil {
			return path
		}
	}
	return ""
}

func (d *disks) getHctl(name string) string {
	dir := fmt.Sprintf("/sys/block/%s/device/scsi_device", name)
	files, err := d.dependencies.ReadDir(dir)
	if err != nil || len(files) == 0 {
		return ""
	}
	return files[0].Name()
}

func (d *disks) getBootable(path string) bool {
	if path == "" {
		return false
	}
	stdout, stderr, exitCode := d.dependencies.Execute("file", "-s", path)
	if exitCode != 0 {
		logrus.Warnf("Could not get bootable information for path %s: %s", path, stderr)
		return false
	}
	return strings.Contains(stdout, "DOS/MBR boot sector")
}

func (d *disks) hasUUID(path string) bool {
	if path == "" {
		return false
	}

	// Openshift requires VSphere (and maybe other platforms) VMs to have a UUID for disks
	// Here we add this information to the inventory for the assisted-service verification.
	// Getting device page 83 which contains the device WWID(disk.UUID)
	// please see: https://access.redhat.com/solutions/93943
	stdout, stderr, exitCode := d.dependencies.Execute("sg_inq", "-p", "0x83", path)

	logrus.Debugf("UUID information for path %s: exit code %d, stdout: %s\n, stderr: %s", path, exitCode, stdout, stderr)

	if exitCode != 0 {
		logrus.Infof("hasUUID is false for path %s: exit code %d, stdout: %s\n, stderr: %s", path, exitCode, stdout, stderr)
		return false
	}

	return true
}

func (d *disks) getSMART(path string) string {
	if path == "" {
		return ""
	}

	if d.subprocessConfig.DryRunEnabled {
		// smartctl is rather slow, better to avoid it dry run mode
		return dryRunSmart
	}

	// We ignore the exit code and stderr because stderr is empty and
	// stdout contains the exit code in `--json=c` mode. Whatever the exit
	// code is, we want to relay the information to the service
	stdout, _, _ := d.dependencies.Execute("smartctl", "--all", "--quiet=errorsonly", path)

	return stdout
}

func unknownToEmpty(value string) string {
	if value == ghwutil.UNKNOWN {
		return ""
	}
	return value
}

func isDeviceMapper(disk *ghw.Disk) bool {
	return strings.HasPrefix(disk.Name, "dm-")
}

func isISCSIDisk(disk *ghw.Disk) bool {
	return strings.Contains(disk.BusPath, "-iscsi-")
}

func isFCDisk(disk *ghw.Disk) bool {
	return strings.Contains(disk.BusPath, "-fc-")
}

func (d *disks) dmUUIDHasPrefix(disk *ghw.Disk, prefix string) bool {
	if !isDeviceMapper(disk) {
		return false
	}
	path := filepath.Join("/sys", "block", disk.Name, "dm", "uuid")
	b, err := d.dependencies.ReadFile(path)
	if err != nil {
		logrus.WithError(err).Warnf("Failed reading dm uuid %s", path)
		return false
	}
	if strings.HasPrefix(string(b), prefix) {
		return true
	}
	return false
}

func (d *disks) isMultipath(disk *ghw.Disk) bool {
	return d.dmUUIDHasPrefix(disk, "mpath-")
}

func (d *disks) isLVM(disk *ghw.Disk) bool {
	return d.dmUUIDHasPrefix(disk, "LVM-")
}

func (d *disks) getHolders(diskName string) string {
	dir := fmt.Sprintf("/sys/block/%s/holders", diskName)
	files, err := d.dependencies.ReadDir(dir)
	if err != nil {
		logrus.WithError(err).Warnf("Failed listing device holders %s", dir)
		return ""
	}
	if len(files) == 0 {
		return ""
	}
	var holders []string
	for _, file := range files {
		holders = append(holders, file.Name())
	}
	return strings.Join(holders, ",")
}

// checkEligibility checks if a disk is eligible for installation by testing
// it against a list of predicates. Returns all the reasons the disk
// was found to be not eligible, or an empty slice if it was found to
// be eligible. Also returns whether the disk appears to be an installation
// media or not.
func (d *disks) checkEligibility(disk *ghw.Disk) (notEligibleReasons []string, isInstallationMedia bool) {
	if disk.IsRemovable {
		notEligibleReasons = append(notEligibleReasons, "Disk is removable")
	}

	if disk.StorageController == ghw.STORAGE_CONTROLLER_UNKNOWN && !d.isMultipath(disk) && !d.isLVM(disk) {
		notEligibleReasons = append(notEligibleReasons, "Disk has unknown storage controller")
	}

	if d.isLVM(disk) {
		notEligibleReasons = append(notEligibleReasons, "Disk is an LVM logical volume")
	}

	if funk.Contains(funk.Map(disk.Partitions, func(p *ghw.Partition) bool {
		return p.Type == "iso9660"
	}), true) {
		notEligibleReasons = append(notEligibleReasons, "Disk appears to be an ISO installation media (has partition with type iso9660)")
		isInstallationMedia = true
	}

	if funk.Contains(funk.Map(disk.Partitions, func(p *ghw.Partition) bool {
		return strings.HasSuffix(p.MountPoint, "iso")
	}), true) {
		notEligibleReasons = append(notEligibleReasons, "Disk appears to be an ISO installation media (has partition with mountpoint suffix iso)")
		isInstallationMedia = true
	}

	return notEligibleReasons, isInstallationMedia
}

func (d *disks) shouldReturnDisk(disk *block.Disk) bool {
	return !((strings.HasPrefix(disk.Name, "dm-") && !(d.isMultipath(disk) || d.isLVM(disk))) || // Device mapper devices, except multipath/LVM
		strings.HasPrefix(disk.Name, "loop") || // Loop devices (see `man loop`)
		strings.HasPrefix(disk.Name, "zram") || // Default name usually assigned to "swap on ZRAM" block devices
		strings.HasPrefix(disk.Name, "md")) // Linux multiple-device-driver block devices
}

func (d *disks) getDriveType(disk *block.Disk) models.DriveType {
	diskString := disk.DriveType.String()
	var driveType models.DriveType

	if isISCSIDisk(disk) {
		driveType = models.DriveTypeISCSI
	} else if isFCDisk(disk) {
		driveType = models.DriveTypeFC
	} else if d.isMultipath(disk) {
		driveType = models.DriveTypeMultipath
	} else if d.isLVM(disk) {
		driveType = models.DriveTypeLVM
	} else if diskString == ghw.DRIVE_TYPE_FDD.String() {
		driveType = models.DriveTypeFDD
	} else if diskString == ghw.DRIVE_TYPE_HDD.String() {
		driveType = models.DriveTypeHDD
	} else if diskString == ghw.DRIVE_TYPE_ODD.String() {
		driveType = models.DriveTypeODD
	} else if diskString == ghw.DRIVE_TYPE_SSD.String() {
		driveType = models.DriveTypeSSD
	} else {
		driveType = models.DriveTypeUnknown
	}

	return driveType
}

func (d *disks) getDisks() []*models.Disk {
	ret := make([]*models.Disk, 0)
	var blockInfo *ghw.BlockInfo
	var err error
	blockInfo, err = d.dependencies.Block(ghw.WithChroot(d.dependencies.GetGhwChrootRoot()))
	if err != nil {
		logrus.WithError(err).Warnf("While getting disks info")
		return ret
	}

	if len(blockInfo.Disks) == 0 {
		return ret
	}

	diskPath2diskWWN := d.getDisksWWNs()

	for diskIndex, disk := range blockInfo.Disks {
		var eligibility models.DiskInstallationEligibility
		var isInstallationMedia bool

		// Filter out disks that we don't want to return
		if !d.shouldReturnDisk(disk) {
			continue
		}

		eligibility.NotEligibleReasons, isInstallationMedia = d.checkEligibility(disk)
		eligibility.Eligible = len(eligibility.NotEligibleReasons) == 0

		// Optical disks should also be considered installation media
		isInstallationMedia = isInstallationMedia || (disk.DriveType == ghw.DRIVE_TYPE_ODD)

		if !eligibility.Eligible {
			reasons := strings.Join(eligibility.NotEligibleReasons, ", ")
			logrus.Infof(
				"Disk (name %s drive type %v bus path %s vendor %s model %s partitions %s) was found to be ineligible for installation for the following reasons: %s",
				disk.Name, disk.DriveType.String(), disk.BusPath, disk.Vendor, disk.Model, disk.Partitions, reasons)
		}

		path := d.getPath(disk.BusPath, disk.Name)

		rec := models.Disk{
			ByID:                    diskPath2diskWWN[path],
			ByPath:                  d.getBusPath(blockInfo.Disks, diskIndex, disk.BusPath),
			Hctl:                    d.getHctl(disk.Name),
			Model:                   unknownToEmpty(disk.Model),
			Name:                    disk.Name,
			Path:                    path,
			DriveType:               d.getDriveType(disk),
			Serial:                  unknownToEmpty(disk.SerialNumber),
			SizeBytes:               int64(disk.SizeBytes),
			Vendor:                  unknownToEmpty(disk.Vendor),
			Wwn:                     unknownToEmpty(disk.WWN),
			Bootable:                d.getBootable(path),
			Removable:               disk.IsRemovable,
			Smart:                   d.getSMART(path),
			IsInstallationMedia:     isInstallationMedia,
			InstallationEligibility: eligibility,
			HasUUID:                 d.hasUUID(path),
			Holders:                 d.getHolders(disk.Name),
		}

		rec.ID = rec.Path

		if rec.ByID != "" {
			rec.ID = rec.ByID
		} else if rec.ByPath != "" {
			rec.ID = rec.ByPath
		}

		ret = append(ret, &rec)
	}
	return ret
}

// getBusPath - Support special case where two disk have the same busType.
// We reproduce this case by creating a machine using virt-install with the following command which creates a machine
// with two IDE disks HDD and CDROM and both have the same busType.
// virt-install --name=master-1 --cdrom=$PATH/cluster-discovery.iso --vcpus=2 --ram=16384 --disk=size="$DISKGIB",pool="$POOL" --os-variant=rhel-unknown --network=bridge=virbr0,model=virtio --graphics=none --noautoconsole
func (d *disks) getBusPath(disks []*block.Disk, index int, busPath string) string {
	if busPath == ghwutil.UNKNOWN {
		return ""
	}

	for i, disk := range disks {
		if i == index || disk.BusPath != busPath {
			continue
		}

		// when two disks share the same bus path, we prefer to pretend they have no bus path at all, to avoid confusing between them
		return ""
	}

	return d.getByPath(busPath)
}

func GetDisks(subprocessConfig *config.SubprocessConfig, dependencies util.IDependencies) []*models.Disk {
	return newDisks(subprocessConfig, dependencies).getDisks()
}
