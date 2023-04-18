package ntp_synchronizer

import (
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"

	"github.com/sirupsen/logrus"
)

const ChronyTimeoutSeconds = 30

//go:generate mockery --name NtpSynchronizerDependencies --inpackage
type NtpSynchronizerDependencies interface {
	ExecutePrivileged(command string, args ...string) (stdout string, stderr string, exitCode int)
	LookupHost(host string) (addrs []string, err error)
	LookupAddr(addr string) (names []string, err error)
}

type ProcessExecuter struct{}

func (e *ProcessExecuter) ExecutePrivileged(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.ExecutePrivileged(command, args...)
}

func (e *ProcessExecuter) LookupHost(host string) (addrs []string, err error) {
	return net.LookupHost(host)
}

func (e *ProcessExecuter) LookupAddr(addr string) (names []string, err error) {
	return net.LookupAddr(addr)
}

func convertSourceState(val string) models.SourceState {
	switch val {
	case "*":
		return models.SourceStateSynced
	case "+":
		return models.SourceStateCombined
	case "-":
		return models.SourceStateNotCombined
	case "?":
		return models.SourceStateUnreachable
	case "x":
		return models.SourceStateError
	case "~":
		return models.SourceStateVariable
	default:
		return models.SourceStateError
	}
}

func addServer(e NtpSynchronizerDependencies, ntpSource string) error {
	stdout, stderr, exitCode := e.ExecutePrivileged("chronyc", "add", "server", ntpSource, "iburst")

	if exitCode == 0 {
		return nil
	} else {
		return errors.Errorf("chronyc exited with non-zero exit code %d: %s\n%s", exitCode, stdout, stderr)
	}
}

func formatChronySourcesOutput(output string) []*models.NtpSource {
	sources := make([]*models.NtpSource, 0)

	for _, line := range strings.Split(output, "\n") {
		// Skip empty lines
		if line == "" {
			continue
		}

		fields := strings.Fields(line)

		// Skip whatever is not a server
		if len(fields) == 0 || fields[0][0] != '^' {
			continue
		}

		sources = append(sources, &models.NtpSource{SourceName: fields[1], SourceState: convertSourceState(string(fields[0][1]))})
	}

	return sources
}

func getNTPSources(e NtpSynchronizerDependencies) ([]*models.NtpSource, error) {
	/* If available we would like to resolve ntp sources hostnames */
	stdout, stderr, exitCode := e.ExecutePrivileged("timeout", strconv.Itoa(ChronyTimeoutSeconds), "chronyc", "-n", "sources")

	switch exitCode {
	case 0:
		return formatChronySourcesOutput(stdout), nil
	case util.TimeoutExitCode:
		return nil, errors.Errorf("chronyc was timed out after %d seconds", ChronyTimeoutSeconds)
	default:
		return nil, errors.Errorf("chronyc exited with non-zero exit code %d: %s\n%s", exitCode, stdout, stderr)
	}
}

func isServerConfigured(executer NtpSynchronizerDependencies, server string) (bool, error) {
	// Check if the server is one of the configured sources
	sources, err := getNTPSources(executer)

	if err != nil {
		return false, errors.Wrapf(err, "Failed to get NTP sources")
	}

	for _, source := range sources {
		if server == source.SourceName {
			return true, nil
		}
	}

	// Check if one of the server CNames is one of the configured sources
	names, err := executer.LookupHost(server)

	if err != nil {
		return false, errors.Wrapf(err, "Failed to lookup server %s", server)
	}

	for _, source := range sources {
		if funk.Contains(names, source.SourceName) {
			return true, nil
		}
	}

	return false, nil
}

func handleNewNtpSources(executer NtpSynchronizerDependencies, log logrus.FieldLogger, commaSeparatedNTPSources string) {
	for _, ntpSource := range strings.Split(commaSeparatedNTPSources, ",") {
		configured, err := isServerConfigured(executer, ntpSource)

		if err != nil {
			/* In case of a failure, just log. */
			log.WithError(err).Warnf("Failed to check if NTP source %s is configured", ntpSource)
		}

		if !configured {
			if err = addServer(executer, ntpSource); err != nil {
				/* In case of a failure, just log. We always want to receive the current sources from the agent */
				log.WithError(err).Errorf("Failed to add NTP server %s", ntpSource)
			}
		}
	}
}

func Run(requestStr string, executer NtpSynchronizerDependencies, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var request models.NtpSynchronizationRequest

	err := json.Unmarshal([]byte(requestStr), &request)
	if err != nil {
		log.WithError(err).Errorf("Failed to unmarshal ntp request string %s", requestStr)
		return "", err.Error(), -1
	}

	if request.NtpSource != nil && swag.StringValue(request.NtpSource) != "" {
		handleNewNtpSources(executer, log, swag.StringValue(request.NtpSource))
	}

	sources, err := getNTPSources(executer)

	if err != nil {
		log.WithError(err).Errorf("Failed to get NTP sources")
		return "", err.Error(), -1
	}

	for index, source := range sources {
		// performs a reverse lookup for the given address
		var names []string
		names, err = executer.LookupAddr(source.SourceName)

		if err != nil {
			log.WithError(err).Debugf("Failed to reverse lookup server %s", source.SourceName)
			continue
		}

		// Sanity check
		if len(names) == 0 {
			log.WithError(err).Debugf("No returned hostnames nor an error was returned on reverse lookup for server %s", source.SourceName)
			continue
		}

		sources[index].SourceName = strings.Trim(names[0], ".")
	}

	var response models.NtpSynchronizationResponse = models.NtpSynchronizationResponse{
		NtpSources: sources,
	}

	b, err := json.Marshal(&response)
	if err != nil {
		log.WithError(err).Errorf("Failed to marshal %v", response)
		return "", err.Error(), -1
	}
	return string(b), "", 0
}
