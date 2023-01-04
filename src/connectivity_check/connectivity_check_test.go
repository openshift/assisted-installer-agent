package connectivity_check

import (
	"errors"
	"fmt"
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/assisted-service/models"
)

const nmapOut = `
<nmaprun>
	<host>
		<status state="up" />
		<address addr="2001:db8::2" addrtype="ipv6" />
		<address addr="02:42:AC:12:00:02" addrtype="mac" />
	</host>
</nmaprun>`

var _ = Describe("nmap analysis test", func() {

	tests := []struct {
		name       string
		dstAddr    string
		dstMAC     string
		srcNIC     string
		allDstMACs []string
		output     func() ([]byte, error)
		expected   *models.L2Connectivity
	}{
		{name: "Happy flow",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(nmapOut), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		},
		{name: "Command error",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(nmapOut), fmt.Errorf("nmap command failed")
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
			}},
		{name: "Invalid XML",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte("plain text"), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
			}},

		{name: "Host down",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(`
				<nmaprun>
					<host>
						<status state="down" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:AC:12:00:02" addrtype="mac" />
					</host>
				</nmaprun>`), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				Successful:      false,
			},
		},
		{name: "Lower-case destination MAC address",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:ac:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(nmapOut), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		},
		{name: "Lower-case discovered MAC address",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(`
				<nmaprun>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:ac:12:00:02" addrtype="mac" />
					</host>
				</nmaprun>`), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		},
		{name: "No MAC address",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(`
				<nmaprun>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
					</host>
				</nmaprun>`), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
			},
		},
		{name: "No hosts",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte("<nmaprun />"), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
			},
		},
		{name: "First matching host",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(`
				<nmaprun>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:AC:AA:00:02" addrtype="mac" />
					</host>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:AC:12:00:02" addrtype="mac" />
					</host>
				</nmaprun>`), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:aa:00:02",
				Successful:      false,
			},
		},
		{name: "Multiple hosts, only one up",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(`
				<nmaprun>
					<host>
						<status state="down" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:AC:AA:00:02" addrtype="mac" />
					</host>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:AC:12:00:02" addrtype="mac" />
					</host>
				</nmaprun>`), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		},
		{name: "Multiple hosts, only one has a MAC address",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:AC:12:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(`
				<nmaprun>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
					</host>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:AC:12:00:02" addrtype="mac" />
					</host>
				</nmaprun>`), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		},
		{name: "Unexpected MAC address",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:CC:14:00:02",
			allDstMACs: []string{"02:42:B:14:00:02", "02:42:C:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(nmapOut), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
			},
		},
		{name: "MAC different than tried",
			dstAddr:    "2001:db8::2",
			dstMAC:     "02:42:CC:10:00:02",
			allDstMACs: []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			output: func() ([]byte, error) {
				return []byte(nmapOut), nil
			},
			expected: &models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		},
	}
	for i := range tests {
		t := tests[i]
		It(t.name, func() {
			out := make(chan any)
			go analyzeNmap(t.dstAddr, t.dstMAC, t.allDstMACs, "eth0", out, testNmap{t.output}, false)
			Expect(<-out).To(Equal(t.expected))
		})
	}

})

type testNmap struct {
	output func() ([]byte, error)
}

func (tn testNmap) command(name string, args []string) ([]byte, error) {
	return tn.output()
}

func (tn testNmap) getHost() *models.ConnectivityCheckHost {
	return nil
}
func (tn testNmap) getOutgoingNICs() []string {

	return nil
}

var _ = Describe("parse ping command", func() {

	tests := []struct {
		name         string
		cmdOutput    string
		averageRTTMs float64
		packetLoss   float64
		errFunc      func(string) string
	}{
		{name: "Nominal: no packet loss",
			cmdOutput: `PING www.acme.com (127.0.0.1) 56(84) bytes of data.

		--- www.acme.com ping statistics ---
		10 packets transmitted, 10 received, 0% packet loss, time 9011ms
		rtt min/avg/max/mdev = 14.278/17.099/19.136/1.876 ms`,
			averageRTTMs: 17.099,
			packetLoss:   0,
		},
		{name: "Nominal: with packet loss",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

		--- 192.168.1.1 ping statistics ---
		10 packets transmitted, 4 received, 60% packet loss, time 9164ms
		rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			averageRTTMs: 2.871,
			packetLoss:   60,
		},
		{name: "Nominal: with packet loss with decimals",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

		--- 192.168.1.1 ping statistics ---
		10 packets transmitted, 4 received, 23.33% packet loss, time 9164ms
		rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			averageRTTMs: 2.871,
			packetLoss:   23.33,
		},
		{name: "KO: unable to parse average RTT",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

			--- 192.168.1.1 ping statistics ---
			10 packets transmitted, 4 received, 60% packet loss, time 9164ms
			rtt min/average/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			averageRTTMs: 0,
			packetLoss:   60,
			errFunc: func(s string) string {
				return fmt.Sprintf(`Unable to retrieve the average RTT for ping: unable to parse %s with regex rtt min\/avg\/max\/mdev = .*\/([^\/]+)\/.*\/.* ms`, s)
			},
		},
		{name: "KO: unable to parse packets loss percentage",
			cmdOutput: `PING 192.168.1.1 (192.168.1.1) 56(84) bytes of data.

			--- 192.168.1.1 ping statistics ---
			10 packets transmitted, 4 received, 60%  packet loss, time 9164ms
			rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`,
			errFunc: func(s string) string {
				return fmt.Sprintf(`Unable to retrieve packet loss percentage: unable to parse %s with regex [\d]+ packets transmitted, [\d]+ received, (([\d]*[.])?[\d]+)%% packet loss, time [\d]+ms`, s)
			},
		},
	}
	for i := range tests {
		t := tests[i]
		It(t.name, func() {
			conn := models.L3Connectivity{}
			err := parsePingCmd(&conn, t.cmdOutput)
			if t.errFunc != nil {
				Expect(err.Error()).To(BeEquivalentTo(t.errFunc(t.cmdOutput)))
			} else {
				Expect(err).To(BeNil())
			}
			Expect(conn.AverageRTTMs).Should(Equal(t.averageRTTMs))
			Expect(conn.PacketLossPercentage).Should(Equal(t.packetLoss))
		})
	}

})

var _ = Describe("check host parallel validation", func() {

	var (
		hostChan chan *models.ConnectivityRemoteHost
	)
	BeforeEach(func() {
		hostChan = make(chan *models.ConnectivityRemoteHost)
	})

	AfterEach(func() {
		close(hostChan)
	})

	tests := []struct {
		name     string
		nics     []string
		hosts    *models.ConnectivityCheckHost
		expected *models.ConnectivityRemoteHost
		success  bool
		l2Conn   []*models.L2Connectivity
		l3Conn   []*models.L3Connectivity
	}{
		{
			name:    "Nominal: IPv4 with 2 addresses",
			success: true,
			nics:    []string{"nic_ipv4"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"192.168.1.1"}, Mac: "74:d0:2b:1c:c6:42"},
				{IPAddresses: []string{"192.168.1.2"}, Mac: "f8:75:a4:4a:33:07"},
			}},
			expected: &models.ConnectivityRemoteHost{
				L2Connectivity: []*models.L2Connectivity{
					{OutgoingNic: "nic_ipv4",
						RemoteIPAddress:   "192.168.1.1",
						OutgoingIPAddress: "192.168.1.133",
						Successful:        true,
						RemoteMac:         "74:d0:2b:1c:c6:42"},
					{OutgoingNic: "nic_ipv4",
						RemoteIPAddress:   "192.168.1.2",
						OutgoingIPAddress: "192.168.1.133",
						Successful:        true,
						RemoteMac:         "f8:75:a4:4a:33:07"},
				},
				L3Connectivity: []*models.L3Connectivity{
					{AverageRTTMs: 2.871,
						OutgoingNic:          "nic_ipv4",
						PacketLossPercentage: 60,
						RemoteIPAddress:      "192.168.1.1",
						Successful:           true,
					},
					{AverageRTTMs: 2.871,
						OutgoingNic:          "nic_ipv4",
						PacketLossPercentage: 60,
						RemoteIPAddress:      "192.168.1.2",
						Successful:           true,
					},
				},
			},
		},
		{name: "Nominal: IPv6",
			success: true,
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"fe80::acae:f113:f40:cfe1"}, Mac: "4c:1d:96:af:22:65"},
			}},
			nics: []string{"nic_ipv6"},
			expected: &models.ConnectivityRemoteHost{
				L2Connectivity: []*models.L2Connectivity{
					{OutgoingNic: "nic_ipv6",
						RemoteIPAddress: "fe80::acae:f113:f40:cfe1",
						RemoteMac:       "4c:1d:96:af:22:65",
						Successful:      true},
				},
				L3Connectivity: []*models.L3Connectivity{
					{AverageRTTMs: 2.871,
						OutgoingNic:          "nic_ipv6",
						PacketLossPercentage: 60,
						RemoteIPAddress:      "fe80::acae:f113:f40:cfe1",
						Successful:           true,
					},
				},
			},
		},
		{name: "KO: IPv4 unable to connect via ping or arp",
			success: false,
			nics:    []string{"nic_ipv4", "nic_ipv41", "nic_ipv42"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"192.168.1.1"}, Mac: "4c:1d:96:af:22:65"},
			}},
			expected: &models.ConnectivityRemoteHost{
				L2Connectivity: []*models.L2Connectivity{
					{OutgoingNic: "",
						RemoteIPAddress: "192.168.1.1",
					},
				},
				L3Connectivity: []*models.L3Connectivity{
					{OutgoingNic: "",
						RemoteIPAddress: "192.168.1.1",
					},
				},
			},
		},
		{name: "KO: IPv6 unable to connect via ping or nmap",
			success: false,
			nics:    []string{"nic_ipv6"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"fe80::acae:f113:f40:cfe1"}, Mac: "4c:1d:96:af:22:65"},
			}},
			expected: &models.ConnectivityRemoteHost{
				L2Connectivity: []*models.L2Connectivity{
					{OutgoingNic: "nic_ipv6",
						RemoteIPAddress: "fe80::acae:f113:f40:cfe1",
					}},
				L3Connectivity: []*models.L3Connectivity{
					{OutgoingNic: "",
						RemoteIPAddress: "fe80::acae:f113:f40:cfe1",
					},
				},
			},
		},
	}

	for i := range tests {
		t := tests[i]
		It(t.name, func() {
			h := testHostChecker{outgoingNICS: t.nics, host: t.hosts, success: t.success}
			c := connectivity{dryRunConfig: &config.DryRunConfig{}}
			go c.checkHost(h, hostChan)
			r := <-hostChan
			Expect(r.L2Connectivity).Should(ContainElements(t.expected.L2Connectivity))
			Expect(r.L3Connectivity).Should(ContainElements(t.expected.L3Connectivity))
		})
	}

})

func newDependenciesMock() *util.MockIDependencies {
	d := &util.MockIDependencies{}
	mockGetGhwChrootRoot(d)
	return d
}

func mockGetGhwChrootRoot(dependencies *util.MockIDependencies) {
	dependencies.On("GetGhwChrootRoot").Return("/host").Maybe()
}

var _ = Describe("getOutgoingNics", func() {
	var interfaces []util.Interface
	var dependencies *util.MockIDependencies

	BeforeEach(func() {
		dependencies = newDependenciesMock()
	})

	Context("for hosts with all types of interfaces", func() {
		BeforeEach(func() {
			interfaces = []util.Interface{
				util.NewMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "192.168.6.7/20", "fe80::d832:8def:dd51:3527/128", "de90::d832:8def:dd51:3527/128"}, 100, "physical"),
				util.NewMockInterface(1400, "eth1", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{}, 10, "physical"),
				util.NewMockInterface(1400, "eth2", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, nil, 5, "physical"),
				util.NewMockInterface(1400, "bond0", "f8:75:a4:a4:00:fd", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.21/24", "192.168.6.10/20", "fe80::d832:8def:dd51:3529/125", "de90::d832:8def:dd51:3529/125"}, -1, "bond"),
				util.NewMockInterface(1400, "eth2.10", "f8:75:a4:a4:00:fc", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.25/24", "192.168.6.14/20", "fe80::d832:8def:dd51:3520/125", "de90::d832:8def:dd51:3520/125"}, -1, "vlan"),
				util.NewMockInterface(1400, "ib2", "f8:75:a4:a4:00:fa", net.FlagBroadcast|net.FlagUp, []string{"fe80:39:192:1::1193/64"}, -1, "some-strange-type"),
			}
			dependencies.On("Interfaces").Return(interfaces, nil).Once()
		})

		It("returns only non-virtual interfaces with address", func() {
			ret := getOutgoingNics(nil, dependencies)
			Expect(len(ret)).To(Equal(3))
			Expect(ret).ToNot(ContainElement("eth1"))
			Expect(ret).ToNot(ContainElement("eth2"))
		})
	})
})

type testHostChecker struct {
	success      bool
	outgoingNICS []string
	host         *models.ConnectivityCheckHost
}

func (t testHostChecker) command(name string, args []string) ([]byte, error) {
	var mac string
	if t.success {
		for _, h := range t.host.Nics {
			for _, ip := range h.IPAddresses {
				if ip == args[len(args)-1] {
					mac = h.Mac.String()
					break
				}
			}
			if len(mac) > 0 {
				break
			}
		}
	}
	switch name {
	case "ping":
		if t.success {
			return []byte(fmt.Sprintf(`PING %[1]s (%[1]s) 56(84) bytes of data.

		--- %[1]s ping statistics ---
		10 packets transmitted, 4 received, 60%% packet loss, time 9164ms
		rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`, args[0])), nil
		}
		return nil, errors.New("unable to connect")
	case "nmap":
		if t.success {
			return []byte(fmt.Sprintf(`<nmaprun>
						<host>
						<status state="up" />
						<address addr="%s" addrtype="ipv6" />
						<address addr="%s" addrtype="mac" />
						</host>
						</nmaprun>`, args[7], mac)), nil
		}
		return nil, errors.New("unable to connect via nmap")
	case "arping":
		if t.success {
			return []byte(fmt.Sprintf(`ARPING %[1]s from 192.168.1.133 %[2]s
Unicast reply from %[1]s [%[3]s]  3.137ms
Sent 1 probes (1 broadcast(s))
Received 1 response(s)
			`, args[5], args[6], mac)), nil
		}
		return []byte(fmt.Sprintf(`ARPING %[1]s from 192.168.1.133 %[2]s
Sent 1 probes (1 broadcast(s))
Received 0 response(s)`, args[5], args[6])), nil
	default:
		log.Errorf("failed to process unknown command %s with arguments %+v", name, args)
		return nil, fmt.Errorf("unknown command %s", name)
	}
}

func (t testHostChecker) getHost() *models.ConnectivityCheckHost {
	return t.host
}

func (t testHostChecker) getOutgoingNICs() []string {
	return t.outgoingNICS
}

func TestConnectivityCheck(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "connectivity check tests")
}
