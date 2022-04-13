package monitor

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/openshift/baremetal-runtimecfg/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"gopkg.in/fsnotify.v1"
	"gopkg.in/yaml.v2"
)

const MonitorConfFileName = "unsupported-monitor.conf"
const leaseFile = "lease-%s"

type vip struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"mac-address"`
	IpAddress  string `yaml:"ip-address"`
}
type yamlVips struct {
	APIVip     *vip `yaml:"api-vip"`
	IngressVip *vip `yaml:"ingress-vip"`
}

func getVipsToLease(cfgPath string) (vips *yamlVips, err error) {
	monitorConfPath := filepath.Join(filepath.Dir(cfgPath), MonitorConfFileName)

	info, err := os.Stat(monitorConfPath)

	if err == nil && info.Mode().IsRegular() {
		log.WithFields(logrus.Fields{
			"file": monitorConfPath,
		}).Info("Monitor conf file exist")

		data, err := ioutil.ReadFile(monitorConfPath)

		if err != nil {
			log.WithFields(logrus.Fields{
				"filename": monitorConfPath,
			}).WithError(err).Error("Failed to read monitor file")
			return nil, err
		}

		return parseMonitorFile(data)
	}
	if os.IsNotExist(err) {
		log.WithFields(logrus.Fields{
			"file": monitorConfPath,
		}).Info("Monitor conf file doesn't exist")
		return nil, nil
	}

	log.WithFields(logrus.Fields{
		"file": monitorConfPath,
	}).WithError(err).Error("Failed to get file status")
	return nil, err
}

func parseMonitorFile(buffer []byte) (*yamlVips, error) {
	var vips yamlVips

	if err := yaml.Unmarshal(buffer, &vips); err != nil {
		log.WithFields(logrus.Fields{
			"buffer": buffer,
		}).WithError(err).Error("Failed to parse monitor file")
		return nil, err
	}

	if vips.APIVip == nil {
		err := fmt.Errorf("APIVip is missing from the yaml content")
		log.Error(err)
		return nil, err
	} else if vips.IngressVip == nil {
		err := fmt.Errorf("IngressVIP is missing from the yaml")
		log.Error(err)
		return nil, err
	}

	log.Info(fmt.Sprintf("Valid monitor file format. APIVip: %+v. IngressVip: %+v", *vips.APIVip, vips.IngressVip))

	return &vips, nil
}

func LeaseVIPs(log logrus.FieldLogger, cfgPath string, vipMasterIface string, vips []vip) error {
	for _, vip := range vips {
		mac, err := net.ParseMAC(vip.MacAddress)

		if err != nil {
			log.WithFields(logrus.Fields{
				"vip": vip,
			}).WithError(err).Error("Failed to parse mac")
			return err
		}

		if err := LeaseVIP(log, cfgPath, vipMasterIface, vip.Name, mac, vip.IpAddress); err != nil {
			log.WithFields(logrus.Fields{
				"masterDevice": vipMasterIface,
				"name":         vip.Name,
				"mac":          mac,
				"ip":           vip.IpAddress,
			}).WithError(err).Error("Failed to lease a vip")
			return err
		}
	}

	return nil
}

func LeaseVIP(log logrus.FieldLogger, cfgPath, masterDevice, name string, mac net.HardwareAddr, ip string) error {
	iface, err := LeaseInterface(log, masterDevice, name, mac)

	if err != nil {
		log.WithFields(logrus.Fields{
			"masterDevice": masterDevice,
			"name":         name,
		}).WithError(err).Error("Failed to lease interface")
		return err
	}

	leaseFile := GetLeaseFile(cfgPath, name)

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
	cmd := exec.Command("dhclient", "-v", iface.Name, "-H", formatHostname(mac.String(), name),
		"-sf", "/bin/true", "-lf", leaseFile, "-d", "--no-pid")
	cmd.Stderr = os.Stderr

	RunInfiniteWatcher(log, watcher, leaseFile, iface.Name, ip)
	return cmd.Start()
}

func formatHostname(mac string, suffix string) string {
	return fmt.Sprintf("%s-%s", strings.ReplaceAll(mac, ":", "-"), suffix)
}

func GetLastLeaseFromFile(log logrus.FieldLogger, fileName string) (string, string, error) {
	data, err := ioutil.ReadFile(fileName)

	if err != nil {
		log.WithFields(logrus.Fields{
			"filename": fileName,
		}).WithError(err).Error("Failed to read lease file")
		return "", "", err
	}

	patternIface := regexp.MustCompile(`\s*interface\s+\"(.+)\";`)
	matchesIface := patternIface.FindAllStringSubmatch(string(data), -1)

	if len(matchesIface) == 0 {
		err := fmt.Errorf("No interfaces in lease file")
		log.WithFields(logrus.Fields{
			"filename": fileName,
		}).Error(err)

		return "", "", err
	}

	patternIp := regexp.MustCompile(`.+fixed-address\s+(.+);`)
	matchesIp := patternIp.FindAllStringSubmatch(string(data), -1)

	if len(matchesIp) == 0 {
		err := fmt.Errorf("No fixed addresses in lease file")
		log.WithFields(logrus.Fields{
			"filename": fileName,
		}).Error(err)

		return "", "", err
	}

	if len(matchesIp) != len(matchesIface) {
		err := fmt.Errorf("Mismatch amount of interfaces and ips")
		log.WithFields(logrus.Fields{
			"matchesIp":    matchesIp,
			"matchesIface": matchesIface,
		}).Error(err)

		return "", "", err
	}

	return matchesIface[len(matchesIface)-1][1], matchesIp[len(matchesIp)-1][1], nil
}

func LeaseInterface(log logrus.FieldLogger, masterDevice string, name string, mac net.HardwareAddr) (*net.Interface, error) {
	// Check if already exist
	if macVlanIfc, err := net.InterfaceByName(name); err == nil {
		return macVlanIfc, nil
	}

	// Read master device
	master, err := netlink.LinkByName(masterDevice)
	if err != nil {
		log.WithFields(logrus.Fields{
			"masterDev": masterDevice,
		}).WithError(err).Error("Failed to read master device")
		return nil, err
	}

	linkAttrs := netlink.LinkAttrs{
		Name:         name,
		ParentIndex:  master.Attrs().Index,
		HardwareAddr: mac,
	}

	mv := &netlink.Macvlan{
		LinkAttrs: linkAttrs,
		Mode:      netlink.MACVLAN_MODE_PRIVATE,
	}

	// Create interface
	if err := netlink.LinkAdd(mv); err != nil {
		log.WithFields(logrus.Fields{
			"masterDev": masterDevice,
			"name":      name,
			"mac":       mac,
		}).WithError(err).Error("Failed to create a macvlan")
		return nil, err
	}

	// Read created link
	macvlanInterfaceLink, err := netlink.LinkByName(name)
	if err != nil {
		log.WithFields(logrus.Fields{
			"name": name,
		}).WithError(err).Error("Failed to read new device")
		return nil, err
	}

	// Bring the interface up
	if err = netlink.LinkSetUp(macvlanInterfaceLink); err != nil {
		log.WithFields(logrus.Fields{
			"interface": name,
		}).WithError(err).Error("Failed to bring interface up")
		return nil, err
	}

	// Read created interface
	macVlanIfc, err := net.InterfaceByName(name)
	if err != nil {
		log.WithFields(logrus.Fields{
			"name": name,
		}).WithError(err).Error("Failed to read new device")
		return nil, err
	}

	return macVlanIfc, nil
}

func RunFiniteWatcher(log logrus.FieldLogger, watcher *fsnotify.Watcher, fileName, expectedIface, expectedIp string, write chan<- error) {
	go func() {
		defer watcher.Close()
		done := false
		var err error

		for !done {
			done, err = utils.RunWatcher(log, watcher, fileName)
		}

		if err == nil {
			err = CheckLastLease(log, fileName, expectedIface, expectedIp)
		}

		write <- err
	}()
}

func RunInfiniteWatcher(log logrus.FieldLogger, watcher *fsnotify.Watcher, fileName, expectedIface, expectedIp string) {
	go func() {
		defer watcher.Close()

		for {
			if done, err := utils.RunWatcher(log, watcher, fileName); done && err == nil {
				_ = CheckLastLease(log, fileName, expectedIface, expectedIp)
			}
		}
	}()
}

func CheckLastLease(log logrus.FieldLogger, fileName, expectedIface, expectedIp string) error {
	if iface, ip, err := GetLastLeaseFromFile(log, fileName); err != nil {
		log.WithFields(logrus.Fields{
			"filename": fileName,
		}).WithError(err).Error("Failed to get lease information from leasing file")
		return err
	} else if iface != expectedIface || (expectedIp != "" && ip != expectedIp) {
		err := fmt.Errorf("A new lease has been written to the lease file with wrong data")
		log.WithFields(logrus.Fields{
			"filename":      fileName,
			"iface":         iface,
			"expectedIface": expectedIface,
			"ip":            ip,
			"expectedIp":    expectedIp,
		}).Error(err)
		return err
	} else {
		log.WithFields(logrus.Fields{
			"fileName": fileName,
			"iface":    iface,
			"ip":       ip,
		}).Info("A new lease has been written to the lease file with the right data")
		return nil
	}
}

func GetLeaseFile(cfgPath, name string) string {
	return filepath.Join(filepath.Dir(cfgPath), fmt.Sprintf(leaseFile, name))
}
