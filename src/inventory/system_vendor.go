package inventory

import (
	"strings"

	"github.com/jaypipes/ghw"
	ghwutil "github.com/jaypipes/ghw/pkg/util"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

const (
	VENDOR_ID   = "vendor_id"
	VM_CTRL_PRG = "VM.*Control Program"
	CTRL_PRG    = "Control Program"
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

	// Check if Manufacturer is unknown (valid for s390x)
	if product.Vendor != ghwutil.UNKNOWN {
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
	} else {
		// parse host /proc/cpuinfo and /proc/sysinfo
		stdout, stderr, exitCode := dependencies.Execute("grep", VENDOR_ID, "/proc/cpuinfo")

		// it makes no sense to continue in case of an error
		if exitCode != 0 {
			logrus.Warnf("Error running grep %s /proc/cpuinfo: %s", VENDOR_ID, stderr)
			return &ret
		}

		for _, part := range strings.Split(strings.TrimSpace(string(stdout)), ":") {
			if !(strings.HasPrefix(part, VENDOR_ID)) {
				ret.Manufacturer = strings.TrimSpace(part)
			}
		}

		// get virtualization type
		stdout, stderr, exitCode = dependencies.Execute("grep", VM_CTRL_PRG, "/proc/sysinfo")

		// it makes no sense to continue in case of an error ... same as above
		if exitCode != 0 {
			logrus.Warnf("Error running grep %s /proc/sysinfo: %s", VM_CTRL_PRG, stderr)
			return &ret
		}

		for _, part := range strings.Split(strings.TrimSpace(string(stdout)), ":") {
			if !(strings.HasPrefix(part, CTRL_PRG)) {
				ret.ProductName = strings.TrimSpace(part)
			}
		}
	}

	return &ret
}
