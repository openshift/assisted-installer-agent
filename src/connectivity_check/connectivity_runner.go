package connectivity_check

import (
	"net"
	"sort"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/openshift/assisted-service/models"
	"github.com/thoas/go-funk"
)

type Features int

const (
	RemoteIPFeature Features = 0x1 << iota
	OutgoingNicFeature
	RemoteMACFeature
)

type OutgoingNic struct {
	MTU              int
	Name             string
	HasIpv4Addresses bool
	HasIpv6Addresses bool
	Addresses        []net.Addr
}

// Attributes to be sent to a checker in order to perform a single checking operation
type Attributes struct {
	// The IP address of the remote host to check
	RemoteIPAddress string

	// The remote MAC address
	RemoteMACAddress string

	// Request to perform the test on a specific outgoing (local) NIC
	OutgoingNIC OutgoingNic

	// All the MAC addresses of the remote host
	RemoteMACAddresses []string
}

//go:generate mockery --name ResultReporter --inpackage
type ResultReporter interface {

	// Report the checking result on the checked host
	Report(resultingHost *models.ConnectivityRemoteHost) error
}

//go:generate mockery --name Checker --inpackage
type Checker interface {

	// Features supported by the current checker
	Features() Features

	// Check performs checking operation
	Check(attributes Attributes) ResultReporter

	// Finalize the check after all data has been collected.  Usually used for sorting or similar.
	Finalize(resultingHost *models.ConnectivityRemoteHost)
}

type connectivityRunner struct {
	checkers []Checker
}

type checkedHostResult struct {
	checkedHost *models.ConnectivityRemoteHost
	err         error
}

func spawnChecker(attributes Attributes, reporterChan chan ResultReporter, wg *sync.WaitGroup) func(c Checker) {
	return func(c Checker) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// There are cases that a checker does not emit any reporter.  In this case, nil is returned, and
			// it is skipped here.
			if reporter := c.Check(attributes); reporter != nil {
				reporterChan <- reporter
			}
		}()
	}
}

func (d *connectivityRunner) ProcessHost(checkHost *models.ConnectivityCheckHost, outgoingNics []OutgoingNic, resultChan chan checkedHostResult) {
	var (
		resultingHost models.ConnectivityRemoteHost
		wg            sync.WaitGroup
		reporterChan  = make(chan ResultReporter, 10)
		err           error
	)
	resultingHost.HostID = checkHost.HostID
	defer func() {
		resultChan <- checkedHostResult{
			checkedHost: &resultingHost,
			err:         err,
		}
	}()
	remoteMACAddresses := getAllMacAddresses(checkHost)
	for _, remoteNic := range checkHost.Nics {
		for _, cidr := range remoteNic.IPAddresses {
			ipAddress := getIPAddressFromCIDR(cidr)
			if ipAddress == "" {
				continue
			}
			for _, outgoingNIC := range outgoingNics {
				attributes := Attributes{
					RemoteIPAddress:    ipAddress,
					RemoteMACAddress:   remoteNic.Mac.String(),
					OutgoingNIC:        outgoingNIC,
					RemoteMACAddresses: remoteMACAddresses,
				}
				// Run the function returned by spawnChecker on all checkers that have the OutgoingNicFeature feature set.
				funk.ForEach(funk.Filter(d.checkers, func(c Checker) bool { return c.Features()&OutgoingNicFeature != 0 }), spawnChecker(attributes, reporterChan, &wg))
			}
			attributes := Attributes{
				RemoteIPAddress:    ipAddress,
				RemoteMACAddress:   remoteNic.Mac.String(),
				RemoteMACAddresses: remoteMACAddresses,
			}

			// Run the function returned by spawnChecker on all checkers that don't have the OutgoingNicFeature feature set.
			funk.ForEach(funk.Filter(d.checkers, func(c Checker) bool { return c.Features()&OutgoingNicFeature == 0 }), spawnChecker(attributes, reporterChan, &wg))
		}
	}
	go func() {
		wg.Wait()
		close(reporterChan)
	}()
	for r := range reporterChan {
		if e := r.Report(&resultingHost); e != nil {
			err = multierror.Append(err, e)
		}
	}
	funk.ForEach(d.checkers, func(c Checker) { c.Finalize(&resultingHost) })
}

func (d *connectivityRunner) Run(params models.ConnectivityCheckParams, outgoingNics []OutgoingNic) (models.ConnectivityReport, error) {
	var (
		ret models.ConnectivityReport
		err error
	)
	resultChan := make(chan checkedHostResult, len(params))
	for _, checkHost := range params {
		go d.ProcessHost(checkHost, outgoingNics, resultChan)
	}
	for i := 0; i != len(params); i++ {
		result := <-resultChan
		ret.RemoteHosts = append(ret.RemoteHosts, result.checkedHost)
		if result.err != nil {
			err = multierror.Append(err, result.err)
		}
	}
	sort.SliceStable(ret.RemoteHosts, func(i, j int) bool {
		return ret.RemoteHosts[i].HostID < ret.RemoteHosts[j].HostID
	})
	if err != nil {
		err = multierror.Flatten(err)
	}
	return ret, err
}
