package inventory

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func newDependenciesMock() *util.MockIDependencies {
	return &util.MockIDependencies{}
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Inventory unit tests")
}
