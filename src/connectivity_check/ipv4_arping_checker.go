package connectivity_check

import (
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type arpingChecker struct {
	executer Executer
}

func (p *arpingChecker) Features() Features {
	return RemoteIPFeature | RemoteMACFeature | OutgoingNicFeature
}

func (p *arpingChecker) Check(attributes Attributes) ResultReporter {
	if !util.IsIPv4Addr(attributes.RemoteIPAddress) || !attributes.OutgoingNIC.HasIpv4Addresses {
		return nil
	}

	result, err := p.executer.Execute("arping", "-c", "10", "-w", "5", "-I", attributes.OutgoingNIC.Name, attributes.RemoteIPAddress)
	if err != nil {
		// Ignore exit code of 1; only 2 or -1 are actual errors
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			log.WithError(err).Error("Error while processing 'arping' command")
			return nil
		}
	}
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		log.Warn("Missing output for arping")
		return nil
	}

	hRegex := regexp.MustCompile("^ARPING ([^ ]+) from ([^ ]+) ([^ ]+)$")
	parts := hRegex.FindStringSubmatch(lines[0])
	if len(parts) != 4 {
		log.Warnf("Wrong format for header line: %s", lines[0])
		return nil
	}

	outgoingIpAddress := parts[2]
	rRegexp := regexp.MustCompile(`^Unicast reply from ([^ ]+) \[([^]]+)\]  [^ ]+$`)

	// We use a map to de-duplicate arping responses from the same mac. They are redundant
	// because the reason we're reporting more than one line of arping in the first place is to allow
	// the service to also detect devices in the network that have IP conflict with cluster hosts
	returnValues := make(map[string]*models.L2Connectivity)
	for _, line := range lines[1:] {
		parts = rRegexp.FindStringSubmatch(line)
		if len(parts) != 3 {
			continue
		}
		remoteMAC := strings.ToLower(parts[2])
		successful := macInDstMacs(remoteMAC, attributes.RemoteMACAddresses)
		if !successful {
			log.Warnf("Unexpected mac address for arping %s on nic %s: %s", attributes.RemoteIPAddress, attributes.OutgoingNIC.Name, remoteMAC)
		}
		if strings.ToLower(attributes.RemoteMACAddress) != remoteMAC {
			log.Infof("Received remote mac %s different then expected mac %s", remoteMAC, attributes.RemoteMACAddress)
		}

		returnValues[remoteMAC] = &models.L2Connectivity{
			OutgoingIPAddress: outgoingIpAddress,
			OutgoingNic:       attributes.OutgoingNIC.Name,
			RemoteIPAddress:   attributes.RemoteIPAddress,
			RemoteMac:         remoteMAC,
			Successful:        successful,
		}
	}
	var l2Connectivity []*models.L2Connectivity
	for _, v := range returnValues {
		l2Connectivity = append(l2Connectivity, v)
	}
	return newArpingResultReporter(l2Connectivity)
}

func (p *arpingChecker) Finalize(resultingHost *models.ConnectivityRemoteHost) {
	l2Key := func(c *models.L2Connectivity) string {
		return c.RemoteIPAddress + c.RemoteMac
	}

	sort.SliceStable(resultingHost.L2Connectivity,
		func(i, j int) bool {
			return l2Key(resultingHost.L2Connectivity[i]) < l2Key(resultingHost.L2Connectivity[j])
		})
}

type arpingResult struct {
	l2Connectivity []*models.L2Connectivity
}

func (p *arpingResult) Report(resultingHost *models.ConnectivityRemoteHost) error {
	resultingHost.L2Connectivity = append(resultingHost.L2Connectivity, p.l2Connectivity...)
	return nil
}

func newArpingResultReporter(l2Connectivity []*models.L2Connectivity) ResultReporter {
	return &arpingResult{
		l2Connectivity: l2Connectivity,
	}
}
