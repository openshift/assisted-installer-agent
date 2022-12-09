package connectivity_check

import (
	"sort"

	"github.com/openshift/assisted-service/models"
)

type dryL2Checker struct{}

func (d *dryL2Checker) Features() Features {
	return RemoteIPFeature | RemoteMACFeature | OutgoingNicFeature
}

func (d *dryL2Checker) Check(attributes Attributes) ResultReporter {
	return newL2ResultReporter(&models.L2Connectivity{
		OutgoingNic:     attributes.OutgoingNIC,
		RemoteIPAddress: attributes.RemoteIPAddress,
		RemoteMac:       attributes.RemoteMACAddress,
		Successful:      true,
	})
}

func (d *dryL2Checker) Finalize(resultingHost *models.ConnectivityRemoteHost) {
	l2Key := func(c *models.L2Connectivity) string {
		return c.RemoteIPAddress + c.RemoteMac
	}

	sort.SliceStable(resultingHost.L2Connectivity,
		func(i, j int) bool {
			return l2Key(resultingHost.L2Connectivity[i]) < l2Key(resultingHost.L2Connectivity[j])
		})
}

type dryL3Checker struct{}

func (d *dryL3Checker) Features() Features {
	return RemoteIPFeature
}

func (d *dryL3Checker) Check(attributes Attributes) ResultReporter {
	return newL3ResultReporter(&models.L3Connectivity{
		RemoteIPAddress: attributes.RemoteIPAddress,
		Successful:      true,
	})
}

func (d *dryL3Checker) Finalize(*models.ConnectivityRemoteHost) {}
