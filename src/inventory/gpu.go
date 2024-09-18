package inventory

import (
	"fmt"
	"os"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// PCI classes, combination of base class (2) + subclass (2)
const (
	PCI_CLASS_DISPLAY_VGA            = "0300"
	PCI_CLASS_DISPLAY_3D             = "0302"
	PCI_CLASS_DISPLAY_CONTROLLER     = "0380"
	PCI_CLASS_PROCESSING_ACCELERATOR = "1200"
)

type GPUConfig struct {
	Classes []string
	Models  []string
	Vendors []string
}

// supportedConfig contains supported device configuration
var supportedConfig = GPUConfig{}

// readGpuConfiguration reads a configuration yaml file with the supported GPU IDs
func readGpuConfiguration(path string) (GPUConfig, error) {
	var config GPUConfig
	var defaultConfig = GPUConfig{
		Classes: []string{
			PCI_CLASS_DISPLAY_VGA,
			PCI_CLASS_DISPLAY_3D,
			PCI_CLASS_DISPLAY_CONTROLLER,
			PCI_CLASS_PROCESSING_ACCELERATOR,
		},
		Models:  []string{},
		Vendors: []string{},
	}

	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return defaultConfig, fmt.Errorf("failed to open file: %v", err)
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			return defaultConfig, fmt.Errorf("failed to get file info: %v", err)
		}

		fileSize := fileInfo.Size()
		data := make([]byte, fileSize)
		_, err = file.Read(data)
		if err != nil {
			return defaultConfig, fmt.Errorf("failed to read file: %v", err)
		}

		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return defaultConfig, fmt.Errorf("failed to unmarshal YAML: %v", err)
		}

		return config, nil
	}

	return defaultConfig, nil
}

// isSupportedClass validates if the PCI Class is supported
func isSupportedClass(class string) bool {
	for _, pciId := range supportedConfig.Classes {
		if class == strings.ReplaceAll(pciId, " ", "") {
			return true
		}
	}
	return false
}

// isSupportedVendor validates if the PCI Vendor ID is in the list of supported GPUs
func isSupportedVendor(vendorID string) bool {
	for _, pciId := range supportedConfig.Vendors {
		if vendorID == strings.ReplaceAll(pciId, " ", "") {
			return true
		}
	}
	return false
}

// isSupportedModels validates if the PCI Vendor ID and PCI Device ID are in the list of supported GPUs
func isSupportedModel(deviceID string) bool {
	for _, pciId := range supportedConfig.Models {
		if deviceID == strings.ReplaceAll(pciId, " ", "") {
			return true
		}
	}
	return false
}

// GetGPUs discovers GPU devices on the system
func GetGPUs(subprocessConfig *config.SubprocessConfig, dependencies util.IDependencies) []*models.Gpu {
	gpus := make([]*models.Gpu, 0)

	pciInfo, err := dependencies.PCI()
	if err != nil {
		logrus.Warnf("Error getting PCI info: %s", err)
		return gpus
	}

	supportedConfig, err = readGpuConfiguration(subprocessConfig.GPUConfigFile)
	if err != nil {
		logrus.Warnf("Error getting GPU configuration: %s", err)
		logrus.Info("Using default GPU discovery configuration")
	}

	for _, device := range pciInfo.Devices {
		deviceClass := device.Class.ID + device.Subclass.ID
		deviceVendor := deviceClass + device.Vendor.ID
		deviceModel := deviceVendor + device.Product.ID

		if isSupportedClass(deviceClass) ||
			isSupportedVendor(deviceVendor) ||
			isSupportedModel(deviceModel) {

			gpu := models.Gpu{
				Address:  device.Address,
				Name:     device.Product.Name,
				DeviceID: device.Product.ID,
				Vendor:   device.Vendor.Name,
			}

			if device.Product.VendorID != "" {
				gpu.VendorID = device.Product.VendorID
			} else {
				gpu.VendorID = device.Vendor.ID
			}

			gpus = append(gpus, &gpu)
		}
	}
	return gpus
}
