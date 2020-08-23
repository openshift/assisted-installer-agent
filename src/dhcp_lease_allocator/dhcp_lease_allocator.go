package dhcp_lease_allocator

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

func Lease(log logrus.FieldLogger, cfgPath string, dhcpAllocationRequest models.DhcpAllocationRequest) (strfmt.IPv4, strfmt.IPv4, error) {
	vips := []VIP{
		{Name: "api", MacAddress: dhcpAllocationRequest.APIVipMac.String(), IpAddress: ""},
		{Name: "ingress", MacAddress: dhcpAllocationRequest.IngressVipMac.String(), IpAddress: ""},
	}

	for _, vip := range vips {
		mac, err := net.ParseMAC(vip.MacAddress)

		if err != nil {
			log.WithFields(logrus.Fields{
				"vip": vip,
			}).WithError(err).Error("Failed to parse mac")
			return "", "", err
		}

		if err := LeaseVIP(log, cfgPath, *dhcpAllocationRequest.Interface, vip.Name, mac, vip.IpAddress); err != nil {
			log.WithFields(logrus.Fields{
				"masterDevice": *dhcpAllocationRequest.Interface,
				"name":         vip.Name,
				"mac":          mac,
				"ip":           vip.IpAddress,
			}).WithError(err).Error("Failed to lease a vip")
			return "", "", err
		}

		_, vip.IpAddress, err = monitor.GetLastLeaseFromFile(log, monitor.GetLeaseFile(cfgPath, vip.Name))

		if err != nil {
			return "", "", err
		}
	}

	return strfmt.IPv4(vips[0].IpAddress), strfmt.IPv4(vips[1].IpAddress), nil
}

func LeaseAllocator(leaseAllocatorRequestStr string, e Executer, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var dhcpAllocationRequest models.DhcpAllocationRequest

	err := json.Unmarshal([]byte(leaseAllocatorRequestStr), &dhcpAllocationRequest)
	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocator: json.Unmarshal")
		return "", err.Error(), -1
	}

	apiVip, ingressVip, err := Lease(log, configPath, dhcpAllocationRequest)

	if err != nil {
		log.WithError(err).Errorf("DhcpLeaseAllocator: Lease")
		return "", err.Error(), -1
	}

	var dhcpAllocationResponse models.DhcpAllocationResponse = models.DhcpAllocationResponse{
		APIVipAddress:     &apiVip,
		IngressVipAddress: &ingressVip,
	}

	b, err := json.Marshal(&dhcpAllocationResponse)
	if err != nil {
		log.WithError(err).Error("DhcpLeaseAllocator: json.Marshal")
		return "", err.Error(), -1
	}
	return string(b), "", 0
}
