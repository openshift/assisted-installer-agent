package free_addresses

import (
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/go-openapi/strfmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("Hostname test", func() {
	var execute *MockExecuter
	var log logrus.FieldLogger

	BeforeEach(func() {
		execute = &MockExecuter{}
		log = logrus.New()
	})

	AfterEach(func() {
		execute.AssertExpectations(GinkgoT())
	})

	var freeRequest = func(args ...string) string {
		ret := models.FreeAddressesRequest(args)
		b, err := json.Marshal(&ret)
		Expect(err).NotTo(HaveOccurred())
		return string(b)
	}

	var oneAddress = `
<nmaprun>
<host>
<status state="up"/>
<address addr="10.0.0.254" addrtype="ipv4"/>
</host>
<host>
<status state="down"/>
<address addr="10.0.0.250" addrtype="ipv4"/>
</host>
</nmaprun>
`

	var mutipleAddresses1 = `
<nmaprun>
<host>
<status state="up"/>
<address addr="192.168.0.1" addrtype="ipv4"/>
</host>
<host>
<status state="down"/>
<address addr="192.168.0.2" addrtype="ipv4"/>
</host>
<host>
<status state="up"/>
<address addr="192.168.0.10" addrtype="ipv4"/>
</host>
</nmaprun>

`

	var mutipleAddresses2 = `
<nmaprun>
<host>
<status state="up"/>
<address addr="192.168.4.1" addrtype="ipv4"/>
</host>
<host>
<status state="down"/>
<address addr="192.168.4.2" addrtype="ipv4"/>
</host>
<host>
<status state="up"/>
<address addr="192.168.4.11" addrtype="ipv4"/>
</host>
</nmaprun>

`

	var mutipleAddresses3 = `
<nmaprun>
<host>
<status state="up"/>
<address addr="192.168.8.1" addrtype="ipv4"/>
</host>
<host>
<status state="down"/>
<address addr="192.168.8.2" addrtype="ipv4"/>
</host>
<host>
<status state="up"/>
<address addr="192.168.8.12" addrtype="ipv4"/>
</host>
</nmaprun>
`

	var mutipleAddresses4 = `
<nmaprun>
<host>
<status state="up"/>
<address addr="192.168.12.1" addrtype="ipv4"/>
</host>
<host>
<status state="down"/>
<address addr="192.168.12.2" addrtype="ipv4"/>
</host>
<host>
<status state="up"/>
<address addr="192.168.12.13" addrtype="ipv4"/>
</host>
</nmaprun>
`
	var empty = "<nmaprun/>"

	var parseResponse = func(response string) *models.FreeNetworksAddresses {
		ret := models.FreeNetworksAddresses{}
		Expect(json.Unmarshal([]byte(response), &ret)).NotTo(HaveOccurred())
		return &ret
	}

	It("Parse Error", func() {
		o, e, exitCode := GetFreeAddresses("blah blah", execute, log, false)
		Expect(exitCode).To(Equal(-1))
		Expect(o).To(BeEmpty())
		Expect(e).To(Equal("invalid character 'b' looking for beginning of value"))
	})
	It("Bad network", func() {
		o, e, exitCode := GetFreeAddresses(freeRequest("10.0.0.1/24"), execute, log, false)
		Expect(exitCode).To(Equal(-1))
		Expect(o).To(BeEmpty())
		Expect(e).To(Equal("Requested CIDR 10.0.0.0/24 is not equal to provided network 10.0.0.1/24"))
	})
	Context("Full cycle", func() {
		It("Happpy flow", func() {
			execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "10.0.0.0/24").
				Return(oneAddress, "", 0)
			o, e, exitCode := GetFreeAddresses(freeRequest("10.0.0.0/24"), execute, log, false)
			Expect(exitCode).To(Equal(0))
			Expect(e).To(BeEmpty())
			parsedResponse := *parseResponse(o)
			Expect(parsedResponse).To(HaveLen(1))
			Expect(parsedResponse[0].Network).To(Equal("10.0.0.0/24"))
			Expect(parsedResponse[0].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("10.0.0.254")))
			Expect(parsedResponse[0].FreeAddresses).To(ContainElement(strfmt.IPv4("10.0.0.250")))
		})
	})
	It("Multiple CIDRs", func() {
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "10.0.0.0/24").
			Return(oneAddress, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.0.0/22").
			Return(mutipleAddresses1, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.4.0/22").
			Return(mutipleAddresses2, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.8.0/22").
			Return(mutipleAddresses3, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.12.0/22").
			Return(mutipleAddresses4, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.16.0/22").
			Return(empty, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.20.0/22").
			Return(empty, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.24.0/22").
			Return(empty, "", 0)
		execute.On("Execute", "nmap", "-sn", "-PR", "-n", "-oX", "-", "192.168.28.0/22").
			Return(empty, "", 0)
		o, e, exitCode := GetFreeAddresses(freeRequest("10.0.0.0/24", "192.168.0.0/18"), execute, log, false)
		Expect(exitCode).To(Equal(0))
		Expect(e).To(BeEmpty())
		parsedResponse := *parseResponse(o)
		Expect(parsedResponse).To(HaveLen(2))
		Expect(parsedResponse[0].Network).To(Equal("10.0.0.0/24"))
		Expect(parsedResponse[0].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("10.0.0.254")))
		Expect(parsedResponse[0].FreeAddresses).To(ContainElement(strfmt.IPv4("10.0.0.250")))

		Expect(parsedResponse[1].Network).To(Equal("192.168.0.0/18"))
		Expect(len(parsedResponse[1].FreeAddresses)).To(BeNumerically(">=", AddressLimit))
		Expect(len(parsedResponse[1].FreeAddresses)).To(BeNumerically("<=", AddressLimit-1+(1<<(32-MinSubnetMaskSize))))
		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.0.1")))
		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.0.10")))
		Expect(parsedResponse[1].FreeAddresses).To(ContainElement(strfmt.IPv4("192.168.3.3")))

		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.4.1")))
		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.4.11")))
		Expect(parsedResponse[1].FreeAddresses).To(ContainElement(strfmt.IPv4("192.168.6.3")))

		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.8.1")))
		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.8.12")))
		Expect(parsedResponse[1].FreeAddresses).To(ContainElement(strfmt.IPv4("192.168.9.3")))

		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.12.1")))
		Expect(parsedResponse[1].FreeAddresses).ToNot(ContainElement(strfmt.IPv4("192.168.12.13")))

		Expect(parsedResponse[1].FreeAddresses).To(ContainElement(strfmt.IPv4("192.168.12.12")))
		Expect(parsedResponse[1].FreeAddresses).To(ContainElement(strfmt.IPv4("192.168.12.11")))
		Expect(parsedResponse[1].FreeAddresses).To(ContainElement(strfmt.IPv4("192.168.12.10")))

		Expect(parsedResponse[1].FreeAddresses).To(ContainElement(strfmt.IPv4("192.168.0.3")))
	})
})

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Free addresses unit tests")
}
