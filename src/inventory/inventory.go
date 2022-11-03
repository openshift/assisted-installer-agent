package inventory

import (
	"encoding/json"
	"net"
	"sort"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

func ReadInventory(subprocessConfig *config.SubprocessConfig, c *Options) *models.Inventory {
	d := util.NewDependencies(&subprocessConfig.DryRunConfig, c.GhwChrootRoot)
	ret := models.Inventory{
		BmcAddress:   GetBmcAddress(subprocessConfig, d),
		BmcV6address: GetBmcV6Address(subprocessConfig, d),
		Boot:         GetBoot(d),
		CPU:          GetCPU(d),
		Disks:        GetDisks(subprocessConfig, d),
		Gpus:         GetGPUs(d),
		Hostname:     GetHostname(d),
		Interfaces:   GetInterfaces(d),
		Memory:       GetMemory(d),
		SystemVendor: GetVendor(d),
		Routes:       GetRoutes(d),
		TpmVersion:   GetTPM(d),
	}
	processInventory(&ret)
	return &ret
}

func CreateInventoryInfo(subprocessConfig *config.SubprocessConfig) []byte {
	in := ReadInventory(subprocessConfig, &Options{GhwChrootRoot: "/host"})

	if subprocessConfig.DryRunEnabled {
		applyDryRunConfig(subprocessConfig, in)
	}

	b, _ := json.Marshal(&in)
	return b
}

type Options struct {
	GhwChrootRoot string
}

// processInventory processes the inventory before sending it to the service. For example, it
// replaces forbidden host names with automatically generated ones.
func processInventory(inventory *models.Inventory) {
	if isForbiddenHostname(inventory.Hostname) {
		calculatedHostname, err := calculateHostname(inventory)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"original": inventory.Hostname,
			}).Error("Failed to generate hostname, will use the original forbidden one")
		} else {
			logrus.WithFields(logrus.Fields{
				"original":   inventory.Hostname,
				"calculated": calculatedHostname,
			}).Info("Replaced original forbidden hostname with calculated one")
			inventory.Hostname = calculatedHostname
		}
	}
}

// forbidenHostnames is the set of host names that are forbidden and need to be replaced with
// automatically generated ones.
var forbiddenHostnames = []string{
	"localhost",
	"localhost.localdomain",
	"localhost4",
	"localhost4.localdomain4",
	"localhost6",
	"localhost6.localdomain6",
}

// isForbiddenHostname checks if the given string is a forbidden host name that needs to be replaced
// with an automatically generated one.
func isForbiddenHostname(hostname string) bool {
	for _, forbiddenHostname := range forbiddenHostnames {
		if hostname == forbiddenHostname {
			return true
		}
	}
	return false
}

// calculateHostname calculates a hostname from the MAC address of one of the network interfaces of
// the given inventory. For example, if the MAC address of the network interface is
// A8:CD:16:AE:79:01 the result will be a8-cd-16-ae-79-01.
func calculateHostname(inventory *models.Inventory) (result string, err error) {
	nic, err := findUsableNIC(inventory)
	if err != nil {
		return
	}
	result = strings.ToLower(strings.ReplaceAll(nic.MacAddress, ":", "-"))
	return
}

// findUsableNIC returns a physical network interface card of the given host inventory that has a
// MAC address and a non-local IP address. Returns nil if the host doesn't have such network
// interface card, and an error if the process fails, for example if some of the IP addresses of the
// host can't be parsed.
func findUsableNIC(inventory *models.Inventory) (result *models.Interface, err error) {
	// Sort the network interfaces by name so that the result will be deterministic:
	nics := make([]*models.Interface, len(inventory.Interfaces))
	copy(nics, inventory.Interfaces)
	sort.Slice(nics, func(i, j int) bool {
		return strings.Compare(nics[i].Name, nics[j].Name) < 0
	})

	// Find the first NIC that has a MAC address an a global IP address:
	for _, nic := range nics {
		isPhysical := nic.Type == "physical"
		hasMAC := nic.MacAddress != ""
		hasV4 := false
		for _, ip := range nic.IPV4Addresses {
			hasV4, err = isGlobalCIDR(ip)
			if err != nil {
				return
			}
			if hasV4 {
				break
			}
		}
		hasV6 := false
		for _, ip := range nic.IPV6Addresses {
			hasV6, err = isGlobalCIDR(ip)
			if err != nil {
				return
			}
			if hasV6 {
				break
			}
		}
		if isPhysical && hasMAC && (hasV4 || hasV6) {
			result = nic
			return
		}
	}
	return
}

// isGlobalCIDR returns a boolean flag indicating if the IP address in the given CIDR is a global
// one and not a local one like 127.0.0.1 or ::1. Returns an error if the given string can't be
// parsed as a CIDR.
func isGlobalCIDR(cidr string) (result bool, err error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return
	}
	result = ip.IsGlobalUnicast()
	return
}
