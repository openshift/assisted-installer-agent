package dhcp_lease_allocate

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"

	"github.com/vishvananda/netlink"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/baremetal-runtimecfg/pkg/monitor"

	"github.com/sirupsen/logrus"
)

const configPath string = "/etc/keepalived"

//go:generate mockery --name Dependencies --inpackage
type Dependencies interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
	GetLastLeaseFromFile(log logrus.FieldLogger, fileName string) (string, string, error)
	LeaseInterface(log logrus.FieldLogger, masterDevice string, name string, mac net.HardwareAddr) (*net.Interface, error)
	LinkByName(name string) (netlink.Link, error)
	LinkDel(link netlink.Link) error
	MkdirAll(path string, perm os.FileMode) error
}

type LeaserDependencies struct{}

func (*LeaserDependencies) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

func (*LeaserDependencies) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return os.WriteFile(filename, data, perm)
}

func (*LeaserDependencies) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (*LeaserDependencies) GetLastLeaseFromFile(log logrus.FieldLogger, fileName string) (string, string, error) {
	return monitor.GetLastLeaseFromFile(log, fileName)
}

func (*LeaserDependencies) LeaseInterface(log logrus.FieldLogger, masterDevice string, name string, mac net.HardwareAddr) (*net.Interface, error) {
	return monitor.LeaseInterface(log, masterDevice, name, mac)
}

func (*LeaserDependencies) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}

func (*LeaserDependencies) LinkDel(link netlink.Link) error {
	return netlink.LinkDel(link)
}

func (*LeaserDependencies) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func NewLeaserDependencies() Dependencies {
	return &LeaserDependencies{}
}

type Leaser struct {
	dependecies Dependencies
}

func NewLeaser(dependencies Dependencies) *Leaser {
	return &Leaser{dependecies: dependencies}
}

func (l *Leaser) leaseByMac(log logrus.FieldLogger, cfgPath, masterDevice, name, macString, leaseFileContents string) (strfmt.IPv4, string, error) {
	mac, err := net.ParseMAC(macString)

	if err != nil {
		log.WithFields(logrus.Fields{
			"hw": mac,
		}).WithError(err).Error("Failed to parse mac")
		return "", "", err
	}

	leaseFile := monitor.GetLeaseFile(filepath.Join(cfgPath, name), name)

	if err = LeaseVIP(l.dependecies, log, leaseFile, masterDevice, name, mac, leaseFileContents); err != nil {
		log.WithFields(logrus.Fields{
			"masterDevice": masterDevice,
			"name":         name,
			"mac":          mac,
			"ip":           "",
		}).WithError(err).Error("Failed to lease a vip")

		return "", "", err
	}

	ifaceName, ip, err := l.dependecies.GetLastLeaseFromFile(log, leaseFile)

	if err != nil {
		log.WithFields(logrus.Fields{
			"fileName": leaseFile,
		}).WithError(err).Error("Failed to get last lease from file")
		return "", "", err
	}

	if ifaceName != name {
		log.WithFields(logrus.Fields{
			"expectedInterface": name,
			"actualInterface":   ifaceName,
		}).WithError(err).Error("Interface name is different from expceted")
		return "", "", err
	}

	lastLease, err := extractLastLease(l.dependecies, leaseFile)
	if err != nil {
		log.WithError(err).WithField("lease-file", leaseFile).Error("Could not extract last lease from lease file")
		return "", "", err
	}

	return strfmt.IPv4(ip), lastLease, nil
}

func (l *Leaser) LeaseAllocate(leaseAllocateRequestStr string, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var dhcpAllocationRequest models.DhcpAllocationRequest

	err := json.Unmarshal([]byte(leaseAllocateRequestStr), &dhcpAllocationRequest)
	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocate: json.Unmarshal")
		return "", err.Error(), -1
	}
	err = l.dependecies.MkdirAll(configPath, os.ModePerm)
	if err != nil {
		log.WithError(err).Errorf("Could not mkdir %s", configPath)
		return "", "", -1
	}

	apiVip, apiLease, err := l.leaseByMac(log, configPath, *dhcpAllocationRequest.Interface, "api", dhcpAllocationRequest.APIVipMac.String(), dhcpAllocationRequest.APIVipLease)

	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocate: leaseByMac api")
		return "", err.Error(), -1
	}

	ingressVip, ingressLease, err := l.leaseByMac(log, configPath, *dhcpAllocationRequest.Interface, "ingress", dhcpAllocationRequest.IngressVipMac.String(), dhcpAllocationRequest.IngressVipLease)

	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocate: leaseByMac ingress")
		return "", err.Error(), -1
	}

	dhcpAllocationResponse := models.DhcpAllocationResponse{
		APIVipAddress:     &apiVip,
		IngressVipAddress: &ingressVip,
		APIVipLease:       apiLease,
		IngressVipLease:   ingressLease,
	}

	b, err := json.Marshal(&dhcpAllocationResponse)
	if err != nil {
		log.WithError(err).Error("DhcpLeaseAllocate: json.Marshal")
		return "", err.Error(), -1
	}
	return string(b), "", 0
}
