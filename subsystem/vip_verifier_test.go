package subsystem

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("vip verifier tests", func() {
	var (
		allocatedIps        map[string]*containerNodeIps
		allocatedIPv4Subnet string
		allocatedIPv6Subnet string
		hostID              string
	)
	BeforeEach(func() {
		resetAll()
		hostID = nextHostID()
		_, _ = addRegisterStub(hostID, http.StatusCreated, InfraEnvID)
		setReplyStartAgent(hostID)
		allocatedIps = getAgentNetworkIps()
		Expect(allocatedIps).ToNot(BeEmpty())
		for _, ips := range allocatedIps {
			_, ipnet, err := net.ParseCIDR(ips.ipv4Address)
			Expect(err).ToNot(HaveOccurred())
			allocatedIPv4Subnet = ipnet.String()
			_, ipnet, err = net.ParseCIDR(ips.ipv6Address)
			Expect(err).ToNot(HaveOccurred())
			allocatedIPv6Subnet = ipnet.String()
			break
		}
	})

	getCidrByFamily := func(contIps *containerNodeIps, isIpv4 bool) string {
		if isIpv4 {
			return contIps.ipv4Address
		}
		return contIps.ipv6Address
	}
	getIPByName := func(name string, isIpv4 bool) net.IP {
		ipCidrs, ok := allocatedIps[name]
		Expect(ok).To(BeTrue())
		ip, _, err := net.ParseCIDR(getCidrByFamily(ipCidrs, isIpv4))
		Expect(err).ToNot(HaveOccurred())
		return ip

	}
	getWiremockIP := func(isIpv4 bool) net.IP {
		return getIPByName("wiremock", isIpv4)
	}

	getAgentIP := func(isIpv4 bool) net.IP {
		return getIPByName("agent", isIpv4)
	}

	incIP := func(ip net.IP) net.IP {
		var ret net.IP
		ret = append(ret, ip...)
		for j := len(ret) - 1; j >= 0; j-- {
			ret[j]++
			if ret[j] > 0 {
				break
			}
		}
		return ret
	}

	getFreeIP := func(start net.IP) net.IP {
		allocatedSet := make(map[string]bool)
		for _, value := range allocatedIps {
			ip, _, err := net.ParseCIDR(getCidrByFamily(value, util.IsIPv4Addr(start.String())))
			Expect(err).ToNot(HaveOccurred())
			allocatedSet[ip.String()] = true
		}
		for candidate := start; ; candidate = incIP(candidate) {
			_, exists := allocatedSet[candidate.String()]
			if !exists {
				return candidate
			}
		}
	}
	firstIP := func(isIpv4 bool) net.IP {
		var (
			ip  net.IP
			err error
		)
		if isIpv4 {
			ip, _, err = net.ParseCIDR(allocatedIPv4Subnet)
		} else {
			ip, _, err = net.ParseCIDR(allocatedIPv6Subnet)
		}
		Expect(err).ToNot(HaveOccurred())
		return incIP(incIP(ip))
	}
	marshalRequest := func(request models.VerifyVipsRequest) string {
		b, err := json.Marshal(&request)
		Expect(err).ToNot(HaveOccurred())
		return string(b)
	}

	getVipVerificationResponse := func(hostID string) models.VerifyVipsResponse {
		Eventually(func() bool {
			return isReplyFound(hostID, &vipResponseVerifier{})
		}).WithTimeout(maxTimeout).WithPolling(5 * time.Second).Should(BeTrue())
		stepReply := getSpecificStep(hostID, &vipResponseVerifier{})
		var ret models.VerifyVipsResponse
		Expect(json.Unmarshal([]byte(stepReply.Output), &ret)).ToNot(HaveOccurred())
		return ret
	}
	expect := func(request models.VerifyVipsRequest, response models.VerifyVipsResponse, verifications []models.VipVerification) {
		Expect(len(response)).To(Equal(len(request)))
		Expect(len(response)).To(Equal(len(verifications)))
		for i := range request {
			Expect(request[i].VipType).To(Equal(response[i].VipType))
			Expect(request[i].Vip).To(Equal(response[i].Vip))
			Expect(response[i].Verification).ToNot(BeNil())
			Expect(*response[i].Verification).To(Equal(verifications[i]))
		}
	}

	runTest := func(apiIP, ingressIP net.IP, expectedVerifications []models.VipVerification) {
		request := models.VerifyVipsRequest{
			{
				Vip:     models.IP(apiIP.String()),
				VipType: models.VipTypeAPI,
			},
			{
				Vip:     models.IP(ingressIP.String()),
				VipType: models.VipTypeIngress,
			},
		}

		step := generateStep(models.StepTypeVerifyVips, []string{marshalRequest(request)})
		_, err := addNextStepStub(hostID, 5, "", step)
		Expect(err).NotTo(HaveOccurred())
		setReplyStartAgent(hostID)
		response := getVipVerificationResponse(hostID)
		expect(request, response, expectedVerifications)
	}
	for _, b := range []bool{true, false} {
		isIpv4 := b
		stack := "ipv6"
		if isIpv4 {
			stack = "ipv4"
		}
		It("happy flow "+stack, func() {
			apiIP := getFreeIP(firstIP(isIpv4))
			ingressIP := getFreeIP(incIP(apiIP))
			runTest(apiIP, ingressIP, []models.VipVerification{models.VipVerificationSucceeded, models.VipVerificationSucceeded})
		})

		It("failed flow "+stack, func() {
			apiIP := getWiremockIP(isIpv4)
			ingressIP := getAgentIP(isIpv4)
			runTest(apiIP, ingressIP, []models.VipVerification{models.VipVerificationFailed, models.VipVerificationFailed})
		})
		It("mixed flow "+stack, func() {
			apiIP := getFreeIP(firstIP(isIpv4))
			ingressIP := getAgentIP(isIpv4)
			runTest(apiIP, ingressIP, []models.VipVerification{models.VipVerificationSucceeded, models.VipVerificationFailed})
		})
	}
})

type vipResponseVerifier struct{}

func (v *vipResponseVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.StepType != models.StepTypeVerifyVips {
		return false
	}
	var verifyVipsResponse models.VerifyVipsResponse
	if err := json.Unmarshal([]byte(actualReply.Output), &verifyVipsResponse); err != nil {
		return false
	}
	return true
}
