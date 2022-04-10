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

	It("Sufficient performance", func() {
		path := "/dev/disk"
		fioSuccess(path, 2)
		stdout, _, exitCode := perfCheck.FioPerfCheck(getRequestStr(path), log)
		Expect(exitCode).Should(Equal(0))
		checkReturn(stdout, path, 2)
	})

	It("Slow disk performance", func() {
		path := "/dev/disk"
		fioSuccess(path, 200)
		stdout, _, exitCode := perfCheck.FioPerfCheck(getRequestStr(path), log)
		Expect(exitCode).To(Equal(0))
		checkReturn(stdout, path, 200)
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
