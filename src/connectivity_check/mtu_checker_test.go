package connectivity_check

import (
	"fmt"
	"net"
	"strconv"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("MTU checker", func() {
	const (
		remoteIPv4Address   = "192.168.127.31"
		remoteIPv6Address   = "3001:db9::22"
		outgoingIPv4Address = "192.168.127.30"
		outgoingIPv6Address = "3001:db9::1f"
		outgoingNIC         = "ens3"
		bigSize             = 9000
		regularSize         = 1500
		smallSize           = 1300
	)

	var (
		checker      Checker
		mockExecuter *MockExecuter
	)

	mockSuccessPingWithoutSize := func(remoteIPAddress string) {
		mockExecuter.On("Execute", "ping", remoteIPAddress, "-c", "3", "-I", outgoingNIC).
			Return("success output", nil).Once()
	}

	mockFailPingWithSize := func(remoteIPAddress string, sizeWithoutIPHeader int) {
		mockExecuter.On("Execute", "ping", remoteIPAddress, "-c", "3", "-M", "do", "-s", strconv.Itoa(sizeWithoutIPHeader), "-I", outgoingNIC).
			Return("failure output", fmt.Errorf("some error")).Once()
	}

	mockSuccessPingWithSize := func(remoteIPAddress string, sizeWithoutIPHeader int) {
		mockExecuter.On("Execute", "ping", remoteIPAddress, "-c", "3", "-M", "do", "-s", strconv.Itoa(sizeWithoutIPHeader), "-I", outgoingNIC).
			Return("success output", nil).Once()
	}

	ipv4Mask := net.CIDRMask(24, 32)
	ipv6Mask := net.CIDRMask(64, 128)

	BeforeEach(func() {
		// for a test. will be removed
		mockExecuter = &MockExecuter{}
		checker = &mtuChecker{executer: mockExecuter}
	})
	AfterEach(func() {
		mockExecuter.AssertExpectations(GinkgoT())
	})
	table.DescribeTable("Successful report", func(remoteIP, outgoingIP string, size, header int, mask net.IPMask) {
		attributes := Attributes{
			RemoteIPAddress: remoteIP,
			OutgoingNIC: OutgoingNic{Name: outgoingNIC, MTU: size, Addresses: []net.Addr{&net.IPNet{
				IP:   net.ParseIP(outgoingIP),
				Mask: mask,
			}}},
		}
		mockSuccessPingWithoutSize(remoteIP)
		mockSuccessPingWithSize(remoteIP, size-header)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.MtuReport).To(HaveLen(1))
		Expect(resultingHost.MtuReport[0].RemoteIPAddress).To(Equal(remoteIP))
		Expect(resultingHost.MtuReport[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.MtuReport[0].MtuSuccessful).To(BeTrue())
	},
		table.Entry("MTU > 1500, IPv4 ", remoteIPv4Address, outgoingIPv4Address, bigSize, ipv4Header, ipv4Mask),
		table.Entry("MTU > 1500, IPv6 ", remoteIPv6Address, outgoingIPv6Address, bigSize, ipv6Header, ipv6Mask),
		table.Entry("MTU < 1500, IPv4 ", remoteIPv4Address, outgoingIPv4Address, smallSize, ipv4Header, ipv4Mask),
		table.Entry("MTU < 1500, IPv6 ", remoteIPv6Address, outgoingIPv6Address, smallSize, ipv6Header, ipv6Mask),
	)
	table.DescribeTable("Unsuccessful report", func(remoteIP, outgoingIP string, size, header int, mask net.IPMask) {
		attributes := Attributes{
			RemoteIPAddress: remoteIP,
			OutgoingNIC: OutgoingNic{Name: outgoingNIC, MTU: size, Addresses: []net.Addr{&net.IPNet{
				IP:   net.ParseIP(outgoingIP),
				Mask: mask,
			}}},
		}
		mockSuccessPingWithoutSize(remoteIP)
		mockFailPingWithSize(remoteIP, size-header)
		reporter := checker.Check(attributes)
		Expect(reporter).ToNot(BeNil())
		var resultingHost models.ConnectivityRemoteHost
		Expect(reporter.Report(&resultingHost)).ToNot(HaveOccurred())
		Expect(resultingHost.MtuReport).To(HaveLen(1))
		Expect(resultingHost.MtuReport[0].RemoteIPAddress).To(Equal(remoteIP))
		Expect(resultingHost.MtuReport[0].OutgoingNic).To(Equal(outgoingNIC))
		Expect(resultingHost.MtuReport[0].MtuSuccessful).To(BeFalse())
	},
		table.Entry("MTU > 1500, IPv4 ", remoteIPv4Address, outgoingIPv4Address, bigSize, ipv4Header, ipv4Mask),
		table.Entry("MTU > 1500, IPv6 ", remoteIPv6Address, outgoingIPv6Address, bigSize, ipv6Header, ipv6Mask),
		table.Entry("MTU < 1500, IPv4 ", remoteIPv4Address, outgoingIPv4Address, smallSize, ipv4Header, ipv4Mask),
		table.Entry("MTU < 1500, IPv6 ", remoteIPv6Address, outgoingIPv6Address, smallSize, ipv6Header, ipv6Mask),
	)
	table.DescribeTable("MTU equal 1500 - should not check", func(remoteIP, outgoingIP string, mask net.IPMask) {
		attributes := Attributes{
			RemoteIPAddress: remoteIP,
			OutgoingNIC: OutgoingNic{Name: outgoingNIC, MTU: regularSize, Addresses: []net.Addr{&net.IPNet{
				IP:   net.ParseIP(outgoingIP),
				Mask: mask,
			}}},
		}
		reporter := checker.Check(attributes)
		Expect(reporter).To(BeNil())
	},
		table.Entry("IPv4", remoteIPv4Address, outgoingIPv4Address, ipv4Mask),
		table.Entry("IPv6", remoteIPv6Address, outgoingIPv6Address, ipv6Mask),
	)
})
