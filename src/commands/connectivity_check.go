package commands

import (
	"encoding/json"
	"encoding/xml"
	"net"
	"os/exec"
	"regexp"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-installer-agent/src/util/nmap"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type Done struct{}

type Any interface{}

func getOutgoingNics() []string {
	ret := make([]string, 0)
	r := regexp.MustCompile("^(?:eth|ens|eno|enp|wlp)\\d")
	interfaces, err := net.Interfaces()
	if err != nil {
		log.WithError(err).Warnf("Get outgoing nics")
		return nil
	}
	for _, intf := range interfaces {
		if r.MatchString(intf.Name) {
			ret = append(ret, intf.Name)
		}
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

func sendDone(ch chan Any) {
	ch <- Done{}
}

func l3CheckAddressOnNic(address string, outgoingNic string, l3chan chan *models.L3Connectivity) {
	ret := &models.L3Connectivity{
		OutgoingNic:     outgoingNic,
		RemoteIPAddress: address,
		Successful:      false,
	}
	cmd := exec.Command("ping", "-c", "2", "-W", "3", "-I", outgoingNic, address)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Infof("Error running ping to %s on interface %s: %s", address, outgoingNic, err.Error())
		ret.Successful = false
	} else {
		ret.Successful = true
	}
	l3chan <- ret
}

func l3CheckAddress(address string, outgoingNics []string, l3chan, doneChan chan Any) {
	defer sendDone(doneChan)
	innerChan := make(chan *models.L3Connectivity, 1000)
	for _, nic := range outgoingNics {
		go l3CheckAddressOnNic(address, nic, innerChan)
	}
	successful := false
	for i := 0; i != len(outgoingNics); i++ {
		ret := <-innerChan
		if ret.Successful {
			l3chan <- ret
			successful = true
		}
	}
	if !successful {
		ret := &models.L3Connectivity{
			RemoteIPAddress: address,
			Successful:      false,
		}
		l3chan <- ret
	}
}

func l3CheckConnectivity(addresses []string, outgoingNics []string, l3chan chan Any) {
	defer sendDone(l3chan)
	doneChan := make(chan Any)
	for _, address := range addresses {
		go l3CheckAddress(address, outgoingNics, l3chan, doneChan)
	}
	for i := 0; i != len(addresses); i++ {
		<-doneChan
	}
}

func macInDstMacs(mac string, allDstMacs []string) bool {
	for _, dstMac := range allDstMacs {
		if strings.ToLower(mac) == strings.ToLower(dstMac) {
			return true
		}
	}
	return false
}

func l2CheckAddressOnNic(dstAddr string, dstMac string, allDstMacs []string, srcNic string, l2chan chan Any) {
	defer sendDone(l2chan)
	if util.IsIPv4Addr(dstAddr) {
		runArping(dstAddr, dstMac, allDstMacs, srcNic, l2chan)
	} else {
		cmd := exec.Command("nmap", "-6", "-sn", "-n", "-oX", "-", "-e", srcNic, dstAddr)
		analyzeNmap(dstAddr, dstMac, allDstMacs, srcNic, l2chan, cmd.Output)
	}
}

func runArping(dstAddr string, dstMac string, allDstMacs []string, srcNic string, l2chan chan Any) {

	ret := &models.L2Connectivity{
		OutgoingNic:     srcNic,
		RemoteIPAddress: dstAddr,
		RemoteMac:       "",
		Successful:      false,
	}
	cmd := exec.Command("arping", "-c", "1", "-w", "2", "-I", srcNic, dstAddr)
	bytes, _ := cmd.CombinedOutput()
	lines := strings.Split(string(bytes), "\n")
	if len(lines) == 0 {
		log.Warnf("Missing output for arping")
		l2chan <- ret
		return
	}

	hRgegex := regexp.MustCompile("^ARPING ([^ ]+) from ([^ ]+) ([^ ]+)$")
	parts := hRgegex.FindStringSubmatch(lines[0])
	if len(parts) != 4 {
		log.Warnf("Wrong format for header line: %s", lines[0])
		l2chan <- ret
		return
	}

	ret.OutgoingIPAddress = parts[2]
	rRegexp := regexp.MustCompile("^Unicast reply from ([^ ]+) \\[([^]]+)\\]  [^ ]+$")
	for _, line := range lines[1:] {
		parts = rRegexp.FindStringSubmatch(line)
		if len(parts) != 3 {
			continue
		}
		remoteMac := strings.ToLower(parts[2])
		ret.RemoteMac = remoteMac
		ret.Successful = macInDstMacs(remoteMac, allDstMacs)
		if !ret.Successful {
			log.Warnf("Unexpected mac address for arping %s on nic %s: %s", dstAddr, srcNic, remoteMac)
		} else if strings.ToLower(dstMac) != remoteMac {
			log.Infof("Received remote mac %s different then expected mac %s", remoteMac, dstMac)
		}
		l2chan <- ret
	}
}

func analyzeNmap(dstAddr string, dstMac string, allDstMacs []string, srcNic string, l2chan chan Any, output func() ([]byte, error)) {

	ret := &models.L2Connectivity{
		OutgoingNic:       srcNic,
		OutgoingIPAddress: "",
		RemoteIPAddress:   dstAddr,
		RemoteMac:         "",
		Successful:        false,
	}

	out, err := output()
	if err != nil {
		log.WithError(err).Warn("nmap command failed")
		l2chan <- ret
		return
	}

	var nmaprun nmap.Nmaprun
	if err := xml.Unmarshal([]byte(out), &nmaprun); err != nil {
		log.WithError(err).Warn("Failed to un-marshal nmap XML")
		l2chan <- ret
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

			remoteMac := strings.ToLower(a.Addr)
			ret.RemoteMac = remoteMac
			ret.Successful = macInDstMacs(remoteMac, allDstMacs)
			if !ret.Successful {
				log.Warnf("Unexpected MAC address for nmap %s on NIC %s: %s", dstAddr, srcNic, remoteMac)
			} else if strings.ToLower(dstMac) != remoteMac {
				log.Infof("Received remote MAC %s different then expected MAC %s", remoteMac, dstMac)
			}

			l2chan <- ret
			return
		}
	}

	l2chan <- ret
}

func l2CheckAddress(dstAddr string, dstMac string, allDstMacs, sourceNics []string, l2chan chan Any, l2DoneChan chan Any) {
	defer sendDone(l2DoneChan)
	innerChan := make(chan Any, 1000)
	for _, srcNic := range sourceNics {
		go l2CheckAddressOnNic(dstAddr, dstMac, allDstMacs, srcNic, innerChan)
	}
	received := false
	for numDone := 0; numDone != len(sourceNics); {
		iret := <-innerChan
		switch ret := iret.(type) {
		case *models.L2Connectivity:
			received = true
			l2chan <- ret
		case Done:
			numDone++
		}
	}
	if !received {
		ret := &models.L2Connectivity{
			OutgoingNic:       "",
			OutgoingIPAddress: "",
			RemoteIPAddress:   dstAddr,
			RemoteMac:         "",
			Successful:        false,
		}
		l2chan <- ret
	}
}

func l2CheckConnectivity(destinationNics []*models.ConnectivityCheckNic, sourceNics []string, l2chan chan Any) {
	defer sendDone(l2chan)
	doneChan := make(chan Any)
	allDstMacs := make([]string, 0)
	for _, destNic := range destinationNics {
		allDstMacs = append(allDstMacs, destNic.Mac)
	}
	numAddresses := 0
	for _, destNic := range destinationNics {
		for _, address := range destNic.IPAddresses {
			numAddresses++
			go l2CheckAddress(address, destNic.Mac, allDstMacs, sourceNics, l2chan, doneChan)
		}
	}
	for i := 0; i != numAddresses; i++ {
		<-doneChan
	}
}

func checkHost(outgoingNics []string, host *models.ConnectivityCheckHost, hostChan chan *models.ConnectivityRemoteHost) {
	ret := &models.ConnectivityRemoteHost{
		HostID:         host.HostID,
		L2Connectivity: make([]*models.L2Connectivity, 0),
		L3Connectivity: make([]*models.L3Connectivity, 0),
	}
	r := regexp.MustCompile("^(?:eth|ens|eno|enp)\\d")
	checkedNics := make([]*models.ConnectivityCheckNic, 0)
	for _, nic := range host.Nics {
		if r.MatchString(nic.Name) {
			checkedNics = append(checkedNics, nic)
		}
	}
	addresses := getOutgoingAddresses(checkedNics)
	ch := make(chan Any, 1000)
	go l3CheckConnectivity(addresses, outgoingNics, ch)
	go l2CheckConnectivity(checkedNics, outgoingNics, ch)
	for numDone := 0; numDone != 2; {
		iret := <-ch
		switch value := iret.(type) {
		case *models.L2Connectivity:
			ret.L2Connectivity = append(ret.L2Connectivity, value)
		case *models.L3Connectivity:
			ret.L3Connectivity = append(ret.L3Connectivity, value)
		case Done:
			numDone++
		}
	}
	hostChan <- ret
}

func ConnectivityCheck(_ string, args ...string) (stdout string, stderr string, exitCode int) {
	if len(args) != 1 {
		return "", "Expecting exactly 1 argument for connectivity command", -1
	}
	params := make(models.ConnectivityCheckParams, 0)
	err := json.Unmarshal([]byte(args[0]), &params)
	if err != nil {
		log.Warnf("Error unmarshalling json %s: %s", args[0], err.Error())
		return "", err.Error(), -1
	}
	nics := getOutgoingNics()
	hostChan := make(chan *models.ConnectivityRemoteHost, 0)
	for _, host := range params {
		go checkHost(nics, host, hostChan)
	}
	ret := models.ConnectivityReport{RemoteHosts: make([]*models.ConnectivityRemoteHost, 0)}
	for i := 0; i != len(params); i++ {
		ret.RemoteHosts = append(ret.RemoteHosts, <-hostChan)
	}
	bytes, err := json.Marshal(&ret)
	if err != nil {
		log.Warnf("Could not marshal json: %s", err.Error())
		return "", err.Error(), -1
	}
	return string(bytes), "", 0
}
