package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/src/util"

	"github.com/openshift/assisted-installer-agent/src/commands/actions"
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
	serviceAPI serviceAPI
}

func newSession() *stepSession {
	invSession, err := session.New(config.GlobalAgentConfig.TargetURL, config.GlobalAgentConfig.PullSecretToken)
	if err != nil {
		log.Fatalf("Failed to initialize connection: %e", err)
	}
	ret := stepSession{
		InventorySession: *invSession,
		serviceAPI:       newServiceAPI(),
	}
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

	err := s.serviceAPI.PostStepReply(&s.InventorySession, &reply)
	if err != nil {
		switch err.(type) {
		case *installer.V2PostStepReplyUnauthorized:
			s.Logger().Warn("User is not authenticated to perform the operation")
		case *installer.V2PostStepReplyForbidden:
			s.Logger().Warn("User is forbidden to perform the operation")
		default:
			s.Logger().Warnf("Error posting step reply: %s", getErrorMessage(err))
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
	if command == "" {
		errStr := "Missing command"
		s.Logger().Warn(errStr)
		return s.createStepReply(stepType, stepID, "", errStr, -1)
	}

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

func (s *stepSession) handleSingleStepV2(stepType models.StepType, stepID string, command string, args []string, handler HandlerType) models.StepReply {
	s.Logger().Infof("Creating execution step for %s %s", stepType, stepID)

	// TODO: MGMT-9451 remove command == "" after agent changes for new protocol will be pushed, added to allow pushing agent before service
	if command == "" {
		actionToRun, err := actions.New(stepType, args)
		if err != nil {
			return models.StepReply{
				StepType: stepType,
				StepID:   stepID,
				ExitCode: int64(-1),
				Output:   "",
				Error:    err.Error(),
			}
		}
		command, args = actionToRun.CreateCmd()
	}

	return s.handleSingleStep(stepType, stepID, command, args, handler)
}

func (s *stepSession) handleSteps(steps *models.Steps) {
	for _, step := range steps.Instructions {

		go func(step *models.Step) {
			if code, err := s.diagnoseSystem(); code != Undetected {
				log.Errorf("System issue detected before running step: <%s>, command: <%s>, args: <%v>: %s - stopping the execution", step.StepID, step.Command, step.Args, err.Error())
				s.sendStepReply(s.createStepReply(step.StepType, step.StepID, "", err.Error(), int(code)))
				return
			}

			reply := s.handleSingleStepV2(step.StepType, step.StepID, step.Command, step.Args, util.ExecutePrivileged)

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
	if config.GlobalDryRunConfig.DryRunEnabled {
		// diagnoseSystem is not necessary in dry mode
		return Undetected, nil
	}

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

	_, err = io.ReadFull(file, make([]byte, 2))
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
	s.Logger().Info("Query for next steps")
	result, err := s.serviceAPI.GetNextSteps(&s.InventorySession)
	if err != nil {
		invalidateCache()
		switch err.(type) {
		case *installer.V2GetNextStepsNotFound:
			s.Logger().WithError(err).Errorf("Infra-env %s was not found in inventory or user is not authorized, going to sleep forever", config.GlobalAgentConfig.InfraEnvID)
			return -1, ""
		case *installer.V2GetNextStepsUnauthorized:
			s.Logger().WithError(err).Errorf("User is not authenticated to perform the operation, going to sleep forever")
			return -1, ""
		case *installer.V2GetNextStepsForbidden:
			s.Logger().WithError(err).Errorf("User is forbidden to perform the operation, going to sleep forever")
			return -1, ""
		default:
			s.Logger().Warnf("Could not query next steps: %s", getErrorMessage(err))
		}
		return int64(config.GlobalAgentConfig.IntervalSecs), ""
	}
	s.handleSteps(result)
	return result.NextInstructionSeconds, *result.PostStepAction
}

func ProcessSteps(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	var nextRunIn int64
	for afterStep := ""; afterStep != models.StepsPostStepActionExit; {
		select {
		case <-ctx.Done():
			return
		default:
			s := newSession()
			nextRunIn, afterStep = s.processSingleSession()
			if nextRunIn == -1 {
				// sleep forever
				select {}
			}
			time.Sleep(time.Duration(nextRunIn) * time.Second)
		}
	}
}
