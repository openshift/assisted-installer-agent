package subsystem

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

const (
	stepNTPID        = "ntp-synchronizer-step"
	singleRunTimeout = 10 * time.Second
)

var _ = Describe("NTP tests", func() {
	var (
		hostID          string
		numberOfSources int
	)

	BeforeEach(func() {
		resetAll()
		hostID = nextHostID()

		addChronyDaemonStub(hostID)
		_, _ = addRegisterStub(hostID, http.StatusCreated, ClusterID)
		setReplyStartAgent(hostID)
		waitforChronyDaemonToStart(hostID)
	})

	It("get_sources", func() {
		setNTPSyncRequestStub(hostID, models.NtpSynchronizationRequest{})

		ntpResponse := getNTPResponse(hostID)
		Expect(ntpResponse).ShouldNot(BeNil())
		numberOfSources = len(ntpResponse.NtpSources)
	})

	It("add_server", func() {
		server := "1.1.1.1"

		setNTPSyncRequestStub(hostID, models.NtpSynchronizationRequest{NtpSource: &server})

		ntpResponse := getNTPResponse(hostID)
		Expect(ntpResponse).ShouldNot(BeNil())
		Expect(ntpResponse.NtpSources).Should(HaveLen(numberOfSources + 1))
		Expect(isSourceInList(server, ntpResponse.NtpSources)).Should(BeTrue())
	})
})

func isSourceInList(sourceName string, ls []*models.NtpSource) bool {
	for _, source := range ls {
		if source.SourceName == sourceName {
			return true
		}
	}

	return false
}

func setNTPSyncRequestStub(hostID string, request models.NtpSynchronizationRequest) {
	b, err := json.Marshal(&request)
	Expect(err).ShouldNot(HaveOccurred())

	_, err = addNextStepStub(hostID, 100, "",
		&models.Step{
			StepType: models.StepTypeNtpSynchronizer,
			StepID:   stepNTPID,
			Command:  "nsenter",
			Args: []string{"-t", "1", "-m", "-i", "--",
				"/usr/bin/ntp_synchronizer",
				string(b),
			},
		},
	)
	Expect(err).NotTo(HaveOccurred())
}

func getNTPResponse(hostID string) *models.NtpSynchronizationResponse {
	Eventually(func() bool {
		return isReplyFound(hostID, &NTPSynchronizerVerifier{})
	}, 30*time.Second, 5*time.Second).Should(BeTrue())

	stepReply := getSpecificStep(hostID, &NTPSynchronizerVerifier{})
	return getNTPResponseFromStepReply(stepReply)
}

type NTPSynchronizerVerifier struct{}

func (i *NTPSynchronizerVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		log.Errorf("NTPSynchronizerVerifier returned with exit code %d. error: %s", actualReply.ExitCode, actualReply.Error)
		return false
	}
	if actualReply.StepType != models.StepTypeNtpSynchronizer {
		// Because of chronyd this might not be the only reply, so we just skip
		return false
	}
	var response models.NtpSynchronizationResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	if err != nil {
		log.Errorf("NTPSynchronizerVerifier failed to unmarshal")
		return false
	}

	return true
}

func getNTPResponseFromStepReply(actualReply *models.StepReply) *models.NtpSynchronizationResponse {
	var response models.NtpSynchronizationResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	Expect(err).NotTo(HaveOccurred())
	return &response
}

/* ===== Chrony Daemon ===== */

const chronyDaemonStepType = models.StepType("chrony")

func addChronyDaemonStub(hostID string) {
	_, err := addNextStepStub(hostID, 10, "",
		createCustomStub(chronyDaemonStepType, "chronyd"),
		&models.Step{
			StepType: models.StepTypeExecute,
			Command:  "bash",
			Args: []string{
				"-c",
				"sleep 1; echo chronyd started",
			},
		},
	)
	Expect(err).ToNot(HaveOccurred())
}

func waitforChronyDaemonToStart(hostID string) {
	EventuallyWithOffset(1, func() bool {
		return isReplyFound(hostID, &EqualReplyVerifier{
			Output:   "chronyd started\n",
			StepType: models.StepTypeExecute,
		})
	}, 30*time.Second, 500*time.Millisecond).Should(BeTrue())
}

type ChronyDaemonVerifier struct{}

func (*ChronyDaemonVerifier) verify(actualReply *models.StepReply) bool {
	return actualReply.StepType == chronyDaemonStepType
}
