package inventory

import (
	"github.com/jaypipes/ghw"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

// For oVirt VMs the correct platform can be detected only by the Family value,
// this function is used to check that and update the productName accordingly.
func isOVirtPlatform(family string) bool {
	return family == "oVirt" || family == "RHV"
}

// Detect if the machine running in Oracle Cloud Infrastructure
// this function is used to check that and update the manufaturer accordingly.
func isOciPlatform(chassisAssetTag string) bool {
	return chassisAssetTag == "OracleCloud.com"
}

func GetVendor(dependencies util.IDependencies) *models.SystemVendor {
	var ret models.SystemVendor
	var product *ghw.ProductInfo
	var chassis *ghw.ChassisInfo
	var err error

	product, err = dependencies.Product(ghw.WithChroot(dependencies.GetGhwChrootRoot()))
	if err != nil {
		logrus.Errorf("Error running ghw.Product with /host chroot:: %s", err)
		return &ret
	}

	chassis, err = dependencies.Chassis(ghw.WithChroot(dependencies.GetGhwChrootRoot()))
	if err != nil {
		logrus.Errorf("Error running ghw.Chassis with /host chroot:: %s", err)
		return &ret
	}

	ret.SerialNumber = product.SerialNumber
	ret.ProductName = product.Name
	ret.Manufacturer = product.Vendor

	if isOVirtPlatform(product.Family) {
		ret.ProductName = "oVirt"
	}

	if isOciPlatform(chassis.AssetTag) {
		ret.Manufacturer = chassis.AssetTag
	}

	stdout, stderr, exitCode := dependencies.Execute("systemd-detect-virt", "--vm")

	if stderr != "" {
		logrus.Warnf("Error running systemd-detect-virt: %s", stderr)
	}

	if exitCode > 0 {
		return &ret
	}

	if exitCode == 0 && stdout != "none" {
		ret.Virtual = true
	}

	return &ret
}
