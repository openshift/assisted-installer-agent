package inventory

import (
	"fmt"
	"strings"

	"github.com/jaypipes/ghw"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
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

func unknownToEmpty(value string) string {
	if value == ghw.UNKNOWN {
		return ""
	}
	return value
}

func (d *disks) getDisks() []*models.Disk {
	ret := make([]*models.Disk, 0)
	blockInfo, err := d.dependencies.Block()
	if err != nil {
		logrus.WithError(err).Warnf("While getting disks info")
		return ret
	}
	for _, disk := range blockInfo.Disks {
		if disk.IsRemovable || disk.SizeBytes == 0 ||
			(disk.BusType == ghw.BUS_TYPE_UNKNOWN && disk.StorageController == ghw.STORAGE_CONTROLLER_UNKNOWN) {
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
		}
		ret = append(ret, &rec)
	}
	return ret
}

func GetDisks(dependencies IDependencies) []*models.Disk {
	return newDisks(dependencies).getDisks()
}
