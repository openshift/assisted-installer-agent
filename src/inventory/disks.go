package inventory

import (
	"fmt"

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

func (d *disks) getPath(busPath string) string {
	path := fmt.Sprintf("/dev/disk/by-path/%s", busPath)
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

func (d *disks) getHctl(name string) string {
	dir := fmt.Sprintf("/sys/block/%s/device/scsi_device", name)
	files, err := d.dependencies.ReadDir(dir)
	if err != nil || len(files) == 0 {
		return ""
	}
	return files[0].Name()
}

func diskVendor(vendor string) string {
	if vendor == ghw.UNKNOWN {
		return ""
	}
	return vendor
}

func (d *disks) getDisks() []*models.Disk {
	ret := make([]*models.Disk, 0)
	blockInfo, err := d.dependencies.Block()
	if err != nil {
		logrus.WithError(err).Warnf("While getting disks info")
		return ret
	}
	for _, disk := range blockInfo.Disks {
		if disk.IsRemovable || disk.BusPath == ghw.UNKNOWN {
			continue
		}
		rec := models.Disk{
			ByPath:    fmt.Sprintf("/dev/disk/by-path/%s", disk.BusPath),
			Hctl:      d.getHctl(disk.Name),
			Model:     disk.Model,
			Name:      disk.Name,
			Path:      d.getPath(disk.BusPath),
			DriveType: disk.DriveType.String(),
			Serial:    disk.SerialNumber,
			SizeBytes: int64(disk.SizeBytes),
			Vendor:    diskVendor(disk.Vendor),
			Wwn:       disk.WWN,
		}
		ret = append(ret, &rec)
	}
	return ret
}

func GetDisks(dependencies IDependencies) []*models.Disk {
	return newDisks(dependencies).getDisks()
}
