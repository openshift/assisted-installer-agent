package subsystem

import (
	"encoding/json"
	"io/ioutil"

	//"io/ioutil"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

const (
	InfraEnvID                    = "11111111-1111-1111-1111-111111111111"
	defaultnextInstructionSeconds = int64(1)
	waitForWiremockTimeout        = 60 * time.Second
)

var log *logrus.Logger

var _ = Describe("Agent tests", func() {
	BeforeSuite(func() {
		Eventually(waitForWiremock, waitForWiremockTimeout, time.Second).ShouldNot(HaveOccurred())
		log = logrus.New()
	})

	BeforeEach(func() {
		Expect(stopAgent()).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		Expect(resetRequests()).NotTo(HaveOccurred())
		Expect(deleteAllStubs()).NotTo(HaveOccurred())
	})

	It("Happy flow", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Next step runner fails - default delay", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStubInvalidCommand(hostID, http.StatusCreated, InfraEnvID, -1)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)

		By("Validate only register request was called")
		verifyNumberOfRegisterRequest("==", 1)
		verifyNumberOfGetNextRequest(hostID, "==", 0)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Next step runner keeps failing - retry registration", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStubInvalidCommand(hostID, http.StatusCreated, InfraEnvID, 3)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(5 * time.Second)

		By("Validate only register request was called, at least twice")
		verifyNumberOfRegisterRequest(">", 1)
		verifyNumberOfGetNextRequest(hostID, "==", 0)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Next step runner fails once, retry succeeds", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStubInvalidCommand(hostID, http.StatusCreated, InfraEnvID, 5)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(3 * time.Second)

		By("Validate only register was called")
		verifyNumberOfRegisterRequest("==", 1)
		verifyNumberOfGetNextRequest(hostID, "==", 0)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())

		registerStubID, err = addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(6 * time.Second)

		By("Validate register and get next step were called after command changed")
		verifyNumberOfRegisterRequest("==", 2)
		verifyNumberOfGetNextRequest(hostID, ">", 0)

		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Next step exits - stop after next step and re-register", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, models.StepsPostStepActionExit)
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)

		By("Validate both register and next step called at least twice")
		verifyNumberOfRegisterRequest(">", 1)
		verifyNumberOfGetNextRequest(hostID, ">", 1)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Next step exits - don't stop and keep polling for next step", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, models.StepsPostStepActionContinue)
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)

		By("Validate register request was called only once, next step multiple times")
		verifyNumberOfRegisterRequest("==", 1)
		verifyNumberOfGetNextRequest(hostID, ">", 1)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	Context("register action makes the agent to wait forever", func() {
		var status int

		It("register conflicted - cluster does not accept hosts", func() {
			status = http.StatusConflict
		})

		It("register forbidden", func() {
			status = http.StatusForbidden
		})

		AfterEach(func() {
			hostID := nextHostID()
			registerStubID, err := addRegisterStub(hostID, status, InfraEnvID)
			Expect(err).NotTo(HaveOccurred())

			nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
			Expect(err).NotTo(HaveOccurred())

			Expect(startAgent()).NotTo(HaveOccurred())
			time.Sleep(10 * time.Second)

			By("Validate only register request was called")
			resp, err := http.Get(RequestsURL)
			Expect(err).ShouldNot(HaveOccurred())
			requests := &Requests{}
			b, err := ioutil.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(json.Unmarshal(b, &requests)).ShouldNot(HaveOccurred())
			req := make([]*RequestOccurrence, 0, len(requests.Requests))
			req = append(req, requests.Requests...)
			Expect(len(req)).Should(Equal(1))
			Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
			Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
		})
	})

	It("register not found - agent should stop trying to register", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusNotFound, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())

		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())

		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)

		By("Validate only register request was called")
		resp, err := http.Get(RequestsURL)
		Expect(err).ShouldNot(HaveOccurred())
		requests := &Requests{}
		b, err := ioutil.ReadAll(resp.Body)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(json.Unmarshal(b, &requests)).ShouldNot(HaveOccurred())
		req := make([]*RequestOccurrence, 0, len(requests.Requests))
		req = append(req, requests.Requests...)
		Expect(len(req)).Should(Equal(1))
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Verify nextInstructionSeconds", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(5 * time.Second)
		verifyRegisterRequest()
		verifyNumberOfGetNextRequest(hostID, ">", 3)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
		Expect(stopAgent()).NotTo(HaveOccurred())

		By("verify changing nextInstructionSeconds to large number")
		hostID = nextHostID()
		registerStubID, err = addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err = addNextStepStub(hostID, 100, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(30 * time.Second)
		verifyRegisterRequest()
		verifyNumberOfGetNextRequest(hostID, "<", 2)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Cluster not exists", func() {
		hostID := nextHostID()
		registerStubID, err := addNextStepClusterNotExistsStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(10 * time.Second)
		verifyRegisterRequest()
		verifyNumberOfGetNextRequest(hostID, "<", 2)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Register recovery", func() {
		hostID := nextHostID()
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, false)
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 6)
		verifyRegisterRequest()
		verifyRegistersSameID()
		verifyGetNextRequest(hostID, true)
		Expect(deleteStub(registerStubID)).NotTo(HaveOccurred())
		Expect(deleteStub(nextStepsStubID)).NotTo(HaveOccurred())
	})

	It("Step not exists", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		stepID := "wrong-step"
		stepType := models.StepType("Step-not-exists")
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "", &models.Step{StepType: stepType, StepID: stepID, Args: make([]string, 0)})
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		expectedReply := &EqualReplyVerifier{
			Error:    "failed to find action for step type Step-not-exists",
			ExitCode: -1,
			Output:   "",
			StepID:   stepID,
			StepType: stepType,
		}
		Expect(isReplyFound(hostID, expectedReply)).Should(BeTrue())
		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(nextStepsStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Multiple steps", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		images := []string{"quay.io/coreos/etcd:latest"}
		removeImage(defaultContainerTool, images[0])
		url := WireMockURLFromSubsystemHost + TestWorkerIgnitionPath
		_, err = addWorkerIgnitionStub()
		Expect(err).NotTo(HaveOccurred())
		b, err := json.Marshal(&models.APIVipConnectivityRequest{
			URL: &url,
		})
		Expect(err).ShouldNot(HaveOccurred())

		_, err = addNextStepStub(hostID, defaultnextInstructionSeconds, "",
			createContainerImageAvailabilityStep(models.ContainerImageAvailabilityRequest{Images: images, Timeout: 60}),
			generateStep(models.StepTypeAPIVipConnectivityCheck,
				[]string{string(b)}),
		)
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(5 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		checkImageAvailabilityResponse(hostID, images, true, true, 60)

		Eventually(func() bool {
			return isReplyFound(hostID, &APIConnectivityCheckVerifier{})
		}, maxTimeout, 5*time.Second).Should(BeTrue())

		err = deleteStub(registerStubID)
		Expect(err).NotTo(HaveOccurred())
		err = deleteStub(replyStubID)
		Expect(err).NotTo(HaveOccurred())
	})
})

type EqualReplyVerifier models.StepReply

func (e *EqualReplyVerifier) verify(actualReply *models.StepReply) bool {
	if *(*models.StepReply)(e) != *actualReply {
		log.Errorf("expected step: %+v actual step: %+v", *(*models.StepReply)(e), *actualReply)
		return false
	}

	return true
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsystem Suite")
}
