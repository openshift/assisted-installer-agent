package inventory

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

type lshw struct {
	Product string `json:"product,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
	Clock   int64  `json:"clock,omitempty"`
	BusInfo string `json:"businfo,omitempty"`
}

func GetGPUs(dependencies util.IDependencies) []*models.Gpu {
	gpus := make([]*models.Gpu, 0)
	o, e, exitCode := dependencies.Execute("lshw", "-class", "display", "-json", "-numeric")
	if exitCode != 0 {
		logrus.Warnf("Error running lshw: %s", e)
		return gpus
	}
	var l []lshw
	o = sanitizeLshwJsonOutput(o)
	err := json.Unmarshal([]byte(o), &l)
	if err != nil {
		logrus.WithError(err).Warnf("Error unmarshalling lshw: %s", o)
		return gpus
	}

	for _, details := range l {
		gpu := models.Gpu{
			BusInfo: details.BusInfo,
			ClockHz: details.Clock,
		}
		addDeviceAndVendorInfo(&gpu, details)
		gpus = append(gpus, &gpu)
	}

	return gpus
}

func addDeviceAndVendorInfo(gpu *models.Gpu, details lshw) {
	productRegex := regexp.MustCompile(`(.*) \[([a-fA-F0-9:]+)]`)
	productMatches := productRegex.FindStringSubmatch(details.Product)
	gpu.Name = details.Product
	if len(productMatches) == 3 {
		gpu.Name = productMatches[1]

		vendorDeviceID := productMatches[2]
		ids := strings.Split(vendorDeviceID, ":")
		if len(ids) == 2 {
			gpu.VendorID = ids[0]
			gpu.DeviceID = ids[1]
		}
	}

	vendorRegex := regexp.MustCompile(`(.*) \[([a-fA-F0-9]+)]`)
	vendorMatches := vendorRegex.FindStringSubmatch(details.Vendor)
	gpu.Vendor = details.Vendor
	if len(vendorMatches) == 3 {
		gpu.Vendor = vendorMatches[1]
	}
}

// sanitizeLshwJsonOutput works around https://ezix.org/project/ticket/631 issue. The tool may return multiple
// JSON objects not wrapped in an array and the last object may be followed by a comma, i.e:
func sanitizeLshwJsonOutput(o string) string {
	if len(o) == 0 {
		return o
	}
	o = strings.TrimSpace(o)
	o = strings.TrimSuffix(o, ",")
	if o[0] == '{' {
		o = "[" + o + "]"
	}
	return o
}
