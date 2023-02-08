package net

import (
	"encoding/json"
	"fmt"
	"net"
)

type Hostname struct {
	Running string `json:"running"`
}

type DNSResolver struct {
	Running DNSConfig `json:"running,omitempty"`
}

type DNSConfig struct {
	Servers       []string `json:"server,omitempty"`
	SearchDomains []string `json:"search,omitempty"`
}

type RoutesRC struct {
	Running []Route `json:"running"`
}

type Route struct {
	Destination  string `json:"destination"`
	NextHopIface string `json:"next-hop-interface"`
	NextHopAddr  string `json:"next-hop-address"`
}

type IPConfig struct {
	Enabled   bool        `json:"enabled,omitempty"`
	Addresses []net.IPNet `json:"address,omitempty"`
}

func (ipc *IPConfig) UnmarshalJSON(data []byte) error {
	type _Address struct {
		IP        string `json:"ip"`
		Prefixlen int    `json:"prefix-length"`
	}
	type _ipConfig struct {
		Enabled   bool       `json:"enabled,omitempty"`
		Addresses []_Address `json:"address,omitempty"`
	}

	tempIpConfig := _ipConfig{}
	if err := json.Unmarshal(data, &tempIpConfig); err != nil {
		return err
	}

	ipc.Enabled = tempIpConfig.Enabled
	for _, address := range tempIpConfig.Addresses {
		ip, netCIDR, err := net.ParseCIDR(fmt.Sprintf("%s/%d", address.IP, address.Prefixlen))
		if err != nil {
			return err
		}
		ipc.Addresses = append(ipc.Addresses, net.IPNet{
			IP:   ip,
			Mask: netCIDR.Mask,
		})
	}
	return nil
}

type Iface struct {
	Name  string   `json:"name"`
	Type  string   `json:"type"`
	State string   `json:"state"`
	MTU   int      `json:"mtu"`
	IPv4  IPConfig `json:"ipv4,omitempty"`
	IPv6  IPConfig `json:"ipv6,omitempty"`
}

type NetState struct {
	Hostname Hostname    `json:"hostname,omitempty"`
	DNS      DNSResolver `json:"dns-resolver,omitempty"`
	Routes   RoutesRC    `json:"routes,omitempty"`
	Ifaces   []Iface     `json:"interfaces"`
}

func IsIPv4DefaultRoute(destination string) (isDefaultRoute bool) {
	switch destination {
	case "0/0", "0.0.0.0/0":
		isDefaultRoute = true
	default:
		isDefaultRoute = false
	}
	return
}

func IsIPv6DefaultRoute(destination string) (isDefaultRoute bool) {
	switch destination {
	case "::/0":
		isDefaultRoute = true
	default:
		isDefaultRoute = false
	}
	return
}

func (ns *NetState) getIfaceByName(ifaceName string) (r *Iface) {
	for i := range ns.Ifaces {
		if ns.Ifaces[i].Name == ifaceName {
			r = &ns.Ifaces[i]
		}
	}
	return
}

func (ns *NetState) GetDefaultNextHopIface() (r *Iface, err error) {
	for _, route := range ns.Routes.Running {
		if IsIPv4DefaultRoute(route.Destination) || IsIPv6DefaultRoute(route.Destination) {
			if r != nil {
				return nil, fmt.Errorf("support for multiple default routes not yet implemented in agent-tui")
			}
			r = ns.getIfaceByName(route.NextHopIface)
		}
	}
	return
}
