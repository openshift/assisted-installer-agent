package inventory

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func newDependenciesMock() *util.MockIDependencies {
	d := &util.MockIDependencies{}
	mockGetGhwChrootRoot(d)
	return d
}

func mockGetGhwChrootRoot(dependencies *util.MockIDependencies) {
	dependencies.On("GetGhwChrootRoot").Return("/host").Maybe()
}

func TestSubsystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Inventory unit tests")
}
