package util

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

var _ = Describe("Test IPv6 address prefix", func() {

	var routeFinder *MockRouteFinder
	var link *MockLink
	var log = logrus.StandardLogger()

	BeforeEach(func() {
		routeFinder = &MockRouteFinder{}
		link = &MockLink{}
	})

	It("Empty addresses", func() {
		addrs := []string{}
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("No link", func() {
		addrs := []string{"fe80::d832:8def:dd51:3527/128"}
		ret := fmt.Errorf("link not found")
		routeFinder.On("LinkByName", "abc").Return(nil, ret)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).Should(Equal(ret))
	})

	It("Failed to get routes", func() {
		addrs := []string{"fe80::d832:8def:dd51:3527/128"}
		routeFinder.On("LinkByName", "abc").Return(link, nil)
		ret := fmt.Errorf("failed to list routes")
		routeFinder.On("RouteList", link, netlink.FAMILY_V6).Return(nil, ret)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).Should(Equal(ret))
	})

	It("Empty routes", func() {
		addrs := []string{"fe80::d832:8def:dd51:3527/128"}
		routeFinder.On("LinkByName", "abc").Return(link, nil)
		routeFinder.On("RouteList", link, netlink.FAMILY_V6).Return([]netlink.Route{}, nil)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(addrs[0]).Should(Equal("fe80::d832:8def:dd51:3527/128"))
	})

	It("Matching route", func() {
		addrs := []string{"fe80::d832:8def:dd51:3527/128"}
		routeFinder.On("LinkByName", "abc").Return(link, nil)
		routes := []netlink.Route{
			{
				Dst:      &net.IPNet{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(64, 128)},
				Protocol: unix.RTPROT_RA,
			},
		}
		routeFinder.On("RouteList", link, netlink.FAMILY_V6).Return(routes, nil)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(addrs[0]).Should(Equal("fe80::d832:8def:dd51:3527/64"))
	})

	It("No matching routes", func() {
		addrs := []string{"fe80::d832:8def:dd51:3527/128"}
		routeFinder.On("LinkByName", "abc").Return(link, nil)
		routes := []netlink.Route{
			{
				Dst:      &net.IPNet{IP: net.ParseIP("1001:db8:"), Mask: net.CIDRMask(64, 128)},
				Protocol: unix.RTPROT_RA,
			},
		}
		routeFinder.On("RouteList", link, netlink.FAMILY_V6).Return(routes, nil)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(addrs[0]).Should(Equal("fe80::d832:8def:dd51:3527/128"))
	})

	It("Not usable route dest nil", func() {
		addrs := []string{"fe80::d832:8def:dd51:3527/128"}
		routeFinder.On("LinkByName", "abc").Return(link, nil)
		routes := []netlink.Route{
			{
				Protocol: unix.RTPROT_RA,
			},
		}
		routeFinder.On("RouteList", link, netlink.FAMILY_V6).Return(routes, nil)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(addrs[0]).Should(Equal("fe80::d832:8def:dd51:3527/128"))
	})

	It("Not usable route non-advertised", func() {
		addrs := []string{"fe80::d832:8def:dd51:3527/128"}
		routeFinder.On("LinkByName", "abc").Return(link, nil)
		routes := []netlink.Route{
			{
				Dst: &net.IPNet{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(64, 128)},
			},
		}
		routeFinder.On("RouteList", link, netlink.FAMILY_V6).Return(routes, nil)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(addrs[0]).Should(Equal("fe80::d832:8def:dd51:3527/128"))
	})

	It("IPv4 address skipped", func() {
		addrs := []string{"10.56.20.70/32"}
		routeFinder.On("LinkByName", "abc").Return(link, nil)
		routes := []netlink.Route{
			{
				Dst:      &net.IPNet{IP: net.ParseIP("10.56.20.0"), Mask: net.CIDRMask(24, 32)},
				Protocol: unix.RTPROT_RA,
			},
		}
		routeFinder.On("RouteList", link, netlink.FAMILY_V6).Return(routes, nil)
		err := SetV6PrefixesForAddress("abc", routeFinder, log, addrs)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(addrs[0]).Should(Equal("10.56.20.70/32"))
	})

	AfterEach(func() {
		routeFinder.AssertExpectations(GinkgoT())
		link.AssertExpectations(GinkgoT())
	})
})
