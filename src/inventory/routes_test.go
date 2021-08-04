package inventory

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/vishvananda/netlink"
)

type netPair struct {
	linkNames []string
	routes    []netlink.Route
}

var (
	ipV4GW = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Dst: nil, Gw: net.IPv4(10, 254, 0, 1)},
			{LinkIndex: 1, Dst: &net.IPNet{IP: net.IPv4(192, 168, 122, 0)}, Gw: net.IPv4zero}},
		linkNames: []string{"eth3", "virbr0"}}

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

	ipv4WithWrongFamily = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Dst: nil, Gw: net.IPv4(10, 254, 0, 1)},
			{MultiPath: []*netlink.NexthopInfo{{LinkIndex: 0, Gw: net.ParseIP("2001:1::1"), Hops: 1}}}},
		linkNames: []string{"eth3", "virbr0"},
	}

	nothing = netPair{
		routes:    []netlink.Route{},
		linkNames: []string{},
	}

	ipV6GW = netPair{
		routes: []netlink.Route{
			{LinkIndex: 0, Gw: net.ParseIP("2001:1::1"), Dst: &net.IPNet{IP: net.IPv6zero}},
			{LinkIndex: 1, Gw: net.IPv6zero, Dst: &net.IPNet{IP: net.ParseIP("2001:2::1")}},
			{LinkIndex: 2, Gw: net.IPv6zero, Dst: &net.IPNet{IP: net.IPv6zero}}},
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
		{Interface: "eth3", Gateway: "10.254.0.1", Destination: "0.0.0.0", Family: int32(familyIPv4)},
		{Interface: "virbr0", Gateway: net.IPv4zero.String(), Destination: "192.168.122.0", Family: int32(familyIPv4)}}
	ipv4RouteNoInternetConnection = []*models.Route{
		{Destination: "10.254.0.0", Gateway: net.IPv4zero.String(), Interface: "docker0", Family: int32(familyIPv4)},
		{Destination: "172.17.0.0", Gateway: net.IPv4zero.String(), Interface: "virbr0", Family: int32(familyIPv4)},
	}

	ipv4RoutWithMultiPath = []*models.Route{
		{Interface: "eth3", Gateway: "10.254.0.1", Destination: "0.0.0.0", Family: int32(familyIPv4)}}

	ipv6Route = []*models.Route{
		{Interface: "eth3", Gateway: "2001:1::1", Destination: net.IPv6zero.String(), Family: int32(familyIPv6)},
		{Interface: "eth3", Gateway: net.IPv6zero.String(), Destination: "2001:2::1", Family: int32(familyIPv6)},
		{Interface: "lo", Gateway: net.IPv6zero.String(), Destination: net.IPv6zero.String(), Family: int32(familyIPv6)}}
	ipv6RouteNoInternetConnection = []*models.Route{
		{Destination: "fd2e:6f44:5dd8:5::9b87", Gateway: net.IPv6zero.String(), Interface: "docker0", Family: int32(familyIPv6)},
		{Destination: "fe80::5054:ff:fedd:a823", Gateway: net.IPv6zero.String(), Interface: "virbr0", Family: int32(familyIPv6)},
	}
	ipv6RouteGWNil = []*models.Route{{Interface: "eth3", Gateway: "", Destination: net.IPv6zero.String(), Family: int32(familyIPv6)}}
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
			{"should find all the routes when the default route is first", testHandler{routes: ipV4GW.routes, linkNames: ipV4GW.linkNames, family: familyIPv4}, len(ipV4GW.routes), ipv4Route, ""},
			{"should have no routes", testHandler{routes: nothing.routes, linkNames: nothing.linkNames, family: familyIPv4}, len(nothing.routes), []*models.Route{}, ""},
			{"should have routes when no internet connection/default route", testHandler{routes: ipv4NoInternetConnection.routes, linkNames: ipv4NoInternetConnection.linkNames, family: familyIPv4}, len(ipv4NoInternetConnection.routes), ipv4RouteNoInternetConnection, ""},
			{"should return error when retrieving routes", testHandler{errorRoutes: fmt.Errorf("cannot retrieve routes"), family: familyIPv4}, 0, nil, "cannot retrieve routes"},
			{"should return error when retrieving link name", testHandler{routes: ipV4GW.routes, errorLinkName: fmt.Errorf("cannot retrieve link name"), family: familyIPv4}, 0, nil, "cannot retrieve link name"},
			{"should parse from multipath", testHandler{routes: ipv4WithMultiPath.routes, linkNames: ipv4WithMultiPath.linkNames, family: familyIPv4}, len(ipv4WithMultiPath.routes), ipv4RoutWithMultiPath, ""},
			{"should filter when route contains the wrong family", testHandler{routes: ipv4WithWrongFamily.routes, linkNames: ipv4WithWrongFamily.linkNames, family: familyIPv4}, 1, ipv4Route, ""},
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
			{"should find all the routes when the default route is first", testHandler{routes: ipV6GW.routes, linkNames: ipV6GW.linkNames, family: familyIPv6}, len(ipV6GW.routes), ipv6Route, ""},
			{"should have no routes", testHandler{routes: nothing.routes, linkNames: nothing.linkNames, family: familyIPv6}, len(nothing.routes), nil, ""},
			{"should have routes when no internet connection/default route", testHandler{routes: ipv6NoInternetConnection.routes, linkNames: ipv6NoInternetConnection.linkNames, family: familyIPv6}, len(ipv6NoInternetConnection.routes), ipv6RouteNoInternetConnection, ""},
			{"should return error when retrieving routes", testHandler{errorRoutes: fmt.Errorf("cannot retrieve routes"), family: familyIPv6}, 0, nil, "cannot retrieve routes"},
			{"should return error when retrieving link name", testHandler{routes: ipV6GW.routes, errorLinkName: fmt.Errorf("cannot retrieve link name"), family: familyIPv6}, 0, nil, "cannot retrieve link name"},
			{"should have a route when gateway is nil", testHandler{routes: ipV6GWNil.routes, linkNames: ipV6GWNil.linkNames, family: familyIPv6}, len(ipV6GWNil.routes), ipv6RouteGWNil, ""},
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
