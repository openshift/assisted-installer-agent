package commands

import (
	"net/http"
	"time"

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

func (s *stepSession) sendStepReply(stepType models.StepType, stepID, output, errStr string, exitCode int) {
	logFunc := s.Logger().Infof
	if exitCode != 0 {
		logFunc = s.Logger().Warnf
	}
	logFunc("Sending step <%s> reply output <%s> error <%s> exit-code <%d>", stepID, output, errStr, exitCode)
	params := installer.PostStepReplyParams{
		HostID:                *CurrentHost.ID,
		ClusterID:             strfmt.UUID(config.GlobalAgentConfig.ClusterID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.AgentVersion,
	}
	reply := models.StepReply{
		StepType: stepType,
		StepID:   stepID,
		ExitCode: int64(exitCode),
		Output:   output,
		Error:    errStr,
	}
	params.Reply = &reply
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
	}
}

func (s *stepSession) handleSingleStep(stepType models.StepType, stepID string, command string, args []string, handler HandlerType) {
	s.Logger().Infof("Executing step: <%s>, command: <%s>, args: <%v>", stepID, command, args)
	stdout, stderr, exitCode := handler(command, args...)
	s.sendStepReply(stepType, stepID, stdout, stderr, exitCode)
}

func (s *stepSession) handleSteps(steps *models.Steps) {
	for _, step := range steps.Instructions {
		if step.Command == "" {
			errStr := "Missing command"
			s.Logger().Warn(errStr)
			s.sendStepReply(step.StepType, step.StepID, "", errStr, -1)
			continue
		}
		go s.handleSingleStep(step.StepType, step.StepID, step.Command, step.Args, util.Execute)
	}
}

func (s *stepSession) processSingleSession() int64 {
	params := installer.GetNextStepsParams{
		HostID:                *CurrentHost.ID,
		ClusterID:             strfmt.UUID(config.GlobalAgentConfig.ClusterID),
		DiscoveryAgentVersion: &config.GlobalAgentConfig.AgentVersion,
	}
	s.Logger().Info("Query for next steps")
	result, err := s.Client().Installer.GetNextSteps(s.Context(), &params)
	if err != nil {
		switch errValue := err.(type) {
		case *installer.GetNextStepsNotFound:
			s.Logger().WithError(err).Errorf("Cluster %s was not found in inventory or user is not authorized, going to sleep forever", params.ClusterID)
			return -1
		case *installer.GetNextStepsUnauthorized:
			s.Logger().WithError(err).Errorf("User is not authenticated to perform the operation, going to sleep forever")
			return -1
		case *installer.GetNextStepsInternalServerError:
			s.Logger().Warnf("Error getting get next steps: %s, %s", http.StatusText(http.StatusInternalServerError), swag.StringValue(errValue.Payload.Reason))
		default:
			s.Logger().WithError(err).Warn("Could not query next steps")
		}
		return int64(config.GlobalAgentConfig.IntervalSecs)
	} else {
		s.handleSteps(result.Payload)
	}
	return result.Payload.NextInstructionSeconds
}

func ProcessSteps() {
	var nextRunIn int64
	for {
		s := newSession()
		nextRunIn = s.processSingleSession()
		if nextRunIn == -1 {
			// sleep forever
			select {}
		}
		time.Sleep(time.Duration(nextRunIn) * time.Second)
	}
}
