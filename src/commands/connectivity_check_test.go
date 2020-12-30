package commands

import (
	"fmt"
	"testing"

	"github.com/onsi/ginkgo"
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

var _ = ginkgo.Describe("nmap analysis test", func() {

	ginkgo.It("Happy flow", func(done ginkgo.Done) {

		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(nmapOut), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:12:00:02",
			Successful:        true,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Command error", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(nmapOut), fmt.Errorf("nmap command failed")
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "",
			Successful:        false,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Invalid XML", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte("plain text"), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "",
			Successful:        false,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Host down", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(`
			<nmaprun>
				<host>
					<status state="down" />
					<address addr="2001:db8::2" addrtype="ipv6" />
					<address addr="02:42:AC:12:00:02" addrtype="mac" />
				</host>
			</nmaprun>`), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "",
			Successful:        false,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Lower-case destination MAC address", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(nmapOut), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:12:00:02",
			Successful:        true,
		}

		analyzeNmap("2001:db8::2", "02:42:ac:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Lower-case discovered MAC address", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(`
			<nmaprun>
				<host>
					<status state="up" />
					<address addr="2001:db8::2" addrtype="ipv6" />
					<address addr="02:42:ac:12:00:02" addrtype="mac" />
				</host>
			</nmaprun>`), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:12:00:02",
			Successful:        true,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("No MAC address", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(`
			<nmaprun>
				<host>
					<status state="up" />
					<address addr="2001:db8::2" addrtype="ipv6" />
				</host>
			</nmaprun>`), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "",
			Successful:        false,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("No hosts", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte("<nmaprun />"), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "",
			Successful:        false,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("First matching host", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
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
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:aa:00:02",
			Successful:        false,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Multiple hosts, only one up", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
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
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:12:00:02",
			Successful:        true,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Multiple hosts, only one has a MAC address", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
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
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:12:00:02",
			Successful:        true,
		}

		analyzeNmap("2001:db8::2", "02:42:AC:12:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("Unexpected MAC address", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(nmapOut), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:12:00:02",
			Successful:        false,
		}

		analyzeNmap("2001:db8::2", "02:42:CC:14:00:02", []string{"02:42:B:14:00:02", "02:42:C:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)

	ginkgo.It("MAC different than tried", func(done ginkgo.Done) {
		out := make(chan Any, 100)

		xml := func() ([]byte, error) {
			return []byte(nmapOut), nil
		}

		expected := &models.L2Connectivity{
			OutgoingNic:       "eth0",
			OutgoingIPAddress: "",
			RemoteIPAddress:   "2001:db8::2",
			RemoteMac:         "02:42:ac:12:00:02",
			Successful:        true,
		}

		analyzeNmap("2001:db8::2", "02:42:CC:10:00:02", []string{"02:42:AC:12:00:02", "02:42:AC:14:00:02"}, "eth0", out, xml)
		Expect(<-out).To(Equal(expected))
		close(done)
	}, 0.2)
})

func TestConnectivityCheck(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "connectivity check tests")
}
