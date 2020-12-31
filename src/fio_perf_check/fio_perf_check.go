package fio_perf_check

import (
	"encoding/json"
	"fmt"
	"time"

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

	log.Infof("FIO result on disk %s :fdatasync duration %d ms , threshold: %d ms", *fioPerfCheckRequest.Path, diskPerf, *fioPerfCheckRequest.DurationThreshold)

	if diskPerf > *fioPerfCheckRequest.DurationThreshold {
		// If the 99th percentile of fdatasync durations is more than 10ms, it's not fast enough for etcd.
		// See: https://www.ibm.com/cloud/blog/using-fio-to-tell-whether-your-storage-is-fast-enough-for-etcd
		log.WithError(err).Errorf("Disk %s is not fast enough for installation (fdatasync duration: %d)", 
			*fioPerfCheckRequest.Path, diskPerf)
		return createResponse(diskPerf), fmt.Sprintf("Disk %s is not fast enough for installation",
			*fioPerfCheckRequest.Path), int(*fioPerfCheckRequest.ExitCode)
	}

	return createResponse(diskPerf), "", 0
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
						Nine9_000000  int64 `json:"99.000000"`
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
