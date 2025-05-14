package inventory

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type netPair struct {
	linkNames []string
	routes    []netlink.Route
}

var (
	ipV4GW = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Dst: nil, Gw: net.IPv4(10, 254, 0, 1), Priority: 100},
			{LinkIndex: 1, Dst: &net.IPNet{IP: net.IPv4(192, 168, 122, 0)}, Gw: net.IPv4zero, Priority: 100},
			{LinkIndex: 2, Dst: &net.IPNet{IP: net.IPv4(192, 168, 122, 0)}, Gw: nil, Priority: 101}},
		linkNames: []string{"eth3", "virbr0", "virbr1"}}

	ipv4NoInternetConnection = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Dst: &net.IPNet{IP: net.IPv4(10, 254, 0, 0)}, Gw: net.IPv4zero},
			{LinkIndex: 1, Dst: &net.IPNet{IP: net.IPv4(172, 17, 0, 0)}, Gw: net.IPv4zero}},
		linkNames: []string{"docker0", "virbr0"},
	}

	ipv4WithMultiPath = netPair{
		routes: []netlink.Route{
			{MultiPath: []*netlink.NexthopInfo{{LinkIndex: 0, Gw: net.IPv4(10, 254, 0, 1), Hops: 1},
				{LinkIndex: 1, Gw: net.IPv4(10, 10, 1, 1), Hops: 2}}}},
		linkNames: []string{"eth3", "virbr0"},
	}

	nothing = netPair{
		routes:    []netlink.Route{},
		linkNames: []string{},
	}

	ipV6GW = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Gw: net.ParseIP("2001:1::1"), Dst: &net.IPNet{IP: net.IPv6zero}, Priority: 101},
			{LinkIndex: 1, Gw: net.IPv6zero, Dst: &net.IPNet{IP: net.ParseIP("2001:2::1")}, Priority: 101},
			{LinkIndex: 2, Gw: nil, Dst: &net.IPNet{IP: net.IPv6zero}, Priority: 102}},
		linkNames: []string{"eth3", "eth3", "lo"},
	}

	ipv6NoInternetConnection = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Dst: &net.IPNet{IP: net.ParseIP("fd2e:6f44:5dd8:5::9b87")}, Gw: net.IPv6zero},
			{LinkIndex: 1, Dst: &net.IPNet{IP: net.ParseIP("fe80::5054:ff:fedd:a823")}, Gw: net.IPv6zero}},
		linkNames: []string{"docker0", "virbr0"},
	}
	ipV6GWNil = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Dst: nil, Gw: nil},
		},
		linkNames: []string{"eth3", "virbr0"},
	}

	ipv4Route = []*models.Route{
		{Interface: "eth3", Gateway: "10.254.0.1", Destination: "0.0.0.0", Family: int32(unix.AF_INET), Metric: 100},
		{Interface: "virbr0", Gateway: net.IPv4zero.String(), Destination: "192.168.122.0", Family: int32(unix.AF_INET), Metric: 100},
		{Interface: "virbr1", Destination: "192.168.122.0", Family: int32(unix.AF_INET), Metric: 101}}
	ipv4RouteNoInternetConnection = []*models.Route{
		{Destination: "10.254.0.0", Gateway: net.IPv4zero.String(), Interface: "docker0", Family: int32(unix.AF_INET)},
		{Destination: "172.17.0.0", Gateway: net.IPv4zero.String(), Interface: "virbr0", Family: int32(unix.AF_INET)},
	}

	ipv4RoutWithMultiPath = []*models.Route{
		{Interface: "eth3", Gateway: "10.254.0.1", Destination: "0.0.0.0", Family: int32(unix.AF_INET)}}

	ipv6Route = []*models.Route{
		{Interface: "eth3", Gateway: "2001:1::1", Destination: net.IPv6zero.String(), Family: int32(unix.AF_INET6), Metric: 101},
		{Interface: "eth3", Gateway: net.IPv6zero.String(), Destination: "2001:2::1", Family: int32(unix.AF_INET6), Metric: 101},
		{Interface: "lo", Destination: net.IPv6zero.String(), Family: int32(unix.AF_INET6), Metric: 102}}
	ipv6RouteNoInternetConnection = []*models.Route{
		{Destination: "fd2e:6f44:5dd8:5::9b87", Gateway: net.IPv6zero.String(), Interface: "docker0", Family: int32(unix.AF_INET6)},
		{Destination: "fe80::5054:ff:fedd:a823", Gateway: net.IPv6zero.String(), Interface: "virbr0", Family: int32(unix.AF_INET6)},
	}
	ipv6RouteGWNil = []*models.Route{{Interface: "eth3", Gateway: "", Destination: net.IPv6zero.String(), Family: int32(unix.AF_INET6)}}
)

type testHandler struct {
	routes        []netlink.Route
	linkNames     []string
	errorRoutes   error
	errorLinkName error
	family        int
}

func (th testHandler) getRouteList() ([]netlink.Route, error) {
	return th.routes, th.errorRoutes
}

func (th testHandler) getLinkName(route netlink.Route) (string, error) {
	if th.errorLinkName != nil {
		return "", th.errorLinkName
	}
	return th.linkNames[route.LinkIndex], nil
}

func (th testHandler) getFamily() int {
	return th.family
}

var _ = Describe("Route test", func() {
	var dependencies *util.MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	When("IPv4", func() {
		testCases := []struct {
			name       string
			handler    handler
			count      int
			expected   []*models.Route
			errStrFrag string
		}{
			{"should find all the routes when the default route is first", testHandler{routes: ipV4GW.routes, linkNames: ipV4GW.linkNames, family: unix.AF_INET}, len(ipV4GW.routes), ipv4Route, ""},
			{"should have no routes", testHandler{routes: nothing.routes, linkNames: nothing.linkNames, family: unix.AF_INET}, len(nothing.routes), []*models.Route{}, ""},
			{"should have routes when no internet connection/default route", testHandler{routes: ipv4NoInternetConnection.routes, linkNames: ipv4NoInternetConnection.linkNames, family: unix.AF_INET}, len(ipv4NoInternetConnection.routes), ipv4RouteNoInternetConnection, ""},
			{"should return error when retrieving routes", testHandler{errorRoutes: fmt.Errorf("cannot retrieve routes"), family: unix.AF_INET}, 0, nil, "cannot retrieve routes"},
			{"should skip routes when retrieving link name fails", testHandler{routes: ipV4GW.routes, errorLinkName: fmt.Errorf("cannot retrieve link name"), family: unix.AF_INET}, 0, nil, ""},
			{"should parse from multipath", testHandler{routes: ipv4WithMultiPath.routes, linkNames: ipv4WithMultiPath.linkNames, family: unix.AF_INET}, len(ipv4WithMultiPath.routes), ipv4RoutWithMultiPath, ""},
		}

		for _, tc := range testCases {
			tc := tc
			It(tc.name, func() {
				routes, err := getIPRoutes(tc.handler)
				if err != nil {
					Expect(err.Error()).To(ContainSubstring(tc.errStrFrag))
				} else {
					Expect(tc.errStrFrag).To(BeEmpty())
					Expect(tc.count).To(Equal(len(routes)))
					Expect(tc.expected).To(ContainElements(routes))
				}
			})
		}
	})

	When("IPv6", func() {
		testCases := []struct {
			name       string
			handler    handler
			count      int
			expected   []*models.Route
			errStrFrag string
		}{
			{"should find all the routes when the default route is first", testHandler{routes: ipV6GW.routes, linkNames: ipV6GW.linkNames, family: unix.AF_INET6}, len(ipV6GW.routes), ipv6Route, ""},
			{"should have no routes", testHandler{routes: nothing.routes, linkNames: nothing.linkNames, family: unix.AF_INET6}, len(nothing.routes), nil, ""},
			{"should have routes when no internet connection/default route", testHandler{routes: ipv6NoInternetConnection.routes, linkNames: ipv6NoInternetConnection.linkNames, family: unix.AF_INET6}, len(ipv6NoInternetConnection.routes), ipv6RouteNoInternetConnection, ""},
			{"should return error when retrieving routes", testHandler{errorRoutes: fmt.Errorf("cannot retrieve routes"), family: unix.AF_INET6}, 0, nil, "cannot retrieve routes"},
			{"should skip routes when retrieving link name fails", testHandler{routes: ipV6GW.routes, errorLinkName: fmt.Errorf("cannot retrieve link name"), family: unix.AF_INET6}, 0, nil, ""},
			{"should have a route when gateway is nil", testHandler{routes: ipV6GWNil.routes, linkNames: ipV6GWNil.linkNames, family: unix.AF_INET6}, len(ipV6GWNil.routes), ipv6RouteGWNil, ""},
		}
		for _, tc := range testCases {
			tc := tc
			It(tc.name, func() {
				routes, err := getIPRoutes(tc.handler)
				if err != nil {
					Expect(err.Error()).To(ContainSubstring(tc.errStrFrag))
				} else {
					Expect(tc.errStrFrag).To(BeEmpty())
					Expect(tc.count).To(Equal(len(routes)))
					Expect(tc.expected).To(ContainElements(routes))
				}
			})
		}
	})
})
