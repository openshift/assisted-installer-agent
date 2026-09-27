package vips_verifier

import (
	"encoding/json"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("vips_verifier", func() {
	var (
		verifier     VipVerifier
		mockExecuter *MockExecuter
	)
	ipv4Requests := models.VerifyVipsRequest{
		{
			Vip:     "1.2.3.4",
			VipType: "api",
		},
		{
			Vip:     "1.2.3.5",
			VipType: "ingress",
		},
	}
	ipv6Requests := models.VerifyVipsRequest{
		{
			Vip:     "ff::1",
			VipType: "api",
		},
		{
			Vip:     "ff::2",
			VipType: "ingress",
		},
	}
	on := func(success bool) {
		format := `
<nmaprun>
<host>
<status state="%s"/>
<address addr="%s" addrtype="%s"/>
</host>
</nmaprun>
`
		state := "up"
		if success {
			state = "down"
		}
		for _, req := range ipv4Requests {
			mockExecuter.On("Execute", "nmap", "-sn", "-n", "-oX", "-", string(req.Vip)).
				Return(fmt.Sprintf(format, state, req.Vip, "ipv4"), "", 0).Once()
		}
		for _, req := range ipv6Requests {
			mockExecuter.On("Execute", "nmap", "-6", "-sn", "-n", "-oX", "-", string(req.Vip)).
				Return(fmt.Sprintf(format, state, req.Vip, "ipv6"), "", 0).Once()
		}
	}
	request := append(ipv4Requests, ipv6Requests...)
	BeforeEach(func() {
		mockExecuter = &MockExecuter{}
		verifier = &vipVerifier{exe: mockExecuter}
	})
	AfterEach(func() {
		mockExecuter.AssertExpectations(GinkgoT())
	})

	marshalRequest := func(request models.VerifyVipsRequest) string {
		b, err := json.Marshal(&request)
		Expect(err).ToNot(HaveOccurred())
		return string(b)
	}

	unmarshalResponse := func(response string) models.VerifyVipsResponse {
		var ret models.VerifyVipsResponse
		Expect(json.Unmarshal([]byte(response), &ret)).ToNot(HaveOccurred())
		return ret
	}

	expect := func(response models.VerifyVipsResponse, verification models.VipVerification) {
		Expect(len(response)).To(Equal(len(request)))
		for i := range request {
			Expect(request[i].VipType).To(Equal(response[i].VipType))
			Expect(request[i].Vip).To(Equal(response[i].Vip))
			Expect(response[i].Verification).ToNot(BeNil())
			Expect(*response[i].Verification).To(Equal(verification))
		}
	}

	verify := func(arg string) (string, string, int) {
		return verifyVips(verifier, arg)
	}
	It("bad request", func() {
		_, _, exitCode := verify("abc")
		Expect(exitCode).ToNot(BeZero())
	})
	It("vips not verified", func() {
		on(false)
		o, _, exitCode := verify(marshalRequest(request))
		Expect(exitCode).To(BeZero())
		expect(unmarshalResponse(o), models.VipVerificationFailed)
	})
	It("vips verified", func() {
		on(true)
		o, _, exitCode := verify(marshalRequest(request))
		Expect(exitCode).To(BeZero())
		expect(unmarshalResponse(o), models.VipVerificationSucceeded)
	})
})

func TestVerifyVips(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vips verifier unit tests")
}
