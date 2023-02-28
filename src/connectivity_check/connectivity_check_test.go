package connectivity_check

import (
	"errors"
	"fmt"
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"
)

var _ = Describe("check host parallel validation", func() {

	var (
		hostChan chan models.ConnectivityRemoteHost
	)
	BeforeEach(func() {
		hostChan = make(chan models.ConnectivityRemoteHost)
	})

	AfterEach(func() {
		close(hostChan)
	})

	setupDispather := func(simulateL2IPConflict, success bool, nics []*models.ConnectivityCheckNic) *connectivityRunner {
		e := &executerMock{
			success:              success,
			simulateL2IPConflict: simulateL2IPConflict,
			nics:                 nics,
		}
		return &connectivityRunner{
			checkers: []Checker{
				&pingChecker{executer: e}, &arpingChecker{executer: e}, &nmapChecker{executer: e},
			},
		}
	}

	tests := []struct {
		name                 string
		nics                 []string
		hosts                *models.ConnectivityCheckHost
		expected             *models.ConnectivityRemoteHost
		success              bool
		l2Conn               []*models.L2Connectivity
		l3Conn               []*models.L3Connectivity
		simulateL2IPConflict bool
		strictMatchingL2     bool
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
						PacketLossPercentage: 60,
						RemoteIPAddress:      "192.168.1.1",
						Successful:           true,
					},
					{AverageRTTMs: 2.871,
						PacketLossPercentage: 60,
						RemoteIPAddress:      "192.168.1.2",
						Successful:           true,
					},
				},
			},
			simulateL2IPConflict: false,
			strictMatchingL2:     false,
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

						PacketLossPercentage: 60,
						RemoteIPAddress:      "fe80::acae:f113:f40:cfe1",
						Successful:           true,
					},
				},
			},
			simulateL2IPConflict: false,
			strictMatchingL2:     false,
		},
		{name: "KO: IPv4 unable to connect via ping or arp",
			success: false,
			nics:    []string{"nic_ipv4", "nic_ipv41", "nic_ipv42"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"192.168.1.1"}, Mac: "4c:1d:96:af:22:65"},
			}},
			expected:             &models.ConnectivityRemoteHost{},
			simulateL2IPConflict: false,
			strictMatchingL2:     false,
		},
		{name: "KO: IPv6 unable to connect via ping or nmap",
			success: false,
			nics:    []string{"nic_ipv6"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"fe80::acae:f113:f40:cfe1"}, Mac: "4c:1d:96:af:22:65"},
			}},
			expected:             &models.ConnectivityRemoteHost{},
			simulateL2IPConflict: false,
			strictMatchingL2:     false,
		},
		{name: "Should correctly report on L2 IP conflicts",
			success: true,
			nics:    []string{"nic_ipv4"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"192.168.1.1"}, Mac: "74:d0:2b:1c:c6:42"},
			}},
			expected: &models.ConnectivityRemoteHost{
				L2Connectivity: []*models.L2Connectivity{
					{OutgoingNic: "nic_ipv4",
						RemoteIPAddress:   "192.168.1.1",
						OutgoingIPAddress: "192.168.1.133",
						Successful:        false,
						RemoteMac:         "00:50:56:95:ba:55"},
					{OutgoingNic: "nic_ipv4",
						RemoteIPAddress:   "192.168.1.1",
						OutgoingIPAddress: "192.168.1.133",
						Successful:        true,
						RemoteMac:         "74:d0:2b:1c:c6:42"},
				},
				L3Connectivity: []*models.L3Connectivity{
					{AverageRTTMs: 2.871,
						PacketLossPercentage: 60,
						RemoteIPAddress:      "192.168.1.1",
						Successful:           true,
					},
				},
			},
			simulateL2IPConflict: true,
			strictMatchingL2:     false,
		},
		{name: "Should deduplicate L2 entries and ensure consistent order of L2 entries",
			success: true,
			nics:    []string{"nic_ipv4"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"192.168.1.1"}, Mac: "74:d0:2b:1c:c6:42"},
			}},
			expected: &models.ConnectivityRemoteHost{
				L2Connectivity: []*models.L2Connectivity{
					{OutgoingNic: "nic_ipv4",
						RemoteIPAddress:   "192.168.1.1",
						OutgoingIPAddress: "192.168.1.133",
						Successful:        false,
						RemoteMac:         "00:50:56:95:ba:55"},
					{OutgoingNic: "nic_ipv4",
						RemoteIPAddress:   "192.168.1.1",
						OutgoingIPAddress: "192.168.1.133",
						Successful:        true,
						RemoteMac:         "74:d0:2b:1c:c6:42"},
				},
				L3Connectivity: []*models.L3Connectivity{},
			},
			simulateL2IPConflict: true,
			strictMatchingL2:     true,
		},
		{name: "Dual stack",
			success: true,
			nics:    []string{"nic"},
			hosts: &models.ConnectivityCheckHost{Nics: []*models.ConnectivityCheckNic{
				{IPAddresses: []string{"192.168.1.1", "fe80::d832:8def:dd51:3527"}, Mac: "74:d0:2b:1c:c6:42"},
			}},
			expected: &models.ConnectivityRemoteHost{
				L2Connectivity: []*models.L2Connectivity{
					{OutgoingNic: "nic",
						RemoteIPAddress:   "192.168.1.1",
						OutgoingIPAddress: "192.168.1.133",
						Successful:        true,
						RemoteMac:         "74:d0:2b:1c:c6:42"},
					{OutgoingNic: "nic",
						RemoteIPAddress:   "fe80::d832:8def:dd51:3527",
						OutgoingIPAddress: "",
						Successful:        true,
						RemoteMac:         "74:d0:2b:1c:c6:42"},
				},
				L3Connectivity: []*models.L3Connectivity{},
			},
			simulateL2IPConflict: false,
			strictMatchingL2:     true,
		},
	}

	for i := range tests {
		t := tests[i]
		It(t.name, func() {
			d := setupDispather(t.simulateL2IPConflict, t.success, t.hosts.Nics)
			ret, err := d.Run(models.ConnectivityCheckParams{t.hosts}, funk.Map(t.nics, func(s string) OutgoingNic {
				return OutgoingNic{Name: s, HasIpv4Addresses: true, HasIpv6Addresses: true}
			}).([]OutgoingNic))
			Expect(err).ToNot(HaveOccurred())
			Expect(ret.RemoteHosts).To(HaveLen(1))
			if !t.strictMatchingL2 {
				Expect(ret.RemoteHosts[0].L2Connectivity).Should(ContainElements(t.expected.L2Connectivity))
			} else {
				Expect(ret.RemoteHosts[0].L2Connectivity).Should(BeEquivalentTo(t.expected.L2Connectivity))
			}
			Expect(ret.RemoteHosts[0].L3Connectivity).Should(ContainElements(t.expected.L3Connectivity))
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
				util.NewFilledMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.18/24", "192.168.6.7/20", "fe80::d832:8def:dd51:3527/128", "de90::d832:8def:dd51:3527/128"}, 100, "physical"),
				util.NewFilledMockInterface(1400, "eth1", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, []string{}, 10, "physical"),
				util.NewFilledMockInterface(1400, "eth2", "f8:75:a4:a4:00:ff", net.FlagBroadcast|net.FlagLoopback, nil, 5, "physical"),
				util.NewFilledMockInterface(1400, "bond0", "f8:75:a4:a4:00:fd", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.21/24", "192.168.6.10/20", "fe80::d832:8def:dd51:3529/125", "de90::d832:8def:dd51:3529/125"}, -1, "bond"),
				util.NewFilledMockInterface(1400, "eth2.10", "f8:75:a4:a4:00:fc", net.FlagBroadcast|net.FlagUp, []string{"10.0.0.25/24", "192.168.6.14/20", "fe80::d832:8def:dd51:3520/125", "de90::d832:8def:dd51:3520/125"}, -1, "vlan"),
				util.NewFilledMockInterface(1400, "ib2", "f8:75:a4:a4:00:fa", net.FlagBroadcast|net.FlagUp, []string{"fe80:39:192:1::1193/64"}, -1, "some-strange-type"),
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

type executerMock struct {
	success              bool
	simulateL2IPConflict bool
	nics                 []*models.ConnectivityCheckNic
}

func (f *executerMock) Execute(command string, args ...string) (string, error) {
	var mac string
	if f.success {
		for _, h := range f.nics {
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
	switch command {
	case "ping":
		if f.success {
			return fmt.Sprintf(`PING %[1]s (%[1]s) 56(84) bytes of data.

		--- %[1]s ping statistics ---
		10 packets transmitted, 4 received, 60%% packet loss, time 9164ms
		rtt min/avg/max/mdev = 2.616/2.871/3.183/0.255 ms`, args[0]), nil
		}
		return "", errors.New("unable to connect")
	case "nmap":
		if f.success {
			return fmt.Sprintf(`<nmaprun>
						<host>
						<status state="up" />
						<address addr="%s" addrtype="ipv6" />
						<address addr="%s" addrtype="mac" />
						</host>
						</nmaprun>`, args[7], mac), nil
		}
		return "", errors.New("unable to connect via nmap")
	case "arping":

		if f.simulateL2IPConflict {
			return fmt.Sprintf(`ARPING %[1]s from 192.168.1.133 %[2]s
Unicast reply from %[1]s [00:50:56:95:BA:55]  1.871ms
Unicast reply from %[1]s [00:50:56:95:BA:55]  1.871ms
Unicast reply from %[1]s [%[3]s]  3.137ms
Sent 1 probes (1 broadcast(s))
Received 1 response(s)
						`, args[5], args[6], mac), nil
		}

		if f.success {
			return fmt.Sprintf(`ARPING %[1]s from 192.168.1.133 %[2]s
Unicast reply from %[1]s [%[3]s]  3.137ms
Sent 1 probes (1 broadcast(s))
Received 1 response(s)
			`, args[5], args[6], mac), nil
		}
		return fmt.Sprintf(`ARPING %[1]s from 192.168.1.133 %[2]s
Sent 1 probes (1 broadcast(s))
Received 0 response(s)`, args[5], args[6]), nil
	default:
		log.Errorf("failed to process unknown command %s with arguments %+v", command, args)
		return "", fmt.Errorf("unknown command %s", command)
	}
}

func TestConnectivityCheck(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "connectivity check tests")
}
