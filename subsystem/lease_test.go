package subsystem

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
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
			secondResponse := getDHCPResponse(hostID)
			Expect(secondResponse).ToNot(BeNil())
			Expect(dhcpResponse.APIVipAddress).To(Equal(secondResponse.APIVipAddress))
			Expect(dhcpResponse.IngressVipAddress).To(Equal(secondResponse.IngressVipAddress))
			Expect(secondResponse.APIVipAddress.String()).ToNot(BeEmpty())
			Expect(secondResponse.IngressVipAddress.String()).ToNot(BeEmpty())
			Expect(secondResponse.APIVipLease).To(ContainSubstring(secondResponse.APIVipAddress.String()))
			Expect(secondResponse.IngressVipLease).To(ContainSubstring(secondResponse.IngressVipAddress.String()))
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

			secondResponse := getDHCPResponse(hostID)
			Expect(secondResponse).ToNot(BeNil())
			Expect(dhcpResponse.APIVipAddress).ToNot(Equal(secondResponse.APIVipAddress))
			Expect(dhcpResponse.IngressVipAddress).ToNot(Equal(secondResponse.IngressVipAddress))
			Expect(secondResponse.APIVipAddress.String()).ToNot(BeEmpty())
			Expect(secondResponse.IngressVipAddress.String()).ToNot(BeEmpty())
			Expect(secondResponse.APIVipLease).To(ContainSubstring(secondResponse.APIVipAddress.String()))
			Expect(secondResponse.IngressVipLease).To(ContainSubstring(secondResponse.IngressVipAddress.String()))
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
	Context("Changing VIPs", func() {
		It("DHCPREQUEST with provided lease", func() {
			var dhcpResponse *models.DhcpAllocationResponse
			resetAll()
			By("1st time", func() {
				_, _ = addRegisterStub(hostID, http.StatusCreated)
				addTcpdumpStub(hostID, ifaceName, 11, 6)
				Expect(startAgent()).ToNot(HaveOccurred())
				waitforTcpdumpToStart(hostID)
				setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
					APIVipMac:     &apiMac,
					IngressVipMac: &ingressMac,
					Interface:     &ifaceName,
				})

				dhcpResponse = getDHCPResponse(hostID)
				Expect(dhcpResponse).ShouldNot(BeNil())
				tcpdumpResponse := getTcpdumpReponse(hostID)
				Expect(countSubstringOccurrences(tcpdumpResponse, "Discover")).To(BeNumerically(">=", 2))
				Expect(countSubstringOccurrences(tcpdumpResponse, "Request")).To(BeNumerically(">=", 2))
			})

			resetAll()

			By("2nd time", func() {

				addTcpdumpStub(hostID, ifaceName, 11, 4)
				_, _ = addRegisterStub(hostID, http.StatusCreated)
				Expect(startAgent()).ToNot(HaveOccurred())
				waitforTcpdumpToStart(hostID)
				setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
					APIVipMac:       &apiMac,
					IngressVipMac:   &ingressMac,
					Interface:       &ifaceName,
					APIVipLease:     formatLease(dhcpResponse.APIVipLease),
					IngressVipLease: formatLease(dhcpResponse.IngressVipLease),
				})

				secondResponse := getDHCPResponse(hostID)
				Expect(secondResponse).ToNot(BeNil())
				Expect(dhcpResponse.APIVipAddress).To(Equal(secondResponse.APIVipAddress))
				Expect(dhcpResponse.IngressVipAddress).To(Equal(secondResponse.IngressVipAddress))
				Expect(secondResponse.APIVipAddress.String()).ToNot(BeEmpty())
				Expect(secondResponse.IngressVipAddress.String()).ToNot(BeEmpty())
				Expect(secondResponse.APIVipLease).To(ContainSubstring(secondResponse.APIVipAddress.String()))
				Expect(secondResponse.IngressVipLease).To(ContainSubstring(secondResponse.IngressVipAddress.String()))
				tcpdumpResponse := getTcpdumpReponse(hostID)
				Expect(countSubstringOccurrences(tcpdumpResponse, "Discover")).To(Equal(0))
				Expect(countSubstringOccurrences(tcpdumpResponse, "Request")).To(BeNumerically(">=", 2))
			})
		})
	})
})

const TcpdumpStepType = models.StepType("tcpdump")

func formatLease(lease string) string {
	c := regexp.MustCompile(`(\s)(renew|rebind|expire) [^;]*;`)
	return c.ReplaceAllString(lease, "${1}${2} never;")
}

func addTcpdumpStub(hostID, ifaceName string, timeoutSecs, count int) {
	countStr := ""
	if count > 0 {
		countStr = fmt.Sprintf("-c %d", count)
	}
	_, err := addNextStepStub(hostID, 5,
		createCustomStub(TcpdumpStepType, "bash", "-c",
			fmt.Sprintf("timeout %d tcpdump -l -i %s %s -v 'udp dst port 67' | grep 'DHCP-Message Option 53' | awk '{print $6}'", timeoutSecs, ifaceName, countStr)),
		&models.Step{
			StepType: models.StepTypeExecute,
			Command:  "bash",
			Args: []string{
				"-c",
				"sleep 1; echo tcpdump started",
			},
		},
	)
	Expect(err).ToNot(HaveOccurred())
}

func waitforTcpdumpToStart(hostID string) {
	EventuallyWithOffset(1, func() bool {
		return isReplyFound(hostID, &EqualReplyVerifier{
			Output:   "tcpdump started\n",
			StepType: models.StepTypeExecute,
		})
	}, 30*time.Second, 500*time.Millisecond).Should(BeTrue())
}

type TcpdumVerifier struct{}

func (*TcpdumVerifier) verify(actualReply *models.StepReply) bool {
	return actualReply.StepType == TcpdumpStepType
}

func getTcpdumpReponse(hostID string) string {
	EventuallyWithOffset(1, func() bool {
		return isReplyFound(hostID, &TcpdumVerifier{})
	}, 30*time.Second, 5*time.Second).Should(BeTrue())

	stepReply := getSpecificStep(hostID, &TcpdumVerifier{})
	return stepReply.Output
}

func countSubstringOccurrences(s, substr string) int {
	ret := 0
	for index := 0; ; {
		i := strings.Index(s[index:], substr)
		switch i {
		case -1:
			return ret
		default:
			ret++
			index = index + i + len(substr) + 1
		}
		// For safe side
		if ret >= 1000 {
			return ret
		}
	}
}

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
	EventuallyWithOffset(1, func() bool {
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
