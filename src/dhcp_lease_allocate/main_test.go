package dhcp_lease_allocate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newDependenciesMock() *MockDependencies {
	return &MockDependencies{}
}

func TestUnitests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DHCP unit tests")
}
