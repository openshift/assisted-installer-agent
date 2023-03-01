package connectivity_check

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/stretchr/testify/mock"
	"github.com/thoas/go-funk"
)

var _ = Describe("connectivity dispatcher", func() {
	var (
		mockCheckers []*MockChecker
		mockReporter *MockResultReporter
		nics         []OutgoingNic
		params       models.ConnectivityCheckParams
	)
	BeforeEach(func() {
		nics = []OutgoingNic{
			{
				Name: "eth0",
			},
			{
				Name: "eth1",
			},
		}
		params = models.ConnectivityCheckParams{
			{
				Nics: []*models.ConnectivityCheckNic{
					{
						IPAddresses: []string{
							"10.1.2.3",
							"de::1",
						},
						Mac:  "f8:75:a4:a4:00:fe",
						Name: "eth0",
					},
					{
						IPAddresses: []string{
							"10.1.2.4",
							"de::2",
						},
						Mac:  "f8:75:a4:a4:00:ff",
						Name: "eth1",
					},
				},
				HostID: "ab21fd4e-5e28-43e7-bc9b-03c2b75bcf3c",
			},
		}
	})
	AfterEach(func() {
		funk.ForEach(mockCheckers, func(m *MockChecker) { m.AssertExpectations(GinkgoT()) })
		if mockReporter != nil {
			mockReporter.AssertExpectations(GinkgoT())
		}

		mockCheckers = nil
		mockReporter = nil
	})
	newMockChecker := func(f Features) *MockChecker {
		m := &MockChecker{}
		m.On("Features").Return(f)
		m.On("Finalize", mock.AnythingOfType("*models.ConnectivityRemoteHost")).Return().Once()
		mockCheckers = append(mockCheckers, m)
		return m
	}

	toCheckers := func() []Checker {
		var ret []Checker
		for _, m := range mockCheckers {
			ret = append(ret, m)
		}
		return ret
	}

	It("single successful checker - no reporters", func() {
		m1 := newMockChecker(RemoteIPFeature | RemoteMACFeature)
		m1.On("Check", mock.AnythingOfType("connectivity_check.Attributes")).Return(nil).Times(4)
		d := &connectivityRunner{checkers: toCheckers()}
		ret, err := d.Run(params, nics)
		Expect(err).ToNot(HaveOccurred())
		Expect(ret.RemoteHosts).To(HaveLen(1))
	})
	It("single successful checker - no reporters - multiple nics", func() {
		m1 := newMockChecker(RemoteIPFeature | RemoteMACFeature | OutgoingNicFeature)
		m1.On("Check", mock.AnythingOfType("connectivity_check.Attributes")).Return(nil).Times(8)
		d := &connectivityRunner{checkers: toCheckers()}
		ret, err := d.Run(params, nics)
		Expect(err).ToNot(HaveOccurred())
		Expect(ret.RemoteHosts).To(HaveLen(1))
	})
	It("single successful checker - emit reporters", func() {
		m1 := newMockChecker(RemoteIPFeature | RemoteMACFeature)
		mockReporter = &MockResultReporter{}
		mockReporter.On("Report", mock.AnythingOfType("*models.ConnectivityRemoteHost")).Return(nil).Times(4)
		m1.On("Check", mock.AnythingOfType("connectivity_check.Attributes")).Return(mockReporter).
			Run(func(args mock.Arguments) {
				attributes, ok := args.Get(0).(Attributes)
				Expect(ok).To(BeTrue())
				Expect(attributes.OutgoingNIC).To(Equal(OutgoingNic{}))
				Expect(attributes.RemoteIPAddress).To(BeElementOf([]string{"10.1.2.3", "10.1.2.4", "de::1", "de::2"}))
				Expect(attributes.RemoteMACAddress).To(BeElementOf([]string{"f8:75:a4:a4:00:fe", "f8:75:a4:a4:00:ff"}))
				Expect(attributes.RemoteMACAddresses).To(ConsistOf([]string{"f8:75:a4:a4:00:fe", "f8:75:a4:a4:00:ff"}))
			}).
			Times(4)
		d := &connectivityRunner{checkers: toCheckers()}
		ret, err := d.Run(params, nics)
		Expect(err).ToNot(HaveOccurred())
		Expect(ret.RemoteHosts).To(HaveLen(1))
	})
	It("single failed checker - emit error reporter", func() {
		m1 := newMockChecker(RemoteIPFeature | RemoteMACFeature)
		mockReporter = &MockResultReporter{}
		mockReporter.On("Report", mock.AnythingOfType("*models.ConnectivityRemoteHost")).Return(errors.New("this is an error")).Times(4)
		m1.On("Check", mock.AnythingOfType("connectivity_check.Attributes")).Return(mockReporter).Times(4)
		d := &connectivityRunner{checkers: toCheckers()}
		ret, err := d.Run(params, nics)
		Expect(err).To(HaveOccurred())
		Expect(ret.RemoteHosts).To(HaveLen(1))
	})
	It("multiple successful checkers", func() {
		m1 := newMockChecker(RemoteIPFeature | RemoteMACFeature | OutgoingNicFeature)
		mockReporter = &MockResultReporter{}
		mockReporter.On("Report", mock.AnythingOfType("*models.ConnectivityRemoteHost")).Return(nil).Times(8)
		m1.On("Check", mock.AnythingOfType("connectivity_check.Attributes")).Return(mockReporter).
			Run(func(args mock.Arguments) {
				attributes, ok := args.Get(0).(Attributes)
				Expect(ok).To(BeTrue())
				Expect(attributes.OutgoingNIC.Name).To(BeElementOf([]string{"eth0", "eth1"}))
				Expect(attributes.RemoteIPAddress).To(BeElementOf([]string{"10.1.2.3", "10.1.2.4", "de::1", "de::2"}))
				Expect(attributes.RemoteMACAddress).To(BeElementOf([]string{"f8:75:a4:a4:00:fe", "f8:75:a4:a4:00:ff"}))
				Expect(attributes.RemoteMACAddresses).To(ConsistOf([]string{"f8:75:a4:a4:00:fe", "f8:75:a4:a4:00:ff"}))
			}).
			Times(8)
		m2 := newMockChecker(RemoteIPFeature | RemoteMACFeature | OutgoingNicFeature)
		m2.On("Check", mock.AnythingOfType("connectivity_check.Attributes")).Return(nil).Times(8)
		m3 := newMockChecker(RemoteIPFeature | RemoteMACFeature)
		m3.On("Check", mock.AnythingOfType("connectivity_check.Attributes")).Return(nil).Times(4)
		d := &connectivityRunner{checkers: toCheckers()}
		ret, err := d.Run(params, nics)
		Expect(err).ToNot(HaveOccurred())
		Expect(ret.RemoteHosts).To(HaveLen(1))
	})
})
