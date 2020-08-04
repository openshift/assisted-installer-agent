package commands

import (
	"reflect"
	"time"

	"github.com/ori-amizur/introspector/src/util"

	"github.com/go-openapi/strfmt"

	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
	"github.com/ori-amizur/introspector/src/config"
	"github.com/ori-amizur/introspector/src/session"
)

type HandlerType func(string, ...string) (stdout string, stderr string, exitCode int)

type stepSession struct {
	session.InventorySession
}

func newSession() *stepSession {
	ret := stepSession{*session.New()}
	return &ret
}

func (s *stepSession) sendStepReply(stepType models.StepType, stepID, output, errStr string, exitCode int) {
	s.Logger().Infof("Sending step <%s> reply output <%s> error <%s> exit-code <%d>", stepID, output, errStr, exitCode)
	params := installer.PostStepReplyParams{
		HostID:    *CurrentHost.ID,
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
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
		s.Logger().Warnf("Error posting step reply: %s", err.Error())
	}
}

func (s *stepSession) handleSingleStep(stepType models.StepType, stepID string, command string, args []string, handler HandlerType) {
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
		HostID:    *CurrentHost.ID,
		ClusterID: strfmt.UUID(config.GlobalAgentConfig.ClusterID),
	}
	s.Logger().Info("Query for next steps")
	result, err := s.Client().Installer.GetNextSteps(s.Context(), &params)
	if err != nil {
		s.Logger().Warnf("Could not query next steps: %s", err.Error())
		if reflect.TypeOf(err) == reflect.TypeOf(installer.NewGetNextStepsNotFound()) {
			s.Logger().WithError(err).Errorf("Cluster %s was not fount in inventory, going to sleep forever", params.ClusterID)
			return -1
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
