package connectivity_check

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("ipv4 arping checker", func() {
	const (
		remoteIPAddress      = "1.2.3.4"
		outgoingIPAddress    = "1.2.3.5"
		remoteMACAddress     = "80:32:53:4f:cf:d6"
		outgoingNIC          = "eth0"
		secondaryMACAddress  = "80:32:53:4f:cf:d7"
		additionalMACAddress = "80:32:53:4f:cf:d8"

		fullReplyFormat = `ARPING %[1]s from %[2]s enp0s31f6
Unicast reply from %[1]s [%[3]s]  1.507ms
Unicast reply from %[1]s [%[3]s]  1.425ms
Unicast reply from %[1]s [%[3]s]  1.304ms
Unicast reply from %[1]s [%[3]s]  1.343ms
Sent 4 probes (1 broadcast(s))
Received 4 response(s)
`
		emptyReplyFormat = `ARPING %[1]s from %[2]s enp0s31f6
Sent 4 probes (1 broadcast(s))
Received 0 response(s)
`
	)
	var (
		checker            Checker
		mockExecuter       *MockExecuter
		remoteMACAddresses = []string{remoteMACAddress, secondaryMACAddress}
	)
	mockFullReply := func(remoteIPAddress, remoteMACAddress string) {
		mockExecuter.On("Execute", "arping", "-c", "10", "-w", "5", "-I", outgoingNIC, remoteIPAddress).
			Return(fmt.Sprintf(fullReplyFormat, remoteIPAddress, outgoingIPAddress, remoteMACAddress), nil).Once()
	}
	mockEmptyReply := func(remoteIPAddress string) {
		mockExecuter.On("Execute", "arping", "-c", "10", "-w", "5", "-I", outgoingNIC, remoteIPAddress).
			Return(fmt.Sprintf(emptyReplyFormat, remoteIPAddress, outgoingIPAddress), nil).Once()
	}

	BeforeEach(func() {
		mockExecuter = &MockExecuter{}
		checker = &arpingChecker{executer: mockExecuter}
	})
	AfterEach(func() {
		mockExecuter.AssertExpectations(GinkgoT())
	})
	It("happy flow", func() {
		attributes := Attributes{
			RemoteIPAddress:    remoteIPAddress,
			RemoteMACAddress:   remoteMACAddress,
			OutgoingNIC:        OutgoingNic{Name: outgoingNIC, HasIpv4Addresses: true, Addresses: getAddr(outgoingIPAddress, 24)},
			RemoteMACAddresses: remoteMACAddresses,
		}
		mockFullReply(remoteIPAddress, remoteMACAddress)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.L2Connectivity).To(HaveLen(1))
		Expect(resultingHost.L2Connectivity[0].RemoteMac).To(Equal(remoteMACAddress))
		Expect(resultingHost.L2Connectivity[0].RemoteIPAddress).To(Equal(remoteIPAddress))
		Expect(resultingHost.L2Connectivity[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.L2Connectivity[0].OutgoingIPAddress).To(Equal(outgoingIPAddress))
		Expect(resultingHost.L2Connectivity[0].Successful).To(BeTrue())
	})
	It("no ipv4 addresses", func() {
		attributes := Attributes{
			RemoteIPAddress:    remoteIPAddress,
			RemoteMACAddress:   remoteMACAddress,
			OutgoingNIC:        OutgoingNic{Name: outgoingNIC, HasIpv6Addresses: true},
			RemoteMACAddresses: remoteMACAddresses,
		}
		reporter := checker.Check(attributes)
		Expect(reporter).To(BeNil())
	})
	It("happy flow with secondary mac", func() {
		attributes := Attributes{
			RemoteIPAddress:    remoteIPAddress,
			RemoteMACAddress:   remoteMACAddress,
			OutgoingNIC:        OutgoingNic{Name: outgoingNIC, HasIpv4Addresses: true, Addresses: getAddr(outgoingIPAddress, 24)},
			RemoteMACAddresses: remoteMACAddresses,
		}
		mockFullReply(remoteIPAddress, secondaryMACAddress)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.L2Connectivity).To(HaveLen(1))
		Expect(resultingHost.L2Connectivity[0].RemoteMac).To(Equal(secondaryMACAddress))
		Expect(resultingHost.L2Connectivity[0].RemoteIPAddress).To(Equal(remoteIPAddress))
		Expect(resultingHost.L2Connectivity[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.L2Connectivity[0].OutgoingIPAddress).To(Equal(outgoingIPAddress))
		Expect(resultingHost.L2Connectivity[0].Successful).To(BeTrue())
	})
	It("unexpected mac", func() {
		attributes := Attributes{
			RemoteIPAddress:    remoteIPAddress,
			RemoteMACAddress:   remoteMACAddress,
			OutgoingNIC:        OutgoingNic{Name: outgoingNIC, HasIpv4Addresses: true, Addresses: getAddr(outgoingIPAddress, 24)},
			RemoteMACAddresses: remoteMACAddresses,
		}
		mockFullReply(remoteIPAddress, additionalMACAddress)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.L2Connectivity).To(HaveLen(1))
		Expect(resultingHost.L2Connectivity[0].RemoteMac).To(Equal(additionalMACAddress))
		Expect(resultingHost.L2Connectivity[0].RemoteIPAddress).To(Equal(remoteIPAddress))
		Expect(resultingHost.L2Connectivity[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.L2Connectivity[0].OutgoingIPAddress).To(Equal(outgoingIPAddress))
		Expect(resultingHost.L2Connectivity[0].Successful).To(BeFalse())
	})
	It("no reply", func() {
		attributes := Attributes{
			RemoteIPAddress:    remoteIPAddress,
			RemoteMACAddress:   remoteMACAddress,
			OutgoingNIC:        OutgoingNic{Name: outgoingNIC, HasIpv4Addresses: true, Addresses: getAddr(outgoingIPAddress, 24)},
			RemoteMACAddresses: remoteMACAddresses,
		}
		mockEmptyReply(remoteIPAddress)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.L2Connectivity).To(HaveLen(0))
	})
	It("outgoing interface in different subnet", func() {
		attributes := Attributes{
			RemoteIPAddress:    remoteIPAddress,
			RemoteMACAddress:   remoteMACAddress,
			OutgoingNIC:        OutgoingNic{Name: outgoingNIC, HasIpv4Addresses: true, Addresses: getAddr("2.2.3.5", 24)},
			RemoteMACAddresses: remoteMACAddresses,
		}
		reporter := checker.Check(attributes)
		Expect(reporter).To(BeNil())
	})
})
