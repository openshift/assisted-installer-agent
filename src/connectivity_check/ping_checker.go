package connectivity_check

import (
	"sort"
	"strconv"

	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const pingCount string = "10"

type pingChecker struct {
	executer Executer
}

func (p *pingChecker) Features() Features {
	return RemoteIPFeature
}

func (p *pingChecker) Check(attributes Attributes) ResultReporter {
	ret := models.L3Connectivity{
		RemoteIPAddress: attributes.RemoteIPAddress,
	}

	b, err := p.executer.Execute("ping", "-c", pingCount, "-W", "3", "-q", attributes.RemoteIPAddress)
	if err != nil {
		log.WithError(err).Errorf("failed to ping address %s", attributes.RemoteIPAddress)
		return newL3ResultReporter(&ret)
	}
	if err = parsePingCmd(&ret, b); err != nil {
		log.WithError(err).Errorf("failed to parse ping result to address %s", attributes.RemoteIPAddress)
		return newL3ResultReporter(&ret)
	}
	ret.Successful = true
	return newL3ResultReporter(&ret)
}

func (p *pingChecker) Finalize(resultingHost *models.ConnectivityRemoteHost) {
	sort.SliceStable(resultingHost.L3Connectivity,
		func(i, j int) bool {
			return resultingHost.L3Connectivity[i].RemoteIPAddress < resultingHost.L3Connectivity[j].RemoteIPAddress
		})
}

type pingResult struct {
	l3Connectivity *models.L3Connectivity
}

func (p *pingResult) Report(resultingHost *models.ConnectivityRemoteHost) error {
	resultingHost.L3Connectivity = append(resultingHost.L3Connectivity, p.l3Connectivity)
	return nil
}

func newL3ResultReporter(l3Connectivity *models.L3Connectivity) ResultReporter {
	return &pingResult{l3Connectivity: l3Connectivity}
}

func parsePingCmd(conn *models.L3Connectivity, cmdOutput string) error {
	if len(cmdOutput) == 0 {
		return errors.Errorf("Missing output for ping or invalid output:\n%s", cmdOutput)
	}
	parts, err := regexMatchFor(`[\d]+ packets transmitted, [\d]+ received, (([\d]*[.])?[\d]+)% packet loss, time [\d]+ms`, cmdOutput)
	if err != nil {
		return errors.Errorf("Unable to retrieve packet loss percentage: %s", err)
	}
	conn.PacketLossPercentage, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return errors.Errorf("Error while trying to convert value for packet loss '%s': %s", parts[1], err)
	}
	parts, err = regexMatchFor(`rtt min\/avg\/max\/mdev = .*\/([^\/]+)\/.*\/.* ms`, cmdOutput)
	if err != nil {
		return errors.Errorf("Unable to retrieve the average RTT for ping: %s", err)
	}
	conn.AverageRTTMs, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return errors.Errorf("Error while trying to convert value for packet loss %s: %s", parts[1], err)
	}
	return nil
}
