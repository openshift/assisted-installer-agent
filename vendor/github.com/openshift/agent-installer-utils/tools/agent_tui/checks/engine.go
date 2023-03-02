package checks

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	CheckTypeReleaseImagePull     = "ReleaseImagePull"
	CheckTypeReleaseImageHostDNS  = "ReleaseImageHostDNS"
	CheckTypeReleaseImageHostPing = "ReleaseImageHostPing"
	CheckTypeReleaseImageHttp     = "ReleaseImageHttp"
)

type Config struct {
	ReleaseImageURL string
	LogPath         string

	ReleaseImageHostname           string
	ReleaseImageSchemeHostnamePort string
}

// ChecksEngine is the model part, and is composed by a number
// of different checks.
// Each Check has a type, frequency and evaluation loop.
// Different checks could have the same type

type CheckResult struct {
	Type    string
	Success bool
	Details string // In case of failure
}

type Check struct {
	Type string
	Freq time.Duration //Note: a ticker could be useful
	Run  func(c chan CheckResult, Freq time.Duration)
}

type Engine struct {
	checks  []*Check
	channel chan CheckResult
	logger  *logrus.Logger
}

type checkFunction func() ([]byte, error)

func (e *Engine) createCheckResult(f checkFunction, checkType string) CheckResult {
	output, err := f()
	result := CheckResult{
		Type:    checkType,
		Success: err == nil,
		Details: string(output),
	}
	if result.Success {
		e.logger.Infof("%s check successful: %s", checkType, result.Details)
	} else {
		e.logger.Warnf("%s check failed with error: %s", checkType, result.Details)
	}
	return result
}

func (e *Engine) newRegistryImagePullCheck(config Config) *Check {
	ctype := CheckTypeReleaseImagePull
	return &Check{
		Type: ctype,
		Freq: 5 * time.Second,
		Run: func(c chan CheckResult, freq time.Duration) {
			for {
				checkFunction := func() ([]byte, error) {
					return exec.Command("podman", "pull", config.ReleaseImageURL).CombinedOutput()
				}
				c <- e.createCheckResult(checkFunction, ctype)
				time.Sleep(freq)
			}
		},
	}
}

func (e *Engine) newReleaseImageHostDNSCheck(config Config) *Check {
	ctype := CheckTypeReleaseImageHostDNS
	return &Check{
		Type: ctype,
		Freq: 5 * time.Second,
		Run: func(c chan CheckResult, freq time.Duration) {
			for {
				checkFunction := func() ([]byte, error) {
					return exec.Command("nslookup", config.ReleaseImageHostname).CombinedOutput()
				}
				c <- e.createCheckResult(checkFunction, ctype)
				time.Sleep(freq)
			}
		},
	}
}

func (e *Engine) newReleaseImageHostPingCheck(config Config) *Check {
	ctype := CheckTypeReleaseImageHostPing
	return &Check{
		Type: ctype,
		Freq: 5 * time.Second,
		Run: func(c chan CheckResult, freq time.Duration) {
			for {
				checkFunction := func() ([]byte, error) {
					return exec.Command("ping", "-c", "4", config.ReleaseImageHostname).CombinedOutput()
				}
				c <- e.createCheckResult(checkFunction, ctype)
				time.Sleep(freq)
			}
		},
	}
}

// This check verifies whether there is a http server response
// when schemeHostnamePort is queried. This does not check
// if the release image exists.
func (e *Engine) newReleaseImageHttpCheck(config Config) *Check {

	ctype := CheckTypeReleaseImageHttp
	return &Check{
		Type: ctype,
		Freq: 5 * time.Second,
		Run: func(c chan CheckResult, freq time.Duration) {
			for {
				checkFunction := func() ([]byte, error) {
					resp, err := http.Get(config.ReleaseImageSchemeHostnamePort)
					if err != nil {
						return []byte(err.Error()), err
					} else {
						// server replied with http response
						// as long as there is a response, the check
						// is a success.
						return []byte(resp.Status), err
					}
				}
				c <- e.createCheckResult(checkFunction, ctype)
				time.Sleep(freq)
			}
		},
	}
}

func NewEngine(c chan CheckResult, config Config) *Engine {
	checks := []*Check{}
	logger := logrus.New()

	// initialize log
	f, err := os.OpenFile(config.LogPath, os.O_RDWR|os.O_CREATE, 0644)
	if errors.Is(err, os.ErrNotExist) {
		// handle the case where the file doesn't exist
		fmt.Printf("Error creating log file %s\n", config.LogPath)
	}
	logger.Out = f

	logger.Infof("Release Image URL: %s", config.ReleaseImageURL)

	e := &Engine{
		checks:  checks,
		channel: c,
		logger:  logger,
	}

	e.checks = []*Check{
		e.newRegistryImagePullCheck(config),
		e.newReleaseImageHostDNSCheck(config),
		e.newReleaseImageHostPingCheck(config),
		e.newReleaseImageHttpCheck(config),
	}

	return e
}

func (e *Engine) Init() {
	for _, chk := range e.checks {
		go chk.Run(e.channel, chk.Freq)
	}
}

func (e *Engine) Size() int {
	return len(e.checks)
}
