package disk_speed_check

import (
	"encoding/json"
	"time"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	dryModeSyncDurationInNS = 1_000_000
)

type DiskSpeedCheck struct {
	dependecies      IDependencies
	subprocessConfig *config.SubprocessConfig
}

func NewDiskSpeedCheck(subprocessConfig *config.SubprocessConfig, dependencies IDependencies) *DiskSpeedCheck {
	return &DiskSpeedCheck{dependecies: dependencies, subprocessConfig: subprocessConfig}
}

func (p *DiskSpeedCheck) FioPerfCheck(diskSpeedCheckRequestStr string, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var diskPerfCheckRequest models.DiskSpeedCheckRequest

	if err := json.Unmarshal([]byte(diskSpeedCheckRequestStr), &diskPerfCheckRequest); err != nil {
		wrapped := errors.Wrap(err, "Error unmarshaling DiskSpeedCheckRequest")
		log.WithError(err).Error(wrapped.Error())
		return "", wrapped.Error(), -1
	}

	if diskPerfCheckRequest.Path == nil {
		err := errors.New("Missing Filename in DiskSpeedCheckRequest")
		log.WithError(err).Error(err.Error())
		return "", err.Error(), -1
	}

	diskPerf, err := p.getDiskPerf(*diskPerfCheckRequest.Path)
	if err != nil {
		log.WithError(err).Warnf("Failed to get disk's I/O performance: %s", *diskPerfCheckRequest.Path)
		return createResponse(0, *diskPerfCheckRequest.Path), err.Error(), -1
	}

	log.Infof("FIO result on disk %s :fdatasync duration %d ms", *diskPerfCheckRequest.Path, diskPerf)

	response := createResponse(diskPerf, *diskPerfCheckRequest.Path)

	return response, "", 0
}

// Returns the 99th percentile of fdatasync durations in milliseconds
func (p *DiskSpeedCheck) getDiskPerf(path string) (int64, error) {
	if path == "" {
		return -1, errors.New("Missing disk path")
	}

	if p.subprocessConfig.DryRunEnabled {
		// Don't want to cause the disk any harm in dry mode, so just pretend it's fast
		return time.Duration(dryModeSyncDurationInNS).Milliseconds(), nil
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

func createResponse(ioSyncDuration int64, path string) string {
	diskSpeedCheckResponse := models.DiskSpeedCheckResponse{
		IoSyncDuration: ioSyncDuration,
		Path:           path,
	}
	bytes, err := json.Marshal(diskSpeedCheckResponse)
	if err != nil {
		return ""
	}
	return string(bytes)
}
