package inventory

import (
	"encoding/json"
	"github.com/filanov/bm-inventory/models"
	"github.com/sirupsen/logrus"
)

type systemVendor struct {
	Product string
	Serial string
	Vendor string
}


func GetVendor(dependencies IDependencies)*models.SystemVendor {
	o, e, exitCode := dependencies.Execute("lshw", "-quiet", "-json")
	var ret models.SystemVendor
	if exitCode != 0 {
		logrus.Errorf("Error running lshw:: %s", e)
		return &ret
	}
	var sv systemVendor
	err := json.Unmarshal([]byte(o), &sv)
	if err != nil {
		logrus.WithError(err).Errorf("Error unmarshaling %s", o)
		return &ret
	}
	ret.SerialNumber = sv.Serial
	ret.ProductName = sv.Product
	ret.Manufacturer = sv.Vendor
	return &ret
}
