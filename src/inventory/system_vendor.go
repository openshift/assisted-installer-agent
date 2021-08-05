package inventory

import (
	"github.com/jaypipes/ghw"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"

	"strings"
)

func isVirtual(product string) bool {
	for _, vmTech := range []string{
		"KVM",
		"VirtualBox",
		"VMware",
		"Virtual Machine",
		"AHV",
		"HVM domU",
		"oVirt",
	} {
		if strings.Contains(product, vmTech) {
			return true
		}
	}

	return false
}

// For oVirt VMs the correct platform can be detected only by the Family value,
// this function is used to check that and update the productName accordingly.
func isOVirtPlatform(family string) bool {
	return family == "oVirt" || family == "RHV"
}

func GetVendor(dependencies util.IDependencies) *models.SystemVendor {
	var ret models.SystemVendor

	product, err := dependencies.Product(ghw.WithChroot("/host"))

	if err != nil {
		logrus.Errorf("Error running ghw.Product with /host chroot:: %s", err)
		return &ret
	}

	ret.SerialNumber = product.SerialNumber
	ret.ProductName = product.Name
	ret.Manufacturer = product.Vendor
	if isOVirtPlatform(product.Family){
		ret.ProductName = "oVirt"
	}
	ret.Virtual = isVirtual(ret.ProductName)

	return &ret
}
