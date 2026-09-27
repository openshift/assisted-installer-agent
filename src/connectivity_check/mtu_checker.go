package connectivity_check

import (
	"net"
	"sort"
	"strconv"

	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

const defaultMTUThreshold = 1500
const ipv4Header = 28
const ipv6Header = 48

type mtuChecker struct {
	executer Executer
}

func (m *mtuChecker) Features() Features {
	return OutgoingNicFeature
}

func (m *mtuChecker) Check(attributes Attributes) ResultReporter {
	ret := models.MtuReport{
		RemoteIPAddress: attributes.RemoteIPAddress,
		OutgoingNic:     attributes.OutgoingNIC.Name,
	}

	// Check if the remote IP address is IPv4 or IPv6
	remoteIP := net.ParseIP(attributes.RemoteIPAddress)
	if remoteIP == nil {
		log.Errorf("MTU checker: Invalid remote IP address %s", attributes.RemoteIPAddress)
		return newMtuResultReporter(&ret)
	}
	isRemoteIPv6 := remoteIP.To4() == nil // If To4() is nil, it's IPv6

	var localIP string

	mtu := attributes.OutgoingNIC.MTU

	// Only consider hosts with MTU != 1500
	if mtu == defaultMTUThreshold {
		return nil
	}

	for _, addr := range attributes.OutgoingNIC.Addresses {
		ipN, ok := addr.(*net.IPNet)
		if !ok {
			log.Errorf("MTU checker: failed convert address %v", addr)
			return newMtuResultReporter(&ret)
		}

		localIP = ipN.IP.String()

		// Check if the local IP address is IPv4 or IPv6
		isLocalIPv6 := ipN.IP.To4() == nil // If To4() is nil, it's IPv6
		if isLocalIPv6 != isRemoteIPv6 {
			continue
		}

		// Perform an initial ping without specifying the MTU to rule out the possibility of failure due to issues unrelated to MTU.
		_, err := m.executer.Execute("ping", attributes.RemoteIPAddress, "-c", "3", "-I", attributes.OutgoingNIC.Name)
		if err != nil {
			log.WithError(err).Errorf("MTU checker: failed first ping. Remote address: %s, nic: %s, local address: %s", attributes.RemoteIPAddress, attributes.OutgoingNIC.Name, localIP)
			return nil
		}

		var sizeWithoutIPHeader int
		if isLocalIPv6 {
			sizeWithoutIPHeader = mtu - ipv6Header
		} else {
			sizeWithoutIPHeader = mtu - ipv4Header
		}

		// Second ping with MTU
		_, err = m.executer.Execute("ping", attributes.RemoteIPAddress, "-c", "3", "-M", "do", "-s", strconv.Itoa(sizeWithoutIPHeader), "-I", attributes.OutgoingNIC.Name)
		if err != nil {
			log.WithError(err).Errorf("MTU checker: failed to ping address %s nic %s mtu %d", attributes.RemoteIPAddress, attributes.OutgoingNIC.Name, mtu)
			return newMtuResultReporter(&ret)
		}
	}
	ret.MtuSuccessful = true
	return newMtuResultReporter(&ret)
}

func (m *mtuChecker) Finalize(resultingHost *models.ConnectivityRemoteHost) {
	sort.SliceStable(resultingHost.MtuReport,
		func(i, j int) bool {
			return resultingHost.MtuReport[i].RemoteIPAddress < resultingHost.MtuReport[j].RemoteIPAddress
		})
}

type mtuResult struct {
	mtuReport *models.MtuReport
}

func (m *mtuResult) Report(resultingHost *models.ConnectivityRemoteHost) error {
	resultingHost.MtuReport = append(resultingHost.MtuReport, m.mtuReport)
	return nil
}

func newMtuResultReporter(mtuReport *models.MtuReport) ResultReporter {
	return &mtuResult{mtuReport: mtuReport}
}
