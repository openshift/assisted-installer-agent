package subsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	agentUtils "github.com/openshift/assisted-installer-agent/src/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/thoas/go-funk"
)

const (
	timeBetweenSteps = 3
)

var _ = Describe("NTP tests", func() {
	var (
		hostID string
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		resetAll()
		hostID = nextHostID()
	})
	AfterEach(func() {
		cancel()
	})

	Context("add_new_server", func() {
		It("IP", func() {
			server := "1.2.3.4"
			startNTPSynchronizer(ctx, hostID, models.NtpSynchronizationRequest{NtpSource: &server})
			ntpResponse := getNTPResponse(hostID, []string{server})
			Expect(ntpResponse).ShouldNot(BeNil())
			printNtpSources(ntpResponse)
			Expect(isSourceInList(server, ntpResponse.NtpSources)).Should(BeTrue())
		})

		It("Hostname", func() {
			server := "dns.google"
			startNTPSynchronizer(ctx, hostID, models.NtpSynchronizationRequest{NtpSource: &server})
			ntpResponse := getNTPResponse(hostID, []string{server})
			Expect(ntpResponse).ShouldNot(BeNil())
			printNtpSources(ntpResponse)
			Expect(isSourceInList(server, ntpResponse.NtpSources)).Should(BeTrue())
		})
	})

	It("add_existing_server", func() {
		server := "2.2.2.2"
		startNTPSynchronizer(ctx, hostID, models.NtpSynchronizationRequest{NtpSource: &server})

		By("Add server 1st time", func() {
			ntpResponse := getNTPResponse(hostID, []string{server})
			Expect(ntpResponse).ShouldNot(BeNil())
			printNtpSources(ntpResponse)
			Expect(isSourceInList(server, ntpResponse.NtpSources)).Should(BeTrue())
		})

		// 2nd time
		By("Add server 2nd time", func() {
			ntpResponse := getNTPResponse(hostID, []string{server})
			Expect(ntpResponse).ShouldNot(BeNil())
			printNtpSources(ntpResponse)
			Expect(isSourceInList(server, ntpResponse.NtpSources)).Should(BeTrue())
		})
	})

	It("add_multiple_servers", func() {
		servers := []string{"1.1.1.3", "1.1.1.4", "1.1.1.5"}
		serversAsString := strings.Join(servers, ",")
		startNTPSynchronizer(ctx, hostID, models.NtpSynchronizationRequest{NtpSource: &serversAsString})
		ntpResponse := getNTPResponse(hostID, servers)
		Expect(ntpResponse).ShouldNot(BeNil())
		printNtpSources(ntpResponse)

		for _, server := range servers {
			Expect(isSourceInList(server, ntpResponse.NtpSources)).Should(BeTrue())
		}
	})
})

func printNtpSources(ntpResponse *models.NtpSynchronizationResponse) {
	for _, source := range ntpResponse.NtpSources {
		fmt.Printf("NTP source %s - %s\n", source.SourceName, source.SourceState)
	}
}

func startNTPSynchronizer(ctx context.Context, hostId string, request models.NtpSynchronizationRequest) {
	_, _ = addRegisterStub(hostId, http.StatusCreated, InfraEnvID)
	setReplyStartAgent(hostId)
	addChronyDaemon(ctx, hostId)
	setNTPSyncRequestStub(hostId, request)
	_, err := addStepReplyStub(hostId)
	Expect(err).NotTo(HaveOccurred())
}

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
	step := generateStep(models.StepTypeNtpSynchronizer, []string{string(b)})
	_, err = addNextStepStub(hostID, timeBetweenSteps, "", step)
	Expect(err).NotTo(HaveOccurred())
}

func getNTPResponse(hostID string, expectedNtpSources []string) *models.NtpSynchronizationResponse {
	Eventually(func() bool {
		return isReplyFound(hostID, &NTPSynchronizerVerifier{expectedNtpSources})
	}, 120*time.Second, timeBetweenSteps*time.Second).Should(BeTrue())

	stepReply := getSpecificStep(hostID, &NTPSynchronizerVerifier{expectedNtpSources})
	return getNTPResponseFromStepReply(stepReply)
}

type NTPSynchronizerVerifier struct {
	expectedNtpSources []string
}

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
		log.Errorf("NTPSynchronizerVerifier failed to unmarshal %s", actualReply.Output)
		return false
	}

	// Verify the response have all the verifier expectedNtpSources
	names := make([]string, len(response.NtpSources))

	for index, source := range response.NtpSources {
		names[index] = source.SourceName
	}

	for _, source := range i.expectedNtpSources {
		if !funk.Contains(names, source) {
			return false
		}
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

func addChronyDaemon(ctx context.Context, hostID string) {
	s, e, errorCode := agentUtils.Execute("docker", []string{"exec", "agent", "chronyd"}...)
	fmt.Println(s, e, errorCode)
	Expect(errorCode).To(Equal(0))
}
