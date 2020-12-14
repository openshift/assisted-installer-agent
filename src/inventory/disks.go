package inventory

import (
	"fmt"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"

	"github.com/openshift/assisted-service/models"
)

type disks struct {
	dependencies IDependencies
}

func newDisks(dependencies IDependencies) *disks {
	return &disks{dependencies: dependencies}
}

func (d *disks) getPath(busPath string, diskName string) string {
	path := fmt.Sprintf("/dev/%s", diskName)
	_, err := d.dependencies.Stat(path)
	if err == nil {
		return path
	}
	path = fmt.Sprintf("/dev/disk/by-path/%s", busPath)
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
	if busPath != ghw.UNKNOWN {
		path := fmt.Sprintf("/dev/disk/by-path/%s", busPath)
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

func (d *disks) getSMART(path string) string {
	if path == "" {
		return ""
	}

	// We ignore the exit code and stderr because stderr is empty and
	// stdout contains the exit code in `--json=c` mode. Whatever the exit
	// code is, we want to relay the information to the service
	stdout, _, _ := d.dependencies.Execute("smartctl", "--xall", "--json=c", path)

	return stdout
}

func unknownToEmpty(value string) string {
	if value == ghw.UNKNOWN {
		return ""
	}
	return value
}

// diskIsRelevant checks if a disk is relevant for installation by testing
// it against a list of predicates. Also returns all the reasons the disk
// was found to be irrelevant
func diskIsRelevant(disk *ghw.Disk) (relevant bool, skipReasons []string) {
	if disk.IsRemovable {
		skipReasons = append(skipReasons, "Disk is removable")
	}

	if disk.SizeBytes == 0 {
		skipReasons = append(skipReasons, "Disk has size 0")
	}

	if disk.BusType == ghw.BUS_TYPE_UNKNOWN && disk.StorageController == ghw.STORAGE_CONTROLLER_UNKNOWN {
		skipReasons = append(skipReasons, "Disk has unknown bus type and storage controller")
	}

	if disk.DriveType == ghw.DRIVE_TYPE_ODD {
		skipReasons = append(skipReasons, "Disk is an optical disk drive")
	}

	if funk.Contains(funk.Map(disk.Partitions, func(p *ghw.Partition) bool {
		return p.Type == "iso9660"
	}), true) {
		skipReasons = append(skipReasons, "Disk appears to be an ISO installation media (has partition with type iso9660)")
	}

	if funk.Contains(funk.Map(disk.Partitions, func(p *ghw.Partition) bool {
		return strings.HasSuffix(p.MountPoint, "iso")
	}), true) {
		skipReasons = append(skipReasons, "Disk appears to be an ISO installation media (has partition with mountpoint suffix iso)")
	}

	relevant = len(skipReasons) == 0

	return
}

func (d *disks) getDisks() []*models.Disk {
	ret := make([]*models.Disk, 0)
	blockInfo, err := d.dependencies.Block(ghw.WithChroot("/host"))
	if err != nil {
		logrus.WithError(err).Warnf("While getting disks info")
		return ret
	}

	for _, disk := range blockInfo.Disks {
		isRelevant, reasons := diskIsRelevant(disk)
		if !isRelevant {
			reasonsLines := strings.Join(reasons, ", ")
			logrus.Infof(
				"Disk (name %s drive type %v bus path %s vendor %s model %s partitions %s) was found to be irrelevant for the following reasons: %s",
				disk.Name, disk.DriveType.String(), disk.BusPath, disk.Vendor, disk.Model, disk.Partitions, reasonsLines)
			continue
		}

		path := d.getPath(disk.BusPath, disk.Name)
		rec := models.Disk{
			ByPath:    d.getByPath(disk.BusPath),
			Hctl:      d.getHctl(disk.Name),
			Model:     unknownToEmpty(disk.Model),
			Name:      disk.Name,
			Path:      path,
			DriveType: disk.DriveType.String(),
			Serial:    unknownToEmpty(disk.SerialNumber),
			SizeBytes: int64(disk.SizeBytes),
			Vendor:    unknownToEmpty(disk.Vendor),
			Wwn:       unknownToEmpty(disk.WWN),
			Bootable:  d.getBootable(path),
			Smart:     d.getSMART(path),
		}
		ret = append(ret, &rec)
	}
	return ret
}

func GetDisks(dependencies IDependencies) []*models.Disk {
	return newDisks(dependencies).getDisks()
}
