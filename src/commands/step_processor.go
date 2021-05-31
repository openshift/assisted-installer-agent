package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-openapi/swag"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/go-openapi/strfmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
)

type HandlerType func(string, ...string) (stdout string, stderr string, exitCode int)

type errorCode int

// TODO - add an ErrorCode Enum to the swagger
const (
	Undetected        errorCode = 999
	MediaDisconnected errorCode = 256
)

type stepSession struct {
	session.InventorySession
}

func newSession() *stepSession {
	invSession, err := session.New(config.GlobalAgentConfig.TargetURL, config.GlobalAgentConfig.PullSecretToken)
	if err != nil {
		log.Fatalf("Failed to initialize connection: %e", err)
	}
	ret := stepSession{*invSession}
	return &ret
}

func (s *stepSession) sendStepReply(reply models.StepReply) {
	if reply.ExitCode == 0 && alreadyExistsInService(reply.StepType, reply.Output) {
		s.Logger().Infof("Result for %s already exists in assisted service", string(reply.StepType))
		return
	}
	logFunc := s.Logger().Infof
	if reply.ExitCode != 0 {
		logFunc = s.Logger().Warnf
	}

	if reply.StepType == models.StepTypeFreeNetworkAddresses {
		// the free-addresses step's output spams the log too much
		logFunc("Sending step <%s> reply error <%s> exit-code <%d>", reply.StepID, reply.Error, reply.ExitCode)
	} else {
		logFunc("Sending step <%s> reply output <%s> error <%s> exit-code <%d>", reply.StepID, reply.Output, reply.Error, reply.ExitCode)
	}

	params := installer.PostStepReplyParams{
		HostID:                strfmt.UUID(config.GlobalAgentConfig.HostID),
		ClusterID:             strfmt.UUID(config.GlobalAgentConfig.ClusterID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.AgentVersion,
		Reply:                 &reply,
	}

	_, err := s.Client().Installer.PostStepReply(s.Context(), &params)
	if err != nil {
		switch errValue := err.(type) {
		case *installer.PostStepReplyInternalServerError:
			s.Logger().Warnf("Assisted service returned status code %s after processing step reply. Reason: %s", http.StatusText(http.StatusInternalServerError), swag.StringValue(errValue.Payload.Reason))
		case *installer.PostStepReplyUnauthorized:
			s.Logger().Warn("User is not authenticated to perform the operation")
		case *installer.PostStepReplyBadRequest:
			s.Logger().Warnf("Assisted service returned status code %s after processing step reply.  Reason: %s", http.StatusText(http.StatusBadRequest), swag.StringValue(errValue.Payload.Reason))
		default:
			s.Logger().WithError(err).Warn("Error posting step reply")
		}
	} else if reply.ExitCode == 0 {
		storeInCache(reply.StepType, reply.Output)
	}
}

func (s *stepSession) createStepReply(stepType models.StepType, stepID string, output string, errStr string, exitCode int) models.StepReply {
	return models.StepReply{
		StepType: stepType,
		StepID:   stepID,
		ExitCode: int64(exitCode),
		Output:   output,
		Error:    errStr,
	}
}

func (s *stepSession) handleSingleStep(stepType models.StepType, stepID string, command string, args []string, handler HandlerType) models.StepReply {
	s.Logger().Infof("Executing step: <%s>, command: <%s>, args: <%v>", stepID, command, args)
	stdout, stderr, exitCode := handler(command, args...)
	if exitCode != 0 {
		s.Logger().Errorf(`Step execution failed (exit code %v): <%s>, command: <%s>, args: <%v>. Output:
stdout:
%v

stderr:
%v
`, exitCode, stepID, command, args, stdout, stderr)
	}

	return s.createStepReply(stepType, stepID, stdout, stderr, exitCode)
}

func (s *stepSession) handleSteps(steps *models.Steps) {
	for _, step := range steps.Instructions {
		if step.Command == "" {
			errStr := "Missing command"
			s.Logger().Warn(errStr)
			s.sendStepReply(s.createStepReply(step.StepType, step.StepID, "", errStr, -1))
			continue
		}

		go func(step *models.Step) {
			if code, err := s.diagnoseSystem(); code != Undetected {
				log.Errorf("System issue detected before running step: <%s>, command: <%s>, args: <%v>: %s - stopping the execution", step.StepID, step.Command, step.Args, err.Error())
				s.sendStepReply(s.createStepReply(step.StepType, step.StepID, "", err.Error(), int(code)))
				return
			}

			reply := s.handleSingleStep(step.StepType, step.StepID, step.Command, step.Args, util.ExecutePrivileged)

			if reply.ExitCode != 0 {
				if code, err := s.diagnoseSystem(); code != Undetected {
					log.Errorf("System issue detected after running step: <%s>, command: <%s>, args: <%v>: %s", step.StepID, step.Command, step.Args, err.Error())
					reply.ExitCode = int64(code)
				}
			}

			s.sendStepReply(reply)
		}(step)
	}
}

// diagnoseSystem runs quick validations that need to need to occur before step and after a failure.
// This is in order to detect and report known problems otherwise manifest as confusing error messages or stuck the whole system in the steps themselves.
// One common example of that is virtual media disconnection.
func (s *stepSession) diagnoseSystem() (errorCode, error) {
	source, err := s.getMountpointSourceDeviceFile()

	if err != nil {
		log.Warn(err)
		return Undetected, nil
	}

	if source == "" {
		return Undetected, nil
	}

	file, err := os.Open(source)
	if err != nil {
		return MediaDisconnected, errors.Wrap(err, "cannot access the media (ISO) - media was likely disconnected")
	}

	defer file.Close()

	_, err = io.ReadFull(file, make([]byte, 2, 2))
	if err != nil {
		return MediaDisconnected, errors.Wrap(err, "cannot read from the media (ISO) - media was likely disconnected")
	}

	return Undetected, nil
}

func (s *stepSession) getMountpointSourceDeviceFile() (string, error) {
	mediaPath := "/run/media/iso"

	// Media disconnection issue occurs only for a full-ISO installation
	// mostly when serving iso via virtual media via sub optimal networking conditions.
	// The minimal-ISO loaded very early and stay in memory. We don't need to read them from the ISO once they're loaded
	// The media path exists only for the full-ISO so we can just eliminate this check.
	if _, err := os.Stat(mediaPath); err != nil {
		return "", nil
	}

	stdout, stderr, exitCode := util.ExecutePrivileged("findmnt", "--raw", "--noheadings", "--output", "SOURCE,TARGET", "--target", mediaPath)

	errorMessage := "failed to validate media disconnection - continuing"

	if exitCode != 0 {
		return "", errors.Errorf("%s: %s", errorMessage, stderr)
	}

	if stdout == "" {
		return "", errors.Errorf("%s: cannot find ISO mountpoint source", errorMessage)
	}

	fields := strings.Fields(stdout)

	if fields[1] != mediaPath {
		return "", fmt.Errorf("%s: media mounted to %s instead of directly to %s", errorMessage, fields[1], mediaPath)
	}

	source := fields[0]
	if source == "" || !strings.HasPrefix(source, "/dev") {
		return "", fmt.Errorf("%s: the mount source isn't a device file %s", errorMessage, source)
	}

	return source, nil
}

func (s *stepSession) processSingleSession() (int64, string) {
	params := installer.GetNextStepsParams{
		HostID:                strfmt.UUID(config.GlobalAgentConfig.HostID),
		ClusterID:             strfmt.UUID(config.GlobalAgentConfig.ClusterID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.AgentVersion,
	}
	s.Logger().Info("Query for next steps")
	result, err := s.Client().Installer.GetNextSteps(s.Context(), &params)
	if err != nil {
		invalidateCache()
		switch errValue := err.(type) {
		case *installer.GetNextStepsNotFound:
			s.Logger().WithError(err).Errorf("Cluster %s was not found in inventory or user is not authorized, going to sleep forever", params.ClusterID)
			return -1, ""
		case *installer.GetNextStepsUnauthorized:
			s.Logger().WithError(err).Errorf("User is not authenticated to perform the operation, going to sleep forever")
			return -1, ""
		case *installer.GetNextStepsInternalServerError:
			s.Logger().Warnf("Error getting get next steps: %s, %s", http.StatusText(http.StatusInternalServerError), swag.StringValue(errValue.Payload.Reason))
		default:
			s.Logger().WithError(err).Warn("Could not query next steps")
		}
		return int64(config.GlobalAgentConfig.IntervalSecs), ""
	}
	s.handleSteps(result.Payload)
	return result.Payload.NextInstructionSeconds, *result.Payload.PostStepAction
}

func ProcessSteps() {
	var nextRunIn int64
	for afterStep := ""; afterStep != models.StepsPostStepActionExit; {
		s := newSession()
		nextRunIn, afterStep = s.processSingleSession()
		if nextRunIn == -1 {
			// sleep forever
			select {}
		}
		time.Sleep(time.Duration(nextRunIn) * time.Second)
	}
}
