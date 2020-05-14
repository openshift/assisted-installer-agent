package inventory

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)


func newDependenciesMock() *MockIDependencies {
	return &MockIDependencies{}
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Inventory unit tests")
}
