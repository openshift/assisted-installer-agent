package nmap

import "encoding/xml"

type Status struct {
	State  string `xml:"state,attr"`
	Reason string `xml:"reason,attr"`
}

type Address struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type Host struct {
	Status    Status     `xml:"status"`
	Addresses []*Address `xml:"address"`
}

type Nmaprun struct {
	XMLName xml.Name `xml:"nmaprun"`
	Hosts   []*Host  `xml:"host"`
}
