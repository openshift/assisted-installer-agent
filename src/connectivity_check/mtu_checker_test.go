package connectivity_check

import (
	"fmt"
	"net"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("MTU checker", func() {
	const (
		remoteIPv4Address         = "192.168.127.31"
		remoteIPv6Address         = "3001:db9::22"
		outgoingIPv4Address       = "192.168.127.30"
		outgoingIPv6Address       = "3001:db9::1f"
		outgoingNIC               = "ens3"
		bigSize                   = 9000
		regularSize               = 1500
		bigSizeWithoutHeaders     = bigSize - headers
		regularSizeWithoutHeaders = regularSize - headers
	)

	var (
		checker      Checker
		mockExecuter *MockExecuter
	)

	mockSuccessPingWithoutSize := func(remoteIPAddress string) {
		mockExecuter.On("Execute", "ping", remoteIPAddress, "-c", "3", "-I", outgoingNIC).
			Return("success output", nil).Once()
	}

	mockSuccessPingWithSize := func(remoteIPAddress string) {
		mockExecuter.On("Execute", "ping", remoteIPAddress, "-c", "3", "-M", "do", "-s", strconv.Itoa(regularSizeWithoutHeaders), "-I", outgoingNIC).
			Return("success output", nil).Once()
	}

	mockFailPingWithSize := func(remoteIPAddress string) {
		mockExecuter.On("Execute", "ping", remoteIPAddress, "-c", "3", "-M", "do", "-s", strconv.Itoa(bigSizeWithoutHeaders), "-I", outgoingNIC).
			Return("failure output", fmt.Errorf("some error")).Once()
	}

	BeforeEach(func() {
		mockExecuter = &MockExecuter{}
		checker = &mtuChecker{executer: mockExecuter}
	})
	AfterEach(func() {
		mockExecuter.AssertExpectations(GinkgoT())
	})
	It("MTU Check Failure - IPv4", func() {

		attributes := Attributes{
			RemoteIPAddress: remoteIPv4Address,
			OutgoingNIC: OutgoingNic{Name: outgoingNIC, MTU: 9000, Addresses: []net.Addr{&net.IPNet{
				IP:   net.ParseIP(outgoingIPv4Address),
				Mask: net.CIDRMask(24, 32),
			}}},
		}
		mockSuccessPingWithoutSize(remoteIPv4Address)
		mockFailPingWithSize(remoteIPv4Address)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.MtuReport).To(HaveLen(1))
		Expect(resultingHost.MtuReport[0].RemoteIPAddress).To(Equal(remoteIPv4Address))
		Expect(resultingHost.MtuReport[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.MtuReport[0].MtuSuccessful).To(BeFalse())
	})
	It("MTU Check Failure - IPv6", func() {

		attributes := Attributes{
			RemoteIPAddress: remoteIPv6Address,
			OutgoingNIC: OutgoingNic{Name: outgoingNIC, MTU: 9000, Addresses: []net.Addr{&net.IPNet{
				IP:   net.ParseIP(outgoingIPv6Address),
				Mask: net.CIDRMask(64, 128),
			}}},
		}
		mockSuccessPingWithoutSize(remoteIPv6Address)
		mockFailPingWithSize(remoteIPv6Address)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.MtuReport).To(HaveLen(1))
		Expect(resultingHost.MtuReport[0].RemoteIPAddress).To(Equal(remoteIPv6Address))
		Expect(resultingHost.MtuReport[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.MtuReport[0].MtuSuccessful).To(BeFalse())
	})
	It("MTU Check Success - IPv4", func() {

		attributes := Attributes{
			RemoteIPAddress: remoteIPv4Address,
			OutgoingNIC: OutgoingNic{Name: outgoingNIC, MTU: 1500, Addresses: []net.Addr{&net.IPNet{
				IP:   net.ParseIP(outgoingIPv4Address),
				Mask: net.CIDRMask(24, 32),
			}}},
		}
		mockSuccessPingWithoutSize(remoteIPv4Address)
		mockSuccessPingWithSize(remoteIPv4Address)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.MtuReport).To(HaveLen(1))
		Expect(resultingHost.MtuReport[0].RemoteIPAddress).To(Equal(remoteIPv4Address))
		Expect(resultingHost.MtuReport[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.MtuReport[0].MtuSuccessful).To(BeTrue())
	})
	It("MTU Check Success - IPv6", func() {

		attributes := Attributes{
			RemoteIPAddress: remoteIPv6Address,
			OutgoingNIC: OutgoingNic{Name: outgoingNIC, MTU: 1500, Addresses: []net.Addr{&net.IPNet{
				IP:   net.ParseIP(outgoingIPv6Address),
				Mask: net.CIDRMask(64, 128),
			}}},
		}
		mockSuccessPingWithoutSize(remoteIPv6Address)
		mockSuccessPingWithSize(remoteIPv6Address)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.MtuReport).To(HaveLen(1))
		Expect(resultingHost.MtuReport[0].RemoteIPAddress).To(Equal(remoteIPv6Address))
		Expect(resultingHost.MtuReport[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.MtuReport[0].MtuSuccessful).To(BeTrue())
	})
})
