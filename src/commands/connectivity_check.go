package commands

import (
	"encoding/json"
	"encoding/xml"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-installer-agent/src/util/nmap"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func getOutgoingNics() []string {
	ret := make([]string, 0)
	d := util.NewDependencies("")
	interfaces, err := d.Interfaces()
	if err != nil {
		log.WithError(err).Warnf("Get outgoing nics")
		return nil
	}
	for _, intf := range interfaces {
		if !(intf.IsPhysical() || intf.IsBonding() || intf.IsVlan()) {
			continue
		}
		ret = append(ret, intf.Name())
	}
	return ret
}

func getOutgoingAddresses(nics []*models.ConnectivityCheckNic) []string {
	ret := make([]string, 0)
	for _, nic := range nics {
		for _, cidr := range nic.IPAddresses {
			address := getIPAddressFromCIDR(cidr)
			if address != "" {
				ret = append(ret, address)
			}
		}
	}
	return ret
}

func getIPAddressFromCIDR(cidr string) string {
	parts := strings.Split(cidr, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

type any interface{}

type done struct{}

func sendDone(ch chan any) {
	ch <- done{}
}

const pingCount string = "10"

func l3CheckAddressOnNic(address string, outgoingNic string, innerChan chan *models.L3Connectivity, conCheck connectivityCmd) {
	ret := &models.L3Connectivity{
		OutgoingNic:     outgoingNic,
		RemoteIPAddress: address,
	}

	if config.GlobalDryRunConfig.DryRunEnabled {
		ret.Successful = true
		ret.PacketLossPercentage = 0.0
		ret.AverageRTTMs = 0.0
		innerChan <- ret
		return
	}

	b, err := conCheck.command("ping", []string{"-c", pingCount, "-W", "3", "-q", "-I", outgoingNic, address})
	if err != nil {
		log.Errorf("Error running ping to %s on interface %s: %s", address, outgoingNic, err.Error())
		innerChan <- ret
		return
	}
	err = parsePingCmd(ret, string(b))
	if err != nil {
		log.Error(err)
		innerChan <- ret
		return
	}
	ret.Successful = true
	innerChan <- ret
}

func regexMatchFor(regex, line string) ([]string, error) {
	r := regexp.MustCompile(regex)
	p := r.FindStringSubmatch(line)
	if len(p) < 2 {
		return nil, errors.Errorf("unable to parse %s with regex %s", line, regex)
	}
	return p, nil
}

func parsePingCmd(conn *models.L3Connectivity, cmdOutput string) error {
	if len(cmdOutput) == 0 {
		return errors.Errorf("Missing output for ping or invalid output:\n%s", cmdOutput)
	}
	parts, err := regexMatchFor(`[\d]+ packets transmitted, [\d]+ received, (([\d]*[.])?[\d]+)% packet loss, time [\d]+ms`, cmdOutput)
	if err != nil {
		return errors.Errorf("Unable to retrieve packet loss percentage: %s", err)
	}
	conn.PacketLossPercentage, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return errors.Errorf("Error while trying to convert value for packet loss '%s': %s", parts[1], err)
	}
	parts, err = regexMatchFor(`rtt min\/avg\/max\/mdev = .*\/([^\/]+)\/.*\/.* ms`, cmdOutput)
	if err != nil {
		return errors.Errorf("Unable to retrieve the average RTT for ping: %s", err)
	}
	conn.AverageRTTMs, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return errors.Errorf("Error while trying to convert value for packet loss %s: %s", parts[1], err)
	}
	return nil
}

func l3CheckConnectivity(addresses []string, dataCh chan any, conCheck connectivityCmd) {

	defer sendDone(dataCh)
	wg := sync.WaitGroup{}
	wg.Add(len(addresses))
	for _, address := range addresses {
		go l3CheckAddress(address, conCheck.getOutgoingNICs(), dataCh, &wg, conCheck)
	}
	wg.Wait()
}

func l3CheckAddress(address string, outgoingNics []string, dataCh chan any, wg *sync.WaitGroup, conCheck connectivityCmd) {
	defer wg.Done()
	innerChan := make(chan *models.L3Connectivity)
	for _, nic := range outgoingNics {
		go l3CheckAddressOnNic(address, nic, innerChan, conCheck)
	}
	successful := false
	for i := 0; i != len(outgoingNics); i++ {
		ret := <-innerChan
		if ret.Successful {
			dataCh <- ret
			successful = true
		}
	}
	if !successful {
		ret := &models.L3Connectivity{
			RemoteIPAddress: address,
		}
		dataCh <- ret
	}
}

func macInDstMacs(mac string, allDstMACs []string) bool {
	for _, dstMAC := range allDstMACs {
		if strings.EqualFold(mac, dstMAC) {
			return true
		}
	}
	return false
}

func l2CheckAddressOnNic(dstAddr string, dstMAC string, allDstMACs []string, srcNIC string, dataCh chan any, conCheck connectivityCmd) {
	defer sendDone(dataCh)
	if util.IsIPv4Addr(dstAddr) {
		l2IPv4Cmd(dstAddr, dstMAC, allDstMACs, srcNIC, dataCh, conCheck)
	} else {
		analyzeNmap(dstAddr, dstMAC, allDstMACs, srcNIC, dataCh, conCheck)
	}

}

func analyzeNmap(dstAddr string, dstMAC string, allDstMACs []string, srcNIC string, dataCh chan any, conCheck connectivityCmd) {

	ret := &models.L2Connectivity{
		OutgoingNic:     srcNIC,
		RemoteIPAddress: dstAddr,
	}

	if config.GlobalDryRunConfig.DryRunEnabled {
		ret.Successful = true
		dataCh <- ret
		return
	}

	out, err := conCheck.command("nmap", []string{"-6", "-sn", "-n", "-oX", "-", "-e", srcNIC, dstAddr})
	if err != nil {
		log.WithError(err).Warn("nmap command failed")
		dataCh <- ret
		return
	}

	var nmaprun nmap.Nmaprun
	if err := xml.Unmarshal(out, &nmaprun); err != nil {
		log.WithError(err).Warn("Failed to un-marshal nmap XML")
		dataCh <- ret
		return
	}

	for _, h := range nmaprun.Hosts {
		if h.Status.State != "up" {
			continue
		}
		for _, a := range h.Addresses {
			if a.AddrType != "mac" {
				continue
			}
			remoteMAC := strings.ToLower(a.Addr)
			ret.RemoteMac = remoteMAC
			ret.Successful = macInDstMacs(remoteMAC, allDstMACs)
			if !ret.Successful {
				log.Warnf("Unexpected MAC address for nmap %s on NIC %s: %s", dstAddr, srcNIC, remoteMAC)
			} else if strings.ToLower(dstMAC) != remoteMAC {
				log.Infof("Received remote MAC %s different then expected MAC %s", remoteMAC, dstMAC)
			}

			dataCh <- ret
			return
		}
	}
	dataCh <- ret
}

func l2CheckAddress(dstAddr string, dstMAC string, allDstMACs []string, dataCh chan any, wg *sync.WaitGroup, conCheck connectivityCmd) {
	defer wg.Done()
	innerChan := make(chan any)
	for _, srcNIC := range conCheck.getOutgoingNICs() {
		go l2CheckAddressOnNic(dstAddr, dstMAC, allDstMACs, srcNIC, innerChan, conCheck)
	}
	received := false
	for numDone := 0; numDone != len(conCheck.getOutgoingNICs()); {
		iret := <-innerChan
		switch ret := iret.(type) {
		case *models.L2Connectivity:
			received = true
			dataCh <- ret
		case done:
			numDone++
		}
	}
	if !received {
		dataCh <- &models.L2Connectivity{
			RemoteIPAddress: dstAddr,
		}
	}
}

func l2CheckConnectivity(dataCh chan any, conCheck connectivityCmd) {
	defer sendDone(dataCh)
	allDstMACs := make([]string, len(conCheck.getHost().Nics))
	for i, destNic := range conCheck.getHost().Nics {
		allDstMACs[i] = destNic.Mac.String()
	}
	numAddresses := 0
	for _, destNic := range conCheck.getHost().Nics {
		numAddresses += len(destNic.IPAddresses)
	}
	wg := sync.WaitGroup{}
	wg.Add(numAddresses)
	for _, destNic := range conCheck.getHost().Nics {
		for _, address := range destNic.IPAddresses {
			go l2CheckAddress(address, destNic.Mac.String(), allDstMACs, dataCh, &wg, conCheck)
		}
	}
	wg.Wait()
}

func l2IPv4Cmd(dstAddr string, dstMAC string, allDstMACs []string, srcNIC string, dataCh chan any, conCheck connectivityCmd) {
	ret := &models.L2Connectivity{
		OutgoingNic:     srcNIC,
		RemoteIPAddress: dstAddr,
	}

	if config.GlobalDryRunConfig.DryRunEnabled {
		ret.Successful = true
		dataCh <- ret
		return
	}

	bytes, err := conCheck.command("arping", []string{"-c", "1", "-w", "2", "-I", srcNIC, dstAddr})
	if err != nil {
		log.Errorf("Error while processing 'arping' command: %s", err)
		dataCh <- ret
		return
	}
	lines := strings.Split(string(bytes), "\n")
	if len(lines) == 0 {
		log.Warnf("Missing output for arping")
		dataCh <- ret
		return
	}

	hRgegex := regexp.MustCompile("^ARPING ([^ ]+) from ([^ ]+) ([^ ]+)$")
	parts := hRgegex.FindStringSubmatch(lines[0])
	if len(parts) != 4 {
		log.Warnf("Wrong format for header line: %s", lines[0])
		dataCh <- ret
		return
	}

	ret.OutgoingIPAddress = parts[2]
	rRegexp := regexp.MustCompile(`^Unicast reply from ([^ ]+) \[([^]]+)\]  [^ ]+$`)
	for _, line := range lines[1:] {
		parts = rRegexp.FindStringSubmatch(line)
		if len(parts) != 3 {
			continue
		}
		remoteMAC := strings.ToLower(parts[2])
		ret.RemoteMac = remoteMAC
		ret.Successful = macInDstMacs(remoteMAC, allDstMACs)
		if !ret.Successful {
			log.Warnf("Unexpected mac address for arping %s on nic %s: %s", dstAddr, srcNIC, remoteMAC)
		}
		if strings.ToLower(dstMAC) != remoteMAC {
			log.Infof("Received remote mac %s different then expected mac %s", remoteMAC, dstMAC)
		}
		dataCh <- ret
	}
}

func checkHost(conCheck connectivityCmd, outCh chan *models.ConnectivityRemoteHost) {
	ret := &models.ConnectivityRemoteHost{
		HostID:         conCheck.getHost().HostID,
		L2Connectivity: []*models.L2Connectivity{},
		L3Connectivity: []*models.L3Connectivity{},
	}
	addresses := getOutgoingAddresses(conCheck.getHost().Nics)
	dataCh := make(chan any)
	go l3CheckConnectivity(addresses, dataCh, conCheck)
	go l2CheckConnectivity(dataCh, conCheck)
	for numDone := 0; numDone != 2; {
		v := <-dataCh
		switch value := v.(type) {
		case *models.L3Connectivity:
			ret.L3Connectivity = append(ret.L3Connectivity, value)
		case *models.L2Connectivity:
			ret.L2Connectivity = append(ret.L2Connectivity, value)
		case done:
			numDone++
		}

	}
	outCh <- ret
}

func canonizeResult(connectivityReport *models.ConnectivityReport) {
	for _, h := range connectivityReport.RemoteHosts {
		l3 := h.L3Connectivity
		sort.Slice(l3, func(i, j int) bool {
			if l3[i].RemoteIPAddress != l3[j].RemoteIPAddress {
				return l3[i].RemoteIPAddress < l3[j].RemoteIPAddress
			}
			return l3[i].OutgoingNic < l3[j].OutgoingNic
		})
		l2 := h.L2Connectivity
		sort.Slice(l2, func(i, j int) bool {
			if l2[i].RemoteIPAddress != l2[j].RemoteIPAddress {
				return l2[i].RemoteIPAddress < l2[j].RemoteIPAddress
			}
			return l2[i].OutgoingNic < l2[j].OutgoingNic
		})
	}
	sort.Slice(connectivityReport.RemoteHosts, func(i, j int) bool {
		return connectivityReport.RemoteHosts[i].HostID.String() < connectivityReport.RemoteHosts[j].HostID.String()
	})
}

func ConnectivityCheck(_ string, args ...string) (stdout string, stderr string, exitCode int) {
	if len(args) != 1 {
		return "", "Expecting exactly 1 argument for connectivity command", -1
	}
	params := models.ConnectivityCheckParams{}
	err := json.Unmarshal([]byte(args[0]), &params)
	if err != nil {
		log.Warnf("Error unmarshalling json %s: %s", args[0], err.Error())
		return "", err.Error(), -1
	}
	nics := getOutgoingNics()
	hostChan := make(chan *models.ConnectivityRemoteHost)
	for _, host := range params {
		h := hostChecker{outgoingNICS: nics, host: host}
		go checkHost(h, hostChan)
	}
	ret := models.ConnectivityReport{RemoteHosts: []*models.ConnectivityRemoteHost{}}
	for i := 0; i != len(params); i++ {
		ret.RemoteHosts = append(ret.RemoteHosts, <-hostChan)
	}
	canonizeResult(&ret)
	bytes, err := json.Marshal(&ret)
	if err != nil {
		log.Warnf("Could not marshal json: %s", err.Error())
		return "", err.Error(), -1
	}
	return string(bytes), "", 0
}

type hostChecker struct {
	host         *models.ConnectivityCheckHost
	outgoingNICS []string
}

type connectivityCmd interface {
	command(name string, args []string) ([]byte, error)
	getHost() *models.ConnectivityCheckHost
	getOutgoingNICs() []string
}

func (hc hostChecker) getHost() *models.ConnectivityCheckHost {
	return hc.host
}

func (hc hostChecker) getOutgoingNICs() []string {
	return hc.outgoingNICS
}

func (hc hostChecker) command(name string, args []string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}
