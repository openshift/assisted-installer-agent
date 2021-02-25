package domain_resolution

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"

	"github.com/thoas/go-funk"
)

var _ = Describe("Domain resolution", func() {
	var (
		domainResolutionDependencies *MockDomainResolutionDependencies
		log                          *logrus.Logger
	)

	BeforeEach(func() {
		domainResolutionDependencies = &MockDomainResolutionDependencies{}
		log = logrus.New()

	})

	AfterEach(func() {
		domainResolutionDependencies.AssertExpectations(GinkgoT())
	})

	testResolution := func(testDomain string, ipv4Count, ipv6Count int, resolution models.DomainResolutionResponseDomain) {
		Expect(resolution.DomainName).To(PointTo(Equal(testDomain)))

		Expect(resolution.IPV4Addresses).To(HaveLen(ipv4Count))
		Expect(resolution.IPV6Addresses).To(HaveLen(ipv6Count))

		for i := 0; i < ipv4Count; i++ {
			Expect(resolution.IPV4Addresses[i].String()).To(Equal(getTestIpv4(i).String()))
		}

		for i := 0; i < ipv6Count; i++ {
			Expect(resolution.IPV6Addresses[i].String()).To(Equal(getTestIpv6(i).String()))
		}
	}

	Context("Resolve", func() {
		var (
			testDomain string
		)

		BeforeEach(func() {
			testDomain = "example.com"
		})

		for _, test := range []struct {
			ipv4Count int
			ipv6Count int
		}{
			{ipv4Count: 0, ipv6Count: 0}, {ipv4Count: 1, ipv6Count: 0}, {ipv4Count: 0, ipv6Count: 1},
			{ipv4Count: 1, ipv6Count: 1}, {ipv4Count: 2, ipv6Count: 2}, {ipv4Count: 3, ipv6Count: 3},
			{ipv4Count: 0, ipv6Count: 2}, {ipv4Count: 2, ipv6Count: 0}, {ipv4Count: 100, ipv6Count: 200},
		} {
			test := test

			It(fmt.Sprintf("Test with %d IPv4 addresses and %d IPv6 addresses",
				test.ipv4Count, test.ipv6Count), func() {
				domainResolutionDependencies.On("Resolve", testDomain).Return(
					generateResolution(test.ipv4Count, test.ipv6Count), nil).Once()
				resolution := handleDomainResolution(domainResolutionDependencies, log, testDomain)
				testResolution(testDomain, test.ipv4Count, test.ipv6Count, resolution)
			})
		}
	})

	Context("Run", func() {
		It("No domains", func() {
			// Request resolution for an empty list of domains
			request := models.DomainResolutionRequest{
				Domains: []*models.DomainResolutionRequestDomain{},
			}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			// Run tool
			stdout, stderr, exitCode := Run(string(b), domainResolutionDependencies, log)
			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			// Parse response
			var response models.DomainResolutionResponse
			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())

			// Make sure no domains appear in the response
			Expect(response.Resolutions).Should(BeEmpty())
		})

		It("Multiple domains", func() {
			// Request resolution for 3 arbitrary domains
			domains := []string{"example.com", "example.net", "example.org"}
			request := models.DomainResolutionRequest{
				Domains: []*models.DomainResolutionRequestDomain{
					{DomainName: &domains[0]},
					{DomainName: &domains[1]},
					{DomainName: &domains[2]},
				},
			}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())

			// Arbitrary amounts of ipv4 and ipv6 addresses returned for each domain
			ipv4Count := 3
			ipv6Count := 4

			// Prepare mock to return ip4Count IPv4 addresses and ip6Count IPv6 addresses for each domain
			for _, domain := range domains {
				domainResolutionDependencies.On("Resolve", domain).Return(
					generateResolution(ipv4Count, ipv6Count), nil).Once()
			}

			// Run tool
			stdout, stderr, exitCode := Run(string(b), domainResolutionDependencies, log)
			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			// Parse response
			var response models.DomainResolutionResponse
			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())

			// Make sure all requested domains appear in the response
			Expect(response.Resolutions).Should(HaveLen(len(domains)))

			// Check resolved IP addresses are the same as what the mock resolver returned for each domain
			for _, zip := range funk.Zip(domains, response.Resolutions) {
				requestedDomain, _ := zip.Element1.(string)
				resolution, _ := zip.Element2.(*models.DomainResolutionResponseDomain)

				Expect(resolution.DomainName).Should(PointTo(Equal(requestedDomain)))

				testResolution(requestedDomain, ipv4Count, ipv6Count, *resolution)
			}
		})
	})
})

func getTestIpv4(index int) net.IP {
	// TEST-NET-1
	return []byte{192, 0, 2, byte(index)}
}

func getTestIpv6(index int) net.IP {
	// 2001:db8::/32 doc/source code reserved block
	return []byte{
		32, 1, 13, 184,
		0, 0, 0, 0,
		0, 0, 0, 0,
		0, 0, 0, byte(index),
	}
}

func generateResolution(ipv4Count, ipv6Count int) []net.IP {
	var result []net.IP
	for i := 0; i < ipv4Count; i++ {
		result = append(result, getTestIpv4(i))
	}

	for i := 0; i < ipv6Count; i++ {
		result = append(result, getTestIpv6(i))
	}

	return result
}

func TestUnitests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Domain resoltuion unit tests")
}
