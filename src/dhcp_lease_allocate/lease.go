package dhcp_lease_allocate

import (
	"fmt"
	"net"
	"regexp"
	"strconv"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const DhclientTimeoutSeconds = 28

type VIP struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"mac-address"`
	IpAddress  string `yaml:"ip-address"`
}

func formatLeaseFile(leaseFileContents, interfaceName string) string {
	r := regexp.MustCompile(`interface\s+"[^"]+"`)
	return r.ReplaceAllString(leaseFileContents, fmt.Sprintf(`interface "%s"`, interfaceName))
}

func LeaseVIP(d Dependencies, log logrus.FieldLogger, leaseFile, masterDevice, name string, mac net.HardwareAddr, leaseFileContents string) error {
	iface, err := d.LeaseInterface(log, masterDevice, name, mac)
	defer deleteInterface(d, log, name)

	if err != nil {
		log.WithFields(logrus.Fields{
			"masterDevice": masterDevice,
			"name":         name,
		}).WithError(err).Error("Failed to lease interface")
		return err
	}

	if leaseFileContents != "" {
		err = d.WriteFile(leaseFile, []byte(formatLeaseFile(leaseFileContents, iface.Name)), 0o644)
		if err != nil {
			return errors.Wrapf(err, "Failed to save lease file %s", leaseFile)
		}
	}

	// -sf avoiding dhclient from setting the received IP to the interface
	// --no-pid in order to allow running multiple `dhclient` simultaneously
	_, stderr, exitCode := d.Execute("timeout", strconv.FormatInt(DhclientTimeoutSeconds, 10), "dhclient", "-v", "-H", name,
		"-sf", "/bin/true", "-lf", leaseFile,
		"--no-pid", "-1", iface.Name)
	switch exitCode {
	case 0:
		return nil
	case util.TimeoutExitCode:
		return errors.Errorf("dhclient was timed out after %d seconds", DhclientTimeoutSeconds)
	default:
		return errors.Errorf("dhclient exited with non-zero exit code %d: %s", exitCode, stderr)
	}
}

func deleteInterface(d Dependencies, log logrus.FieldLogger, name string) {
	iface, err := d.LinkByName(name)

	if err != nil {
		log.WithError(err).Errorf("deleteInterface: failed to get link by name %s", name)
		return
	}

	if err := d.LinkDel(iface); err != nil {
		log.WithError(err).Errorf("deleteInterface: failed to delete link %s", name)
		return
	}
}

func extractLastLease(d Dependencies, leaseFile string) (string, error) {
	b, err := d.ReadFile(leaseFile)
	if err != nil {
		return "", errors.Wrapf(err, "Could not read lease file")
	}
	r := regexp.MustCompile(`(?:\A|\s)(lease\s*[{][^}{]*[}])\s*\z`)
	groups := r.FindStringSubmatch(string(b))
	if len(groups) != 2 {
		return "", errors.Errorf("Failed to extract last lease from file %s", leaseFile)
	}
	return groups[1], nil
}
