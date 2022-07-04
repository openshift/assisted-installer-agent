package disk_speed_check

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-openapi/swag"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disk speed check test", func() {
	var dependencies *MockIDependencies
	var perfCheck *DiskSpeedCheck
	var log logrus.FieldLogger
	var subprocessConfig *config.SubprocessConfig

	BeforeEach(func() {
		subprocessConfig = &config.SubprocessConfig{}
		dependencies = &MockIDependencies{}
		perfCheck = NewDiskSpeedCheck(subprocessConfig, dependencies)
		log = logrus.New()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	fioSuccess := func(file string, durationInMS int64) {
		durationInNS := durationInMS * int64(time.Millisecond)
		dependencies.On("Execute", "fio", "--filename", file,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return(fmt.Sprintf(
			`{ "jobs": 
					[{
						"sync":
						{ "lat_ns":
							{ "percentile":
								{ "99.000000": %d }
							}
						}
					}]
				}`, durationInNS), "", 0).Once()
	}

	fioFailure := func(file string) {
		dependencies.On("Execute", "fio", "--filename", file,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return("", "failure", -1).Once()
	}

	It("Check succeeded", func() {
		path := "/dev/disk"
		for i := 1; i <= numOfFioJobs; i++ {
			fioSuccess(path, 2)
		}
		stdout, _, exitCode := perfCheck.FioPerfCheck(getRequestStr(path), log)
		Expect(exitCode).Should(Equal(0))
		checkReturn(stdout, path, 2)
	})

	It("Check succeeded - various latencies", func() {
		path := "/dev/disk"
		for i := 1; i <= numOfFioJobs; i++ {
			fioSuccess(path, int64(i)*2)
		}
		stdout, _, exitCode := perfCheck.FioPerfCheck(getRequestStr(path), log)
		Expect(exitCode).Should(Equal(0))
		checkReturn(stdout, path, numOfFioJobs*2)
	})

	It("Check failed - unmarshal error", func() {
		_, stderr, exitCode := perfCheck.FioPerfCheck("", log)
		Expect(exitCode).To(Equal(-1))
		Expect(stderr).To(Equal("Failed to unmarshal DiskSpeedCheckRequest: unexpected end of JSON input"))
	})

	It("Check failed - missing path", func() {
		_, stderr, exitCode := perfCheck.FioPerfCheck(getRequestStr(""), log)
		Expect(exitCode).To(Equal(-1))
		Expect(stderr).To(Equal("Missing disk path"))
	})

	It("Check failed - all fio requests returned errors", func() {
		path := "/dev/disk"
		for i := 1; i <= numOfFioJobs; i++ {
			fioFailure(path)
		}
		_, stderr, exitCode := perfCheck.FioPerfCheck(getRequestStr(path), log)
		Expect(exitCode).To(Equal(-1))
		Expect(stderr).To(Equal("Could not get I/O performance for path /dev/disk (fio exit code: -1, stderr: failure)"))
	})

	It("Check succeeded - one fio request returned an error", func() {
		path := "/dev/disk"
		fioFailure(path)
		for i := 1; i <= numOfFioJobs-1; i++ {
			fioSuccess(path, 1)
		}
		stdout, _, exitCode := perfCheck.FioPerfCheck(getRequestStr(path), log)
		Expect(exitCode).Should(Equal(0))
		checkReturn(stdout, path, 1)
	})
})

func getRequestStr(path string) string {
	request := models.DiskSpeedCheckRequest{
		Path: swag.String(path),
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return ""
	}
	return string(requestBytes)
}

func checkReturn(stdout, path string, duration int64) {
	var response models.DiskSpeedCheckResponse
	Expect(json.Unmarshal([]byte(stdout), &response)).ToNot(HaveOccurred())
	Expect(path).To(Equal(response.Path))
	Expect(duration).To(Equal(response.IoSyncDuration))
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FIO performance check check unit tests")
}
