package fio_perf_check

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	FioPerfCheckCmdExitCode int64 = 126
	FioDurationThreshold    int64 = 10
)

var _ = Describe("FIO performance check test", func() {
	var dependencies *MockIDependencies
	var perfCheck *PerfCheck
	var log logrus.FieldLogger

	var duration int64 = FioDurationThreshold
	var cmdExitCode int64 = FioPerfCheckCmdExitCode

	BeforeEach(func() {
		dependencies = &MockIDependencies{}
		perfCheck = NewPerfCheck(dependencies)
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

	It("Sufficient performance", func() {
		path := "/dev/disk"
		fioSuccess(path, 2)
		_, _, exitCode := perfCheck.FioPerfCheck(getRequestStr(&path, &duration, &cmdExitCode), log)
		Expect(exitCode).Should(Equal(0))
	})

	It("Slow disk performance", func() {
		path := "/dev/disk"
		fioSuccess(path, 200)
		_, _, exitCode := perfCheck.FioPerfCheck(getRequestStr(&path, &duration, &cmdExitCode), log)
		Expect(exitCode).Should(Equal(int(cmdExitCode)))
	})
})

func getRequestStr(path *string, threshold *int64, exitCode *int64) string {
	request := models.FioPerfCheckRequest{
		Path:                path,
		DurationThresholdMs: threshold,
		ExitCode:            exitCode,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return ""
	}
	return string(requestBytes)
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FIO performance check check unit tests")
}
