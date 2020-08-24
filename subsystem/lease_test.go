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
	"github.com/prometheus/common/log"
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
			SetDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			dhcpResponse = GetDHCPResponse(hostID)
			Expect(dhcpResponse).ShouldNot(BeNil())
		})

		resetAll()

		By("2nd time", func() {
			SetDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			Expect(*GetDHCPResponse(hostID)).Should(Equal(*dhcpResponse))
		})
	})

	It("different_mac_different_ip", func() {
		var dhcpResponse *models.DhcpAllocationResponse

		By("1st time", func() {
			SetDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			dhcpResponse = GetDHCPResponse(hostID)
			Expect(dhcpResponse).ShouldNot(BeNil())
		})

		resetAll()

		By("2nd time", func() {
			apiMac = generateMacString()
			ingressMac = generateMacString()

			SetDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			Expect(*GetDHCPResponse(hostID)).ShouldNot(Equal(*dhcpResponse))
		})
	})
})

func SetDHCPLeaseRequestStub(hostID string, request models.DhcpAllocationRequest) {
	_, err := addRegisterStub(hostID, http.StatusCreated)
	Expect(err).NotTo(HaveOccurred())

	b, err := json.Marshal(&request)
	Expect(err).ShouldNot(HaveOccurred())

	_, err = addNextStepStub(hostID, 100,
		&models.Step{
			StepType: models.StepTypeDhcpLeaseAllocate,
			StepID:   "dhcp-lease-allocate-step",
			Command:  "docker",
			Args: []string{
				"run", "--privileged", "--net=subsystem_agent_network", "--rm",
				"-v", "/var/log:/var/log",
				"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
				"quay.io/ocpmetal/assisted-installer-agent:latest",
				"dhcp_lease_allocator",
				string(b),
			},
		},
	)
	Expect(err).NotTo(HaveOccurred())
}

func GetDHCPResponse(hostID string) *models.DhcpAllocationResponse {
	_, err := addStepReplyStub(hostID)
	Expect(err).NotTo(HaveOccurred())
	Expect(startAgent()).NotTo(HaveOccurred())
	time.Sleep(5 * time.Second)
	verifyRegisterRequest()
	verifyGetNextRequest(hostID, true)
	Eventually(func() bool {
		return isReplyFound(hostID, &DHCPLeaseAllocatorVerifier{})
	}, 300*time.Second, 5*time.Second).Should(BeTrue())

	stepReply := getSpecificStep(hostID, &DHCPLeaseAllocatorVerifier{})
	return getLeaseResponseFromStepReply(stepReply)
}

type DHCPLeaseAllocatorVerifier struct{}

func (i *DHCPLeaseAllocatorVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		log.Errorf("DHCPLeaseAllocatorVerifier returned with exit code %d", actualReply.ExitCode)
		return false
	}
	if actualReply.StepType != models.StepTypeDhcpLeaseAllocate {
		log.Errorf("DHCPLeaseAllocatorVerifier invalid step replay %s", actualReply.StepType)
		return false
	}
	var response models.DhcpAllocationResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	if err != nil {
		log.Errorf("DHCPLeaseAllocatorVerifier failed to unmarshal")
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
