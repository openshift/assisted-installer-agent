package connectivity_check

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
)

var _ = Describe("get outgoing nics", func() {
	var (
		d *util.MockIDependencies
	)

	BeforeEach(func() {
		d = &util.MockIDependencies{}
	})

	AfterEach(func() {
		d.AssertExpectations(GinkgoT())
	})

	It("ipv4", func() {
		d.On("Interfaces").Return([]util.Interface{util.NewFilledMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagUp, []string{"1.2.3.4/24"},
			100, "physical")}, nil)
		outgoingNics := getOutgoingNics(nil, d)
		Expect(outgoingNics).To(HaveLen(1))
		Expect(outgoingNics[0].Name).To(Equal("eth0"))
		Expect(outgoingNics[0].HasIpv4Addresses).To(Equal(true))
		Expect(outgoingNics[0].Addresses[0].String()).To(Equal("1.2.3.4/24"))
	})
	It("ipv6", func() {
		d.On("Interfaces").Return([]util.Interface{util.NewFilledMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagUp, []string{"2003::10/64"},
			100, "physical")}, nil)
		outgoingNics := getOutgoingNics(nil, d)
		Expect(outgoingNics).To(HaveLen(1))
		Expect(outgoingNics[0].Name).To(Equal("eth0"))
		Expect(outgoingNics[0].HasIpv6Addresses).To(Equal(true))
		Expect(outgoingNics[0].Addresses[0].String()).To(Equal("2003::10/64"))
	})
	It("dual stack", func() {
		d.On("Interfaces").Return([]util.Interface{util.NewFilledMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagUp, []string{"1.2.3.4/24",
			"2003::10/64"}, 100, "physical")}, nil)
		outgoingNics := getOutgoingNics(nil, d)
		Expect(outgoingNics).To(HaveLen(1))
		Expect(outgoingNics[0].Name).To(Equal("eth0"))
		Expect(outgoingNics[0].HasIpv4Addresses).To(Equal(true))
		Expect(outgoingNics[0].HasIpv6Addresses).To(Equal(true))
		Expect(outgoingNics[0].Addresses[0].String()).To(Equal("1.2.3.4/24"))
		Expect(outgoingNics[0].Addresses[1].String()).To(Equal("2003::10/64"))
	})
	It("ipv4 link local", func() {
		d.On("Interfaces").Return([]util.Interface{util.NewFilledMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagUp, []string{"169.254.0.5/16"},
			100, "physical")}, nil)
		outgoingNics := getOutgoingNics(nil, d)
		Expect(outgoingNics).To(BeEmpty())
	})
	It("ipv6 link local", func() {
		d.On("Interfaces").Return([]util.Interface{util.NewFilledMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagUp, []string{"fe80::f/10"},
			100, "physical")}, nil)
		outgoingNics := getOutgoingNics(nil, d)
		Expect(outgoingNics).To(BeEmpty())
	})
	It("no addressexs", func() {
		d.On("Interfaces").Return([]util.Interface{util.NewFilledMockInterface(1500, "eth0", "f8:75:a4:a4:00:fe", net.FlagUp, []string{},
			100, "physical")}, nil)
		outgoingNics := getOutgoingNics(nil, d)
		Expect(outgoingNics).To(BeEmpty())
	})
})
