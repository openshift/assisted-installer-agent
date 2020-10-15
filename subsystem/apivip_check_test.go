package subsystem

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/assisted-installer-agent/src/apivip_check"
	"github.com/openshift/assisted-service/models"
)

const (
	stepAPIConnectivityID = "apivip-connectivity-check-step"
)

var _ = Describe("API VIP connectivity check tests", func() {
	var (
		hostID string
	)

	BeforeEach(func() {
		resetAll()
		hostID = nextHostID()
	})

	It("verify API connectivity", func() {
		url := "http://127.0.0.1:8362"
		setWorkerIgnitionStub(hostID, &models.APIVipConnectivityRequest{
			URL: &url,
		})
		setReplyStartAgent(hostID)

		Eventually(func() bool {
			return isReplyFound(hostID, &APIConnectivityCheckVerifier{})
		}, 300*time.Second, 5*time.Second).Should(BeTrue())
	})
})

func setWorkerIgnitionStub(hostID string, request *models.APIVipConnectivityRequest) {
	_, err := addRegisterStub(hostID, http.StatusCreated, ClusterID)
	Expect(err).NotTo(HaveOccurred())

	_, err = addWorkerIgnitionStub()
	Expect(err).NotTo(HaveOccurred())

	b, err := json.Marshal(&request)
	Expect(err).ShouldNot(HaveOccurred())

	_, err = addNextStepStub(hostID, 5,
		&models.Step{
			StepType: models.StepTypeAPIVipConnectivityCheck,
			StepID:   stepAPIConnectivityID,
			Command:  "docker",
			Args: []string{
				"run", "--privileged", "--net=host", "--rm",
				"-v", "/var/log:/var/log",
				"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
				"quay.io/ocpmetal/assisted-installer-agent:latest",
				"apivip_check",
				string(b),
			},
		},
	)
	Expect(err).NotTo(HaveOccurred())
}

type APIConnectivityCheckVerifier struct{}

func (i *APIConnectivityCheckVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		log.Errorf("APIConnectivityCheckVerifier returned with exit code %d. error: %s", actualReply.ExitCode, actualReply.Error)
		return false
	}
	if actualReply.StepType != models.StepTypeAPIVipConnectivityCheck {
		log.Errorf("APIConnectivityCheckVerifier invalid step replay %s", actualReply.StepType)
		return false
	}

	return int(actualReply.ExitCode) == 0
}

func addWorkerIgnitionStub() (string, error) {
	ignitionConfig, err := apivip_check.FormatNodeIgnitionFile(WireMockURL + apivip_check.WorkerIgnitionPath)
	if err != nil {
		return "", err
	}
	stub := StubDefinition{
		Request: &RequestDefinition{
			URL:    apivip_check.WorkerIgnitionPath,
			Method: "GET",
		},
		Response: &ResponseDefinition{
			Status: 200,
			Body:   string(ignitionConfig),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	return addStub(&stub)
}
