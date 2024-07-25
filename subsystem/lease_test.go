package subsystem

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	agentUtils "github.com/openshift/assisted-installer-agent/src/util"

	"github.com/go-openapi/strfmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
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

			setReplyStartAgent(hostID)
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
			setReplyStartAgent(hostID)
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

			setReplyStartAgent(hostID)
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

			setReplyStartAgent(hostID)
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
				Expect(stopContainer("dhcpd")).ShouldNot(HaveOccurred())

				setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
					APIVipMac:     &apiMac,
					IngressVipMac: &ingressMac,
					Interface:     &ifaceName,
				})

				setReplyStartAgent(hostID)
				Eventually(isReplyFound(hostID, &DHCPLeaseAllocateVerifier{})).Should(BeFalse())
			})

			By("restart dhcpd", func() {
				Expect(startContainer("dhcpd")).ShouldNot(HaveOccurred())

				Eventually(func() bool {
					return isReplyFound(hostID, &DHCPLeaseAllocateVerifier{})
				}, maxTimeout, 5*time.Second).Should(BeTrue())
			})
		})

		It("invalid_mac", func() {
			apiMac = "invalid_mac"

			stepID := setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			setReplyStartAgent(hostID)
			Eventually(isReplyFound(hostID, &EqualReplyVerifier{
				Error:    fmt.Sprintf("validation failure list:\napi_vip_mac in body must be of type mac: \"%s\"", apiMac),
				ExitCode: -1,
				Output:   "",
				StepID:   stepID,
				StepType: models.StepTypeDhcpLeaseAllocate,
			})).Should(BeTrue())
		})

		It("invalid_interface", func() {
			ifaceName = "invalid-interface"

			stepID := setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
				APIVipMac:     &apiMac,
				IngressVipMac: &ingressMac,
				Interface:     &ifaceName,
			})

			setReplyStartAgent(hostID)
			Eventually(isReplyFound(hostID, &EqualReplyVerifier{
				Error:    "numerical result out of range",
				ExitCode: -1,
				Output:   "",
				StepID:   stepID,
				StepType: models.StepTypeDhcpLeaseAllocate,
			})).Should(BeTrue())
		})
	})
	Context("Changing VIPs", func() {
		It("DHCPREQUEST with provided lease", func() {
			var dhcpResponse *models.DhcpAllocationResponse
			resetAll()
			By("1st time", func() {
				_, _ = addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
				setReplyStartAgent(hostID)
				messages := make(chan string)
				//Expect(startAgent()).NotTo(HaveOccurred())
				time.Sleep(60 * time.Second)
				go startTcpDump(messages, ifaceName, 70, 6)
				waitforTcpdumpToStart()
				setDHCPLeaseRequestStub(hostID, models.DhcpAllocationRequest{
					APIVipMac:     &apiMac,
					IngressVipMac: &ingressMac,
					Interface:     &ifaceName,
				})
				dhcpResponse = getDHCPResponse(hostID)
				Expect(dhcpResponse).ShouldNot(BeNil())
				tcpdumpResponse := <-messages
				Expect(countSubstringOccurrences(tcpdumpResponse, "Discover")).To(BeNumerically(">=", 2))
				Expect(countSubstringOccurrences(tcpdumpResponse, "Request")).To(BeNumerically(">=", 2))
			})

			resetAll()

			By("2nd time", func() {

				messages := make(chan string)
				_, _ = addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
				setReplyStartAgent(hostID)
				time.Sleep(60 * time.Second)
				go startTcpDump(messages, ifaceName, 120, 4)
				waitforTcpdumpToStart()
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
				tcpdumpResponse := <-messages
				Expect(countSubstringOccurrences(tcpdumpResponse, "Discover")).To(Equal(0))
				Expect(countSubstringOccurrences(tcpdumpResponse, "Request")).To(BeNumerically(">=", 2))
			})
		})
	})
})

func formatLease(lease string) string {
	c := regexp.MustCompile(`(\s)(renew|rebind|expire) [^;]*;`)
	return c.ReplaceAllString(lease, "${1}${2} never;")
}

func startTcpDump(output chan string, ifaceName string, timeoutSecs, count int) {
	countStr := ""
	if count > 0 {
		countStr = fmt.Sprintf("-c %d", count)
	}
	dump := fmt.Sprintf("timeout %d tcpdump -l -i %s %s -v 'udp dst port 67' | awk '/DHCP-Message \\(53\\)/{print $5}'", timeoutSecs, ifaceName, countStr)
	s, e, errorCode := agentUtils.Execute("docker", []string{"exec", "agent", "bash", "-c", dump}...)
	fmt.Println(s, e, errorCode)
	output <- s
}

func waitforTcpdumpToStart() {
	EventuallyWithOffset(1, func() bool {
		_, _, exitCode := agentUtils.Execute("docker", []string{"exec", "agent", "bash", "-c", "ps aux | grep -v grep | grep tcpdump"}...)
		return exitCode == 0
	}, 30*time.Second, 500*time.Millisecond).Should(BeTrue())
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

func setDHCPLeaseRequestStub(hostID string, request models.DhcpAllocationRequest) string {
	_, err := addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
	Expect(err).NotTo(HaveOccurred())

	b, err := json.Marshal(&request)
	Expect(err).ShouldNot(HaveOccurred())

	step := generateStep(models.StepTypeDhcpLeaseAllocate,
		[]string{string(b)})
	_, err = addNextStepStub(hostID, 5, "", step)
	Expect(err).NotTo(HaveOccurred())

	return step.StepID
}

func getDHCPResponse(hostID string) *models.DhcpAllocationResponse {
	setPostReply(hostID)
	EventuallyWithOffset(1, func() bool {
		return isReplyFound(hostID, &DHCPLeaseAllocateVerifier{})
	}, 120*time.Second, 5*time.Second).Should(BeTrue())
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
		log.Errorf("DHCPLeaseAllocateVerifier invalid step reply %s", actualReply.StepType)
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
