package dhcp_lease_allocator

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/openshift/baremetal-runtimecfg/pkg/monitor"
	"github.com/openshift/baremetal-runtimecfg/pkg/utils"
	"github.com/sirupsen/logrus"
)

type VIP struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"mac-address"`
	IpAddress  string `yaml:"ip-address"`
}

func LeaseVIP(log logrus.FieldLogger, cfgPath, masterDevice, name string, mac net.HardwareAddr, ip string) error {
	iface, err := monitor.LeaseInterface(log, masterDevice, name, mac)

	if err != nil {
		log.WithFields(logrus.Fields{
			"masterDevice": masterDevice,
			"name":         name,
		}).WithError(err).Error("Failed to lease interface")
		return err
	}

	leaseFile := monitor.GetLeaseFile(cfgPath, name)

	if f, err := os.OpenFile(leaseFile, os.O_RDWR|os.O_CREATE, 0666); err != nil {
		log.WithFields(logrus.Fields{
			"name": leaseFile,
		}).WithError(err).Error("Failed to create lease file")
		return err
	} else {
		f.Close()
	}

	watcher, err := utils.CreateFileWatcher(log, leaseFile)

	if err != nil {
		log.WithFields(logrus.Fields{
			"filename": leaseFile,
		}).WithError(err).Error("Failed to create a watcher for lease file")
		return err
	}

	// -sf avoiding dhclient from setting the received IP to the interface
	// --no-pid in order to allow running multiple `dhclient` simultaneously
	// -pf allow killing the process
	cmd := exec.Command("dhclient", "-v", iface.Name, "-H", name,
		"-sf", "/bin/true", "-lf", leaseFile, "-d",
		"--no-pid", "-pf", fmt.Sprintf("/var/run/dhclient.%s.pid", iface.Name))
	cmd.Stderr = os.Stderr

	write := make(chan error)
	defer close(write)

	monitor.RunFiniteWatcher(log, watcher, leaseFile, iface.Name, ip, write)

	if err := cmd.Start(); err != nil {
		log.WithFields(logrus.Fields{
			"cmd": cmd.Args,
		}).WithError(err).Error("Failed to execute")
		return err
	}

	if err := <-write; err != nil {
		return err
	}

	return cmd.Process.Kill()
}
