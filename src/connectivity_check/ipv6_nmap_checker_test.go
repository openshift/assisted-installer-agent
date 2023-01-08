package connectivity_check

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

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
	DescribeTable("nmap test cases",
		func(remoteIPAddress, remoteMACAddress, outgoingNIC string, remoteMACAddresses []string, output string, err error, expected *models.L2Connectivity) {
			e := &MockExecuter{}
			e.On("Execute", "nmap", "-6", "-sn", "-n", "-oX", "-", "-e", outgoingNIC, remoteIPAddress).Return(output, err).Once()
			checker := &nmapChecker{
				executer: e,
			}
			attributes := Attributes{
				RemoteIPAddress:    remoteIPAddress,
				RemoteMACAddress:   remoteMACAddress,
				OutgoingNIC:        outgoingNIC,
				RemoteMACAddresses: remoteMACAddresses,
			}
			var remoteHost models.ConnectivityRemoteHost
			if reporter := checker.Check(attributes); reporter != nil {
				Expect(reporter.Report(&remoteHost)).ToNot(HaveOccurred())
			}
			if expected == nil {
				Expect(remoteHost.L2Connectivity).To(BeEmpty())
			} else {
				Expect(remoteHost.L2Connectivity).To(HaveLen(1))
				Expect(remoteHost.L2Connectivity[0]).To(Equal(expected))
			}
			e.AssertExpectations(GinkgoT())
		},
		Entry("Happy flow", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, nmapOut, nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		),
		Entry("Command error", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, nmapOut, errors.New("nmap command failed"),
			nil),
		Entry("Invalid XML", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "plain text", nil,
			nil),
		Entry("Host down", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			`<nmaprun>
					<host>
						<status state="down" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:AC:12:00:02" addrtype="mac" />
					</host>
				</nmaprun>`, nil,
			nil),
		Entry("Lower-case destination MAC address", "2001:db8::2", "02:42:ac:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, nmapOut, nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		),
		Entry("Lower-case discovered MAC address", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			`
				<nmaprun>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
						<address addr="02:42:ac:12:00:02" addrtype="mac" />
					</host>
				</nmaprun>`,
			nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		),
		Entry("No MAC address", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			`
				<nmaprun>
					<host>
						<status state="up" />
						<address addr="2001:db8::2" addrtype="ipv6" />
					</host>
				</nmaprun>`, nil,
			nil),
		Entry("No hosts", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			`<nmaprun />`, nil,
			nil),
		Entry("First matching host", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			`<nmaprun>
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
				</nmaprun>`, nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:aa:00:02",
				Successful:      false,
			}),
		Entry("Multiple hosts, only one up", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			`
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
				</nmaprun>`, nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			}),
		Entry("Multiple hosts, only one has a MAC address", "2001:db8::2", "02:42:AC:12:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"},
			`
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
				</nmaprun>`, nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			}),
		Entry("Unexpected MAC address", "2001:db8::2", "02:42:CC:14:00:02", "eth0", []string{"02:42:B:14:00:02", "02:42:C:14:00:02"},
			nmapOut, nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
			}),
		Entry("MAC different than tried", "2001:db8::2", "02:42:CC:10:00:02", "eth0", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, nmapOut, nil,
			&models.L2Connectivity{
				OutgoingNic:     "eth0",
				RemoteIPAddress: "2001:db8::2",
				RemoteMac:       "02:42:ac:12:00:02",
				Successful:      true,
			},
		),
	)
})
