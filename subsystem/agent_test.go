package subsystem

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

const (
	InfraEnvID                     = "11111111-1111-1111-1111-111111111111"
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
			Error:    "Missing command",
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

	It("Execute echo", func() {
		hostID := nextHostID()
		registerStubID, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		Expect(err).NotTo(HaveOccurred())
		stepID := "execute-step"
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "", &models.Step{
			StepType: models.StepTypeExecute,
			StepID:   stepID,
			Command:  "echo",
			Args: []string{
				"Hello",
				"world",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(1 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		expectedReply := &EqualReplyVerifier{
			Error:    "",
			ExitCode: 0,
			Output:   "Hello world\n",
			StepID:   stepID,
			StepType: models.StepTypeExecute,
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
		nextStepsStubID, err := addNextStepStub(hostID, defaultnextInstructionSeconds, "",
			&models.Step{
				StepType: models.StepTypeExecute,
				StepID:   "echo-step-1",
				Command:  "echo",
				Args: []string{
					"Hello",
					"world",
				},
			},
			&models.Step{
				StepType: models.StepTypeExecute,
				StepID:   "echo-step-2",
				Command:  "echo",
				Args: []string{
					"Bye",
					"bye",
					"world",
				},
			},
			generateContainerStep(models.StepTypeInventory,
				[]string{
					"--net=host",
					"-v", "/sys/block:/host/sys/block:ro", // discover disks by inventory
					"-v", "/sys/class/net:/sys/class/net:ro", // discover network interfaces by inventory

					// Asked by ghw - https://github.com/jaypipes/ghw/blob/master/pkg/linuxpath/path_linux.go
					// Currently, we don't use all of ghw capabilities. But let's keep them for now as it's harmless.
					"-v", "/run/udev:/run/udev",
					"-v", "/dev/disk:/dev/disk",
					"-v", "/var/log:/host/var/log:ro",
					"-v", "/proc/meminfo:/host/proc/meminfo:ro",
					"-v", "/sys/kernel/mm/hugepages:/host/sys/kernel/mm/hugepages:ro",
					"-v", "/proc/cpuinfo:/host/proc/cpuinfo:ro",
					"-v", "/etc/mtab:/host/etc/mtab:ro",
					"-v", "/sys/devices:/host/sys/devices:ro",
					"-v", "/sys/bus:/host/sys/bus:ro",
					"-v", "/sys/class:/host/sys/class:ro",
					"-v", "/run/udev:/host/run/udev:ro",
					"-v", "/dev/disk:/host/dev/disk:ro",
				},
				[]string{"/usr/bin/inventory"}),
		)
		Expect(err).NotTo(HaveOccurred())
		replyStubID, err := addStepReplyStub(hostID)
		Expect(err).NotTo(HaveOccurred())
		Expect(startAgent()).NotTo(HaveOccurred())
		time.Sleep(5 * time.Second)
		verifyRegisterRequest()
		verifyGetNextRequest(hostID, true)
		Expect(isReplyFound(hostID, &EqualReplyVerifier{
			Error:    "",
			ExitCode: 0,
			Output:   "Hello world\n",
			StepID:   "echo-step-1",
			StepType: models.StepTypeExecute,
		})).Should(BeTrue())
		Expect(isReplyFound(hostID, &EqualReplyVerifier{
			Error:    "",
			ExitCode: 0,
			Output:   "Bye bye world\n",
			StepID:   "echo-step-2",
			StepType: models.StepTypeExecute,
		})).Should(BeTrue())
		Eventually(func() bool {
			return isReplyFound(hostID, &InventoryVerifier{})
		}, 10*time.Second, 2*time.Second).Should(BeTrue())
		stepReply := getSpecificStep(hostID, &InventoryVerifier{})
		inventory := getInventoryFromStepReply(stepReply)
		Expect(len(inventory.Interfaces) > 0).To(BeTrue())
		freeAddressesRequest := models.FreeAddressesRequest{}
		for _, intf := range inventory.Interfaces {
			for _, ipAddr := range intf.IPV4Addresses {
				var ip net.IP
				var cidr *net.IPNet
				ip, cidr, err = net.ParseCIDR(ipAddr)
				Expect(err).ToNot(HaveOccurred())
				ones, _ := cidr.Mask.Size()
				if ones < 24 {
					_, cidr, err = net.ParseCIDR(ip.To4().String() + "/24")
					Expect(err).ToNot(HaveOccurred())
				}
				freeAddressesRequest = append(freeAddressesRequest, cidr.String())
			}
		}
		if len(freeAddressesRequest) > 0 {
			// TODO:: Need to support this part for all hosts.  Currently, we so a case that only virtual nics have ip addresses
			var b []byte
			b, err = json.Marshal(&freeAddressesRequest)
			Expect(err).ToNot(HaveOccurred())
			err = deleteStub(nextStepsStubID)
			Expect(err).NotTo(HaveOccurred())

			step := generateContainerStep(models.StepTypeFreeNetworkAddresses,
				[]string{"--net=host"},
				[]string{"/usr/bin/free_addresses", string(b)})
			_, err = addNextStepStub(hostID, defaultnextInstructionSeconds, "", step)
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				return isReplyFound(hostID, &FreeAddressesVerifier{})
			}, maxTimeout, 5*time.Second).Should(BeTrue())
		}
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

type InventoryVerifier struct{}

func (i *InventoryVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		return false
	}
	if actualReply.StepType != models.StepTypeInventory {
		return false
	}
	var inventory models.Inventory
	err := json.Unmarshal([]byte(actualReply.Output), &inventory)
	if err != nil {
		return false
	}
	return inventory.Memory != nil && inventory.Memory.UsableBytes > 0 && inventory.Memory.PhysicalBytes > 0 &&
		inventory.CPU != nil && inventory.CPU.Count > 0 &&
		len(inventory.Disks) > 0 &&
		len(inventory.Interfaces) > 0 &&
		inventory.Gpus != nil &&
		inventory.Hostname != ""
}

type FreeAddressesVerifier struct{}

func (f *FreeAddressesVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.StepType != models.StepTypeFreeNetworkAddresses {
		return false
	}
	Expect(actualReply.ExitCode).To(BeZero())
	var freeAddresses models.FreeNetworksAddresses
	Expect(json.Unmarshal([]byte(actualReply.Output), &freeAddresses)).ToNot(HaveOccurred())
	Expect(len(freeAddresses) > 0).To(BeTrue())
	_, _, err := net.ParseCIDR(freeAddresses[0].Network)
	Expect(err).ToNot(HaveOccurred())
	if len(freeAddresses[0].FreeAddresses) > 0 {
		ip := net.ParseIP(freeAddresses[0].FreeAddresses[0].String())
		Expect(ip).ToNot(BeNil())
		Expect(ip.To4()).ToNot(BeNil())
	}
	return true
}

func getInventoryFromStepReply(actualReply *models.StepReply) *models.Inventory {
	var inventory models.Inventory
	err := json.Unmarshal([]byte(actualReply.Output), &inventory)
	Expect(err).NotTo(HaveOccurred())
	return &inventory
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsystem Suite")
}
