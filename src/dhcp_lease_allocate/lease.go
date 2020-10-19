package dhcp_lease_allocate

import (
	"net"
	"strconv"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/baremetal-runtimecfg/pkg/monitor"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const DhclientTimeoutSeconds = 5

type VIP struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"mac-address"`
	IpAddress  string `yaml:"ip-address"`
}

func LeaseVIP(log logrus.FieldLogger, cfgPath, masterDevice, name string, mac net.HardwareAddr, ip string) error {
	iface, err := monitor.LeaseInterface(log, masterDevice, name, mac)
	defer deleteInterface(log, name)

	if err != nil {
		log.WithFields(logrus.Fields{
			"masterDevice": masterDevice,
			"name":         name,
		}).WithError(err).Error("Failed to lease interface")
		return err
	}

	leaseFile := monitor.GetLeaseFile(cfgPath, name)

	// -sf avoiding dhclient from setting the received IP to the interface
	// --no-pid in order to allow running multiple `dhclient` simultaneously
	_, stderr, exitCode := util.Execute("timeout", strconv.FormatInt(DhclientTimeoutSeconds, 10), "dhclient", "-v", "-H", name,
		"-sf", "/bin/true", "-lf", leaseFile,
		"--no-pid", "-1", iface.Name)
	switch exitCode {
	case 0:
		return nil
	case 124:
		return errors.Errorf("dhclient was timed out after %d seconds", DhclientTimeoutSeconds)
	default:
		return errors.Errorf("dhclient exited with non-zero exit code %d: %s", exitCode, stderr)
	}
}

func deleteInterface(log logrus.FieldLogger, name string) {
	iface, err := netlink.LinkByName(name)

	if err != nil {
		log.WithError(err).Errorf("deleteInterface: failed to get link by name %s", name)
		return
	}

	if err := netlink.LinkDel(iface); err != nil {
		log.WithError(err).Errorf("deleteInterface: failed to delete link %s", name)
		return
	}
}
