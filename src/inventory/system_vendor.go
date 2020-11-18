package inventory

import (
	"github.com/jaypipes/ghw"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"

	"strings"
)

func isVirtual(product string) bool {
	for _, vmTech := range []string {
		"KVM",
		"VirtualBox",
		"VMware",
		"Virtual Machine",
	} {
		if strings.Contains(product, vmTech) {
			return true
		}
	}

	return false
}

func GetVendor(dependencies IDependencies) *models.SystemVendor {
	var ret models.SystemVendor

	product, err := dependencies.Product(ghw.WithChroot("/host"))

	if err != nil {
		logrus.Errorf("Error running ghw.Product with /host chroot:: %s", err)
		return &ret
	}

	ret.SerialNumber = product.SerialNumber
	ret.ProductName = product.Name
	ret.Manufacturer = product.Vendor
	ret.Virtual = isVirtual(product.Name)

	return &ret
}
