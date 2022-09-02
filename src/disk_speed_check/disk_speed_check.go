package disk_speed_check

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	dryModeSyncDurationInNS = 1_000_000
	numOfFioJobs            = 2
)

type DiskSpeedCheck struct {
	dependecies      IDependencies
	subprocessConfig *config.SubprocessConfig
}

type fioCheckResponse struct {
	latency int64
	err     error
}

func NewDiskSpeedCheck(subprocessConfig *config.SubprocessConfig, dependencies IDependencies) *DiskSpeedCheck {
	return &DiskSpeedCheck{dependecies: dependencies, subprocessConfig: subprocessConfig}
}

func (p *DiskSpeedCheck) FioPerfCheck(diskSpeedCheckRequestStr string, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var diskPerfCheckRequest models.DiskSpeedCheckRequest

	if err := json.Unmarshal([]byte(diskSpeedCheckRequestStr), &diskPerfCheckRequest); err != nil {
		wrapped := errors.Wrap(err, "Failed to unmarshal DiskSpeedCheckRequest")
		log.WithError(err).Error(wrapped.Error())
		return "", wrapped.Error(), -1
	}

	if diskPerfCheckRequest.Path == nil {
		err := errors.New("Missing Filename in DiskSpeedCheckRequest")
		log.WithError(err).Error(err.Error())
		return "", err.Error(), -1
	}

	// Invoke multiple FIO checks concurrently to simulate overload on disk
	responseCh := make(chan fioCheckResponse, numOfFioJobs)
	p.executeMultipleDiskPerf(*diskPerfCheckRequest.Path, responseCh, numOfFioJobs)

	var maxLatency int64 = -1
	var errMsg string
	for res := range responseCh {
		if res.err != nil {
			errMsg = res.err.Error()
			log.Warnf("Failed to get disk's I/O performance: %s", errMsg)
			// Ignoring the error (as it might be temporary)
			continue
		}

		// Store the worst latency
		if res.latency > maxLatency {
			maxLatency = res.latency
		}
	}

	if maxLatency == -1 {
		// Return an error if all requests failed
		return createResponse(0, *diskPerfCheckRequest.Path), errMsg, -1
	}

	log.Infof("FIO result on disk %s :fdatasync duration %d ms", *diskPerfCheckRequest.Path, maxLatency)
	response := createResponse(maxLatency, *diskPerfCheckRequest.Path)
	return response, "", 0
}

func (p *DiskSpeedCheck) executeMultipleDiskPerf(path string, responseCh chan fioCheckResponse, numOfJobs int) {
	defer close(responseCh)
	wg := sync.WaitGroup{}
	for i := 1; i <= numOfJobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Execute FIO on specified path and send response to channel
			responseCh <- p.getDiskPerf(path)
		}()
	}
	wg.Wait()
}

// Returns the 99th percentile of fdatasync durations in milliseconds
func (p *DiskSpeedCheck) getDiskPerf(path string) fioCheckResponse {
	if path == "" {
		return fioCheckResponse{latency: -1, err: errors.New("Missing disk path")}
	}

	if p.subprocessConfig.DryRunEnabled {
		// Don't want to cause the disk any harm in dry mode, so just pretend it's fast
		return fioCheckResponse{latency: time.Duration(dryModeSyncDurationInNS).Milliseconds(), err: nil}
	}

	// FIO treats colons as multiple device separator, which breaks paths like /dev/disk/by-path/pci-0000:06:0000.0
	escaped_path := strings.ReplaceAll(path, ":", "\\:")
	args := []string{"--filename", escaped_path, "--name=test", "--rw=write", "--ioengine=sync",
		"--size=22m", "-bs=2300", "--fdatasync=1", "--output-format=json"}
	stdout, stderr, exitCode := p.dependecies.Execute("fio", args...)
	if exitCode != 0 {
		err := errors.Errorf("Could not get I/O performance for path %s (fio exit code: %d, stderr: %s)",
			path, exitCode, stderr)
		return fioCheckResponse{latency: -1, err: err}
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
		err1 := errors.Errorf("Failed to get sync duration from I/O info for path %s: %v", path, err)
		return fioCheckResponse{latency: -1, err: err1}
	}
	syncDurationInNS := fio.Jobs[0].Sync.LatNs.Percentile.Nine9_000000
	return fioCheckResponse{latency: time.Duration(syncDurationInNS).Milliseconds(), err: nil}
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
