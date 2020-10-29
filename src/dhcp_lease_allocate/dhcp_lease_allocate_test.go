package dhcp_lease_allocate

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/vishvananda/netlink"
)

var _ = Describe("Lease allocate", func() {
	var (
		dependencies *MockDependencies
		leaser       *Leaser
		mac1         = "80:32:53:4f:cf:d6"
		mac2         = "52:54:00:09:de:93"
		apiLease     = `lease { api }`
		ingressLease = `lease { ingress }`
		leases       = []string{
			apiLease,
			ingressLease,
		}
		log *logrus.Logger
	)

	BeforeEach(func() {
		dependencies = newDependenciesMock()
		leaser = NewLeaser(dependencies)
		log = logrus.New()
		dependencies.On("MkdirAll", "/etc/keepalived", os.ModePerm).Return(nil)
	})

	AfterEach(func() {
		dependencies.AssertExpectations(GinkgoT())
	})

	asMac := func(mac string) *strfmt.MAC {
		ret := strfmt.MAC(mac)
		return &ret
	}

	createLeaseRequest := func(iface, apiMac, ingressMac, apiLease, ingressLease string) string {
		request := &models.DhcpAllocationRequest{
			APIVipLease:     apiLease,
			APIVipMac:       asMac(apiMac),
			IngressVipLease: ingressLease,
			IngressVipMac:   asMac(ingressMac),
			Interface:       swag.String(iface),
		}
		b, err := json.Marshal(request)
		Expect(err).ToNot(HaveOccurred())
		return string(b)
	}

	extractLeaseResponse := func(response string) *models.DhcpAllocationResponse {
		var ret models.DhcpAllocationResponse
		Expect(json.Unmarshal([]byte(response), &ret)).ToNot(HaveOccurred())
		return &ret
	}

	Context("Lease allocate", func() {
		It("Success - first time", func() {
			r := createLeaseRequest("eth0", mac1, mac2, "", "")
			for i, vipName := range []string{"api", "ingress"} {
				leaseFile := fmt.Sprintf("/etc/keepalived/lease-%s", vipName)
				dependencies.On("LinkByName", vipName).Return(&netlink.Macvlan{}, nil)
				dependencies.On("LeaseInterface", mock.Anything, "eth0", vipName, mock.Anything).Return(&net.Interface{Name: vipName}, nil)
				dependencies.On("Execute", "timeout", "28", "dhclient", "-v", "-H", vipName, "-sf", "/bin/true", "-lf", leaseFile, "--no-pid", "-1", vipName).Return("", "", 0)
				dependencies.On("GetLastLeaseFromFile", mock.Anything, leaseFile).Return(vipName, fmt.Sprintf("1.2.3.%d", i), nil)
				dependencies.On("ReadFile", leaseFile).Return([]byte(leases[i]), nil)
			}
			dependencies.On("LinkDel", mock.Anything).Return(nil).Times(2)
			stdout, stderr, exitCode := leaser.LeaseAllocate(r, log)
			Expect(exitCode).To(BeZero())
			Expect(stdout).ToNot(BeEmpty())
			response := extractLeaseResponse(stdout)
			Expect(stderr).To(BeEmpty())
			Expect(response.APIVipAddress.String()).To(Equal("1.2.3.0"))
			Expect(response.IngressVipAddress.String()).To(Equal("1.2.3.1"))
			Expect(response.APIVipLease).To(Equal(apiLease))
			Expect(response.IngressVipLease).To(Equal(ingressLease))
		})
		It("Success - second time", func() {
			r := createLeaseRequest("eth0", mac1, mac2, apiLease, ingressLease)
			for i, vipName := range []string{"api", "ingress"} {
				leaseFile := fmt.Sprintf("/etc/keepalived/lease-%s", vipName)
				dependencies.On("LinkByName", vipName).Return(&netlink.Macvlan{}, nil)
				dependencies.On("LeaseInterface", mock.Anything, "eth0", vipName, mock.Anything).Return(&net.Interface{Name: vipName}, nil)
				dependencies.On("Execute", "timeout", "28", "dhclient", "-v", "-H", vipName, "-sf", "/bin/true", "-lf", leaseFile, "--no-pid", "-1", vipName).Return("", "", 0)
				dependencies.On("GetLastLeaseFromFile", mock.Anything, leaseFile).Return(vipName, fmt.Sprintf("1.2.3.%d", i), nil)
				dependencies.On("ReadFile", leaseFile).Return([]byte(leases[i]), nil)
				dependencies.On("WriteFile", leaseFile, []byte(leases[i]), os.FileMode(0o644)).Return(nil)
			}
			dependencies.On("LinkDel", mock.Anything).Return(nil).Times(2)
			stdout, stderr, exitCode := leaser.LeaseAllocate(r, log)
			Expect(exitCode).To(BeZero())
			Expect(stdout).ToNot(BeEmpty())
			response := extractLeaseResponse(stdout)
			Expect(stderr).To(BeEmpty())
			Expect(response.APIVipAddress.String()).To(Equal("1.2.3.0"))
			Expect(response.IngressVipAddress.String()).To(Equal("1.2.3.1"))
			Expect(response.APIVipLease).To(Equal(apiLease))
			Expect(response.IngressVipLease).To(Equal(ingressLease))
		})
		It("Error reading lease file", func() {
			r := createLeaseRequest("eth0", mac1, mac2, apiLease, ingressLease)
			vipName := "api"
			leaseFile := fmt.Sprintf("/etc/keepalived/lease-%s", vipName)
			dependencies.On("LinkByName", vipName).Return(&netlink.Macvlan{}, nil)
			dependencies.On("LeaseInterface", mock.Anything, "eth0", vipName, mock.Anything).Return(&net.Interface{Name: vipName}, nil)
			dependencies.On("Execute", "timeout", "28", "dhclient", "-v", "-H", vipName, "-sf", "/bin/true", "-lf", leaseFile, "--no-pid", "-1", vipName).Return("", "", 0)
			dependencies.On("GetLastLeaseFromFile", mock.Anything, leaseFile).Return(vipName, "1.2.3.0", nil)
			dependencies.On("ReadFile", leaseFile).Return(nil, errors.New("Blah"))
			dependencies.On("WriteFile", leaseFile, []byte(apiLease), os.FileMode(0o644)).Return(nil)
			dependencies.On("LinkDel", mock.Anything).Return(nil)
			stdout, stderr, exitCode := leaser.LeaseAllocate(r, log)
			Expect(exitCode).ToNot(BeZero())
			Expect(stdout).To(BeEmpty())
			Expect(stderr).ToNot(BeEmpty())
		})
		It("Error writing lease file", func() {
			r := createLeaseRequest("eth0", mac1, mac2, apiLease, ingressLease)
			vipName := "api"
			leaseFile := fmt.Sprintf("/etc/keepalived/lease-%s", vipName)
			dependencies.On("LinkByName", vipName).Return(&netlink.Macvlan{}, nil)
			dependencies.On("LeaseInterface", mock.Anything, "eth0", vipName, mock.Anything).Return(&net.Interface{Name: vipName}, nil)
			dependencies.On("WriteFile", leaseFile, []byte(apiLease), os.FileMode(0o644)).Return(errors.New("Blah"))

			dependencies.On("LinkDel", mock.Anything).Return(nil)
			stdout, stderr, exitCode := leaser.LeaseAllocate(r, log)
			Expect(exitCode).ToNot(BeZero())
			Expect(stdout).To(BeEmpty())
			Expect(stderr).ToNot(BeEmpty())
		})

	})
})
