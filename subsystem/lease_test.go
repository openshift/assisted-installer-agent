package subsystem

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

const (
	stepDHCPID = "dhcp-lease-allocate-step"
)

var _ = Describe("Lease tests", func() {
	var (
		hostID     string
		apiMac     strfmt.MAC
		ingressMac strfmt.MAC
		ifaceName  string
	)

	BeforeEach(func() {
		resetAll()
		hostID = nextHostID()

		apiMac = generateMacString()
		ingressMac = generateMacString()
		ifaceName = "eth0"
	})

	It("same_mac_same_ip", func() {
		var dhcpResponse *models.DhcpAllocationResponse

		By("1st time", func() {
			setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			dhcpResponse = getDHCPResponse(hostID)
			Expect(dhcpResponse).ShouldNot(BeNil())
		})

		resetAll()

		By("2nd time", func() {
			setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			Expect(*getDHCPResponse(hostID)).Should(Equal(*dhcpResponse))
		})
	})

	It("different_mac_different_ip", func() {
		var dhcpResponse *models.DhcpAllocationResponse

		By("1st time", func() {
			setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			dhcpResponse = getDHCPResponse(hostID)
			Expect(dhcpResponse).ShouldNot(BeNil())
		})

		resetAll()

		By("2nd time", func() {
			apiMac = generateMacString()
			ingressMac = generateMacString()

			setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			Expect(*getDHCPResponse(hostID)).ShouldNot(Equal(*dhcpResponse))
		})
	})

	Context("Negative", func() {
		It("no_dhcp_server", func() {
			By("stop dhcpd", func() {
				stopContainer("dhcpd")

				setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
					APIVipMac:     &apiMac,
					IngressVipMac: &ingressMac,
					Interface:     &ifaceName,
				})

				setReplyStartAgent(hostID)
				Eventually(isReplyFound(hostID, &DHCPLeaseAllocateVerifier{})).Should(BeFalse())
			})

			By("restart dhcpd", func() {
				startContainer("dhcpd")

				Eventually(func() bool {
					return isReplyFound(hostID, &DHCPLeaseAllocateVerifier{})
				}, 300*time.Second, 5*time.Second).Should(BeTrue())
			})
		})

		It("invalid_mac", func() {
			apiMac = "invalid-mac"

			setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			setReplyStartAgent(hostID)
			Eventually(isReplyFound(hostID, &EqualReplyVerifier{
				Error:    fmt.Sprintf("address %s: invalid MAC address", apiMac),
				ExitCode: 255,
				Output:   "",
				StepID:   stepDHCPID,
				StepType: models.StepTypeDhcpLeaseAllocate,
			})).Should(BeTrue())
		})

		It("invalid_interface", func() {
			ifaceName = "invalid-interface"

			setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			setReplyStartAgent(hostID)
			Eventually(isReplyFound(hostID, &EqualReplyVerifier{
				Error:    "numerical result out of range",
				ExitCode: 255,
				Output:   "",
				StepID:   stepDHCPID,
				StepType: models.StepTypeDhcpLeaseAllocate,
			})).Should(BeTrue())
		})
	})
})

func setDHCPLeaseRequestStub(hostID string, request models.DhcpAllocationRequest) {
	_, err := addRegisterStub(hostID, http.StatusCreated)
	Expect(err).NotTo(HaveOccurred())

	b, err := json.Marshal(&request)
	Expect(err).ShouldNot(HaveOccurred())

	_, err = addNextStepStub(hostID, 5,
		&models.Step{
			StepType: models.StepTypeDhcpLeaseAllocate,
			StepID:   stepDHCPID,
			Command:  "docker",
			Args: []string{
				"run", "--privileged", "--net=subsystem_agent_network", "--rm",
				"-v", "/var/log:/var/log",
				"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
				"quay.io/ocpmetal/assisted-installer-agent:latest",
				"dhcp_lease_allocate",
				string(b),
			},
		},
	)
	Expect(err).NotTo(HaveOccurred())
}

func setReplyStartAgent(hostID string) {
	_, err := addStepReplyStub(hostID)
	Expect(err).NotTo(HaveOccurred())
	Expect(startAgent()).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	verifyRegisterRequest()
	verifyGetNextRequest(hostID, true)
}

func getDHCPResponse(hostID string) *models.DhcpAllocationResponse {
	setReplyStartAgent(hostID)
	Eventually(func() bool {
		return isReplyFound(hostID, &DHCPLeaseAllocateVerifier{})
	}, 30*time.Second, 5*time.Second).Should(BeTrue())

	stepReply := getSpecificStep(hostID, &DHCPLeaseAllocateVerifier{})
	return getLeaseResponseFromStepReply(stepReply)
}

type DHCPLeaseAllocateVerifier struct{}

func (i *DHCPLeaseAllocateVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		log.Errorf("DHCPLeaseAllocateVerifier returned with exit code %d. error: %s", actualReply.ExitCode, actualReply.Error)
		return false
	}
	if actualReply.StepType != models.StepTypeDhcpLeaseAllocate {
		log.Errorf("DHCPLeaseAllocateVerifier invalid step replay %s", actualReply.StepType)
		return false
	}
	var response models.DhcpAllocationResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	if err != nil {
		log.Errorf("DHCPLeaseAllocateVerifier failed to unmarshal")
		return false
	}

	return response.APIVipAddress != nil && *response.APIVipAddress != "" &&
		response.IngressVipAddress != nil && *response.IngressVipAddress != ""
}

func getLeaseResponseFromStepReply(actualReply *models.StepReply) *models.DhcpAllocationResponse {
	var response models.DhcpAllocationResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	Expect(err).NotTo(HaveOccurred())
	return &response
}

func generateMacString() strfmt.MAC {
	var MacPrefixQumranet = [...]byte{0x00, 0x1A, 0x4A}

	buf := make([]byte, 3)
	_, err := rand.Read(buf)
	Expect(err).ShouldNot(HaveOccurred())

	return strfmt.MAC(fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		MacPrefixQumranet[0], MacPrefixQumranet[1], MacPrefixQumranet[2], buf[0], buf[1], buf[2]))
}
