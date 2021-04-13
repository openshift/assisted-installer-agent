package inventory

import (
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

func GetGPUs(dependencies util.IDependencies) []*models.Gpu {
	gpus := make([]*models.Gpu, 0)
	gpuInfo, err := dependencies.GPU()
	if err != nil {
		logrus.Warnf("Error getting GPU info: %s", err)
		return gpus
	}

	for _, card := range gpuInfo.GraphicsCards {
		gpu := models.Gpu{
			Address: card.Address,
		}
		if card.DeviceInfo != nil {
			if card.DeviceInfo.Product != nil {
				gpu.Name = card.DeviceInfo.Product.Name
				gpu.DeviceID = card.DeviceInfo.Product.ID
				gpu.VendorID = card.DeviceInfo.Product.VendorID
			}
			if card.DeviceInfo.Vendor != nil {
				gpu.Vendor = card.DeviceInfo.Vendor.Name
				if gpu.VendorID == "" {
					gpu.VendorID = card.DeviceInfo.Vendor.ID
				}
			}
		}
		gpus = append(gpus, &gpu)
	}
	return gpus
}
