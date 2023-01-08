package connectivity_check

import (
	"encoding/xml"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-installer-agent/src/util/nmap"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type nmapChecker struct {
	executer Executer
}

func (p *nmapChecker) Features() Features {
	return RemoteIPFeature | RemoteMACFeature | OutgoingNicFeature
}

func (p *nmapChecker) Check(attributes Attributes) ResultReporter {
	if util.IsIPv4Addr(attributes.RemoteIPAddress) {
		return nil
	}
	result, err := p.executer.Execute("nmap", "-6", "-sn", "-n", "-oX", "-", "-e", attributes.OutgoingNIC, attributes.RemoteIPAddress)
	if err != nil {
		log.WithError(err).Error("Error while processing 'nmap' command")
		return nil
	}
	var nmaprun nmap.Nmaprun
	if err := xml.Unmarshal([]byte(result), &nmaprun); err != nil {
		log.WithError(err).Warn("Failed to un-marshal nmap XML")
		return nil
	}

	ret := models.L2Connectivity{
		OutgoingNic:     attributes.OutgoingNIC,
		RemoteIPAddress: attributes.RemoteIPAddress,
	}

	for _, h := range nmaprun.Hosts {
		if h.Status.State != "up" {
			continue
		}
		for _, a := range h.Addresses {
			if a.AddrType != "mac" {
				continue
			}
			remoteMAC := strings.ToLower(a.Addr)
			ret.RemoteMac = remoteMAC
			ret.Successful = macInDstMacs(remoteMAC, attributes.RemoteMACAddresses)
			if !ret.Successful {
				log.Warnf("Unexpected MAC address for nmap %s on NIC %s: %s", attributes.RemoteIPAddress, attributes.OutgoingNIC, remoteMAC)
			} else if strings.ToLower(attributes.RemoteMACAddress) != remoteMAC {
				log.Infof("Received remote MAC %s different then expected MAC %s", remoteMAC, attributes.RemoteMACAddress)
			}

			return newL2ResultReporter(&ret)
		}
	}
	return nil
}

func (p *nmapChecker) Finalize(*models.ConnectivityRemoteHost) {}

type l2Result struct {
	l2Connectivity *models.L2Connectivity
}

func (p *l2Result) Report(resultingHost *models.ConnectivityRemoteHost) error {
	resultingHost.L2Connectivity = append(resultingHost.L2Connectivity, p.l2Connectivity)
	return nil
}

func newL2ResultReporter(l2Connectivity *models.L2Connectivity) ResultReporter {
	return &l2Result{
		l2Connectivity: l2Connectivity,
	}
}
