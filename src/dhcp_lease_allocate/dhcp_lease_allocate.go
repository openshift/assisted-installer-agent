package dhcp_lease_allocate

import (
	"encoding/json"
	"net"

	"github.com/go-openapi/strfmt"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/baremetal-runtimecfg/pkg/monitor"

	"github.com/sirupsen/logrus"
)

const configPath string = "/tmp/config.path"

//go:generate mockery -name Executer -inpkg
type Executer interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
}

type ProcessExecuter struct{}

func (e *ProcessExecuter) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

func LeaseByMac(log logrus.FieldLogger, cfgPath, masterDevice, name, macString string) (strfmt.IPv4, error) {
	mac, err := net.ParseMAC(macString)

	if err != nil {
		log.WithFields(logrus.Fields{
			"hw": mac,
		}).WithError(err).Error("Failed to parse mac")
		return "", err
	}

	if err := LeaseVIP(log, cfgPath, masterDevice, name, mac, ""); err != nil {
		log.WithFields(logrus.Fields{
			"masterDevice": masterDevice,
			"name":         name,
			"mac":          mac,
			"ip":           "",
		}).WithError(err).Error("Failed to lease a vip")

		return "", err
	}

	ifaceName, ip, err := monitor.GetLastLeaseFromFile(log, monitor.GetLeaseFile(cfgPath, name))

	if err != nil {
		log.WithFields(logrus.Fields{
			"fileName": monitor.GetLeaseFile(cfgPath, name),
		}).WithError(err).Error("Failed to get last lease from file")
		return "", err
	}

	if ifaceName != name {
		log.WithFields(logrus.Fields{
			"expectedInterface": name,
			"actualInterface":   ifaceName,
		}).WithError(err).Error("Interface name is different from expceted")
		return "", err
	}

	return strfmt.IPv4(ip), nil
}

func LeaseAllocate(leaseAllocateRequestStr string, e Executer, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var dhcpAllocationRequest models.DhcpAllocationRequest

	err := json.Unmarshal([]byte(leaseAllocateRequestStr), &dhcpAllocationRequest)
	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocate: json.Unmarshal")
		return "", err.Error(), -1
	}

	apiVip, err := LeaseByMac(log, configPath, *dhcpAllocationRequest.Interface, "api", dhcpAllocationRequest.APIVipMac.String())

	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocate: LeaseByMac api")
		return "", err.Error(), -1
	}

	ingressVip, err := LeaseByMac(log, configPath, *dhcpAllocationRequest.Interface, "ingress", dhcpAllocationRequest.IngressVipMac.String())

	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocate: LeaseByMac ingress")
		return "", err.Error(), -1
	}

	var dhcpAllocationResponse models.DhcpAllocationResponse = models.DhcpAllocationResponse{
		APIVipAddress:     &apiVip,
		IngressVipAddress: &ingressVip,
	}

	b, err := json.Marshal(&dhcpAllocationResponse)
	if err != nil {
		log.WithError(err).Error("DhcpLeaseAllocate: json.Marshal")
		return "", err.Error(), -1
	}
	return string(b), "", 0
}
