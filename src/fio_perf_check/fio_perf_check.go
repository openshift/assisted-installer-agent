package fio_perf_check

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/session"
	"github.com/openshift/assisted-service/client/installer"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type PerfCheck struct {
	dependecies IDependencies
}

func NewPerfCheck(dependencies IDependencies) *PerfCheck {
	return &PerfCheck{dependecies: dependencies}
}

func (p *PerfCheck) FioPerfCheck(fioPerfCheckRequestStr string, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var fioPerfCheckRequest models.FioPerfCheckRequest

	if err := json.Unmarshal([]byte(fioPerfCheckRequestStr), &fioPerfCheckRequest); err != nil {
		wrapped := errors.Wrap(err, "Error unmarshaling FioPerfCheckRequest")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(-1), wrapped.Error(), -1
	}

	if fioPerfCheckRequest.Path == nil {
		err := errors.New("Missing Filename in FioPerfCheckRequest")
		log.WithError(err).Error(err.Error())
		return createResponse(-1), err.Error(), -1
	}

	diskPerf, err := p.getDiskPerf(*fioPerfCheckRequest.Path)
	if err != nil {
		log.WithError(err).Warnf("Failed to get disk's I/O performance: %s", *fioPerfCheckRequest.Path)
		return createResponse(-1), err.Error(), -1
	}

	log.Infof("FIO result on disk %s :fdatasync duration %d ms , threshold: %d ms", *fioPerfCheckRequest.Path, diskPerf, *fioPerfCheckRequest.DurationThresholdMs)

	response := createResponse(diskPerf)

	invSession, err := session.New(config.GlobalAgentConfig.TargetURL, config.GlobalAgentConfig.PullSecretToken)
	if err != nil {
		log.Fatalf("Failed to initialize connection: %e", err)
	} else {
		params := installer.PostStepReplyParams{
			HostID:                strfmt.UUID(config.GlobalAgentConfig.HostID),
			ClusterID:             strfmt.UUID(config.GlobalAgentConfig.ClusterID),
			DiscoveryAgentVersion: &config.GlobalAgentConfig.AgentVersion,
		}
		reply := models.StepReply{
			StepType: models.StepTypeFioPerfCheck,
			StepID:   "fio-perf-check",
			ExitCode: 0,
			Output:   response,
			Error:    "",
		}

		params.Reply = &reply
		_, err = invSession.Client().Installer.PostStepReply(context.Background(), &params)
		if err != nil {
			switch errValue := err.(type) {
			case *installer.PostStepReplyInternalServerError:
				log.Warnf("Assisted service returned status code %s after processing step reply. Reason: %s", http.StatusText(http.StatusInternalServerError), swag.StringValue(errValue.Payload.Reason))
			case *installer.PostStepReplyUnauthorized:
				log.Warn("User is not authenticated to perform the operation")
			case *installer.PostStepReplyBadRequest:
				log.Warnf("Assisted service returned status code %s after processing step reply.  Reason: %s", http.StatusText(http.StatusBadRequest), swag.StringValue(errValue.Payload.Reason))
			default:
				log.WithError(err).Warn("Error posting step reply")
			}
		}
	}

	if diskPerf > *fioPerfCheckRequest.DurationThresholdMs {
		// If the 99th percentile of fdatasync durations is more than 10ms, it's not fast enough for etcd.
		// See: https://www.ibm.com/cloud/blog/using-fio-to-tell-whether-your-storage-is-fast-enough-for-etcd
		log.WithError(err).Errorf("Disk %s is not fast enough for installation (fdatasync duration: %d)",
			*fioPerfCheckRequest.Path, diskPerf)
		return response, fmt.Sprintf("Disk %s is not fast enough for installation",
			*fioPerfCheckRequest.Path), int(*fioPerfCheckRequest.ExitCode)
	}

	return response, "", 0
}

// Returns the 99th percentile of fdatasync durations in milliseconds
func (p *PerfCheck) getDiskPerf(path string) (int64, error) {
	if path == "" {
		return -1, errors.New("Missing disk path")
	}

	args := []string{"--filename", path, "--name=test", "--rw=write", "--ioengine=sync",
		"--size=22m", "-bs=2300", "--fdatasync=1", "--output-format=json"}
	stdout, stderr, exitCode := p.dependecies.Execute("fio", args...)
	if exitCode != 0 {
		return -1, errors.Errorf("Could not get I/O performance for path %s: (fio exit code %d) %s",
			path, exitCode, stderr)
	}

	type FIO struct {
		Jobs []struct {
			Sync struct {
				LatNs struct {
					Percentile struct {
						Nine9_000000 int64 `json:"99.000000"`
					} `json:"percentile"`
				} `json:"lat_ns"`
			} `json:"sync"`
		} `json:"jobs"`
	}

	fio := FIO{}
	err := json.Unmarshal([]byte(stdout), &fio)
	if err != nil {
		return -1, errors.Errorf("Failed to get sync duration from I/O info for path %s", path)
	}
	syncDurationInNS := fio.Jobs[0].Sync.LatNs.Percentile.Nine9_000000
	return time.Duration(syncDurationInNS).Milliseconds(), nil
}

func createResponse(ioSyncDuration int64) string {
	fioPerfCheckResponse := models.FioPerfCheckResponse{
		IoSyncDuration: ioSyncDuration,
	}
	bytes, err := json.Marshal(fioPerfCheckResponse)
	if err != nil {
		return ""
	}
	return string(bytes)
}
