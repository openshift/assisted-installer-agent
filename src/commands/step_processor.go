package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-openapi/swag"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-installer-agent/src/util"
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
	cancel            context.CancelFunc
	serviceAPI        serviceAPI
	toolRunnerFactory ToolRunnerFactory
	agentConfig       *config.AgentConfig
	stepCache         *cache.Cache
}

func newSession(cancel context.CancelFunc, agentConfig *config.AgentConfig, toolRunnerFactory ToolRunnerFactory, c *cache.Cache, log log.FieldLogger) *stepSession {
	invSession, err := session.New(agentConfig, agentConfig.TargetURL, agentConfig.PullSecretToken, log)
	if err != nil {
		log.Fatalf("Failed to initialize connection: %e", err)
	}
	ret := stepSession{
		InventorySession:  *invSession,
		cancel:            cancel,
		serviceAPI:        newServiceAPI(agentConfig),
		toolRunnerFactory: toolRunnerFactory,
		agentConfig:       agentConfig,
		stepCache:         c,
	}
	return &ret
}

func (s *stepSession) sendStepReply(reply models.StepReply) {
	if reply.ExitCode == 0 && alreadyExistsInService(s.stepCache, reply.StepType, reply.Output) {
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
		storeInCache(s.stepCache, reply.StepType, reply.Output)
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

func (s *stepSession) handleSingleStep(stepType models.StepType, stepID string, runner Runner) models.StepReply {
	stdout, stderr, exitCode := runner.Run()
	if exitCode != 0 {
		// In case the format of the message below changes, please modify the triage pattern of
		// MSG_PATTERN in repo assisted-installer-deployment in file tools/add_triage_signature.py
		// current link https://github.com/openshift-assisted/assisted-installer-deployment/blob/3f97206dda756dd07886ac038b4cfad32dcc5ee1/tools/add_triage_signature.py#L960-L964
		s.Logger().Errorf(`Step execution failed (exit code %v): <%s>, command: <%s>, args: <%v>. Output:
stdout:
%v

stderr:
%v
`, exitCode, stepID, runner.Command(), runner.Args(), stdout, stderr)
	}

	return s.createStepReply(stepType, stepID, stdout, stderr, exitCode)
}

func (s *stepSession) handleSingleStepV2(stepType models.StepType, stepID string, args []string) models.StepReply {
	s.Logger().Infof("Creating execution step for %s %s args <%v>", stepType, stepID, args)
	nextStepRunner, err := s.toolRunnerFactory.Create(s.agentConfig, stepType, args)
	if err != nil {
		s.Logger().WithError(err).Errorf("Unable to create runner for step <%s>, args <%v>", stepID, args)
		return s.createStepReply(stepType, stepID, "", err.Error(), int(-1))
	}
	return s.handleSingleStep(stepType, stepID, nextStepRunner)
}

func (s *stepSession) handleSteps(steps *models.Steps) {
	for _, step := range steps.Instructions {

		go func(step *models.Step) {
			if code, err := s.diagnoseSystem(); code != Undetected {
				s.Logger().Errorf("System issue detected before running step: <%s>, args: <%v>: %s - stopping the execution", step.StepID, step.Args, err.Error())
				s.sendStepReply(s.createStepReply(step.StepType, step.StepID, "", err.Error(), int(code)))
				return
			}

			reply := s.handleSingleStepV2(step.StepType, step.StepID, step.Args)

			if reply.ExitCode != 0 {
				if code, err := s.diagnoseSystem(); code != Undetected {
					s.Logger().Errorf("System issue detected after running step: <%s>, Type: <%s>, args: <%v>: %s", step.StepID, step.StepType, step.Args, err.Error())
					reply.ExitCode = int64(code)
				}
			}

			s.sendStepReply(reply)

			if s.requiresRestart(reply) {
				s.cancel()
			}
		}(step)
	}
}

// diagnoseSystem runs quick validations that need to need to occur before step and after a failure.
// This is in order to detect and report known problems otherwise manifest as confusing error messages or stuck the whole system in the steps themselves.
// One common example of that is virtual media disconnection.
func (s *stepSession) diagnoseSystem() (errorCode, error) {
	if s.agentConfig.DryRunEnabled {
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

func (s *stepSession) processSingleSession() (delay time.Duration, exit bool, err error) {
	s.Logger().Info("Query for next steps")
	result, err := s.serviceAPI.GetNextSteps(&s.InventorySession)
	if err != nil {
		invalidateCache(s.stepCache)
		switch err.(type) {
		case *installer.V2GetNextStepsNotFound:
			s.Logger().WithError(err).Errorf(
				"infra-env %s was not found in inventory, will freeze",
				s.agentConfig.InfraEnvID,
			)
			select {}
		case *installer.V2GetNextStepsUnauthorized:
			err = errors.Wrapf(err, "user is not authenticated to perform the operation")
		case *installer.V2GetNextStepsForbidden:
			err = errors.Wrapf(err, "user is forbidden to perform the operation")
		default:
			err = fmt.Errorf("could not query next steps: %s", getErrorMessage(err))
		}
		return
	}
	s.handleSteps(result)
	delay = time.Duration(result.NextInstructionSeconds * int64(time.Second))
	exit = swag.StringValue(result.PostStepAction) == models.StepsPostStepActionExit
	return
}

// requiresRestart checks if the agent needs to be restarted after completing this step. Currently
// restarting is necessary after successfully completing the 'upgrade_agent' step.
func (s *stepSession) requiresRestart(reply models.StepReply) bool {
	if reply.StepType != models.StepTypeUpgradeAgent {
		return false
	}
	var result models.UpgradeAgentResponse
	err := json.Unmarshal([]byte(reply.Output), &result)
	if err != nil {
		s.Logger().WithError(err).WithFields(
			logrus.Fields{
				"output": reply.Output,
				"code":   reply.ExitCode,
			}).Error(
			"Failed to unmarshal step result, will assume that the agent doesn't " +
				"need to be restarted",
		)
		return false
	}
	return result.Result == models.UpgradeAgentResultSuccess
}

func ProcessSteps(ctx context.Context, cancel context.CancelFunc, agentConfig *config.AgentConfig, toolRunnerFactory ToolRunnerFactory, wg *sync.WaitGroup, log log.FieldLogger) {
	defer wg.Done()

	c := newCache()

	// We send requests to get next steps in a loop, and the server tells us when to exit and
	// how long to wait before the next iteration of the loop. We also want to retry each
	// iteration with an exponential back-off. But the contract of back-off library that we use
	// is an operation function that returns only an error. We need these `delay` and `exit`
	// variables to get those extra results from the operation function.
	var exit bool
	var delay time.Duration
	operation := func() error {
		s := newSession(cancel, agentConfig, toolRunnerFactory, c, log)
		var err error
		delay, exit, err = s.processSingleSession()
		return err
	}
	notify := func(err error, delay time.Duration) {
		log.Errorf("Step processing failed, will try again in %s: %v", delay, err)
	}
	for !exit {
		backOff := backoff.NewExponentialBackOff()
		err := backoff.RetryNotify(operation, backOff, notify)
		if err != nil {
			log.Errorf("Step processing failed, will exit: %v", err)
			return
		}
		select {
		case <-ctx.Done():
			log.Infof("Step processing has been cancelled")
			return
		case <-time.After(delay):
		}
	}
}
