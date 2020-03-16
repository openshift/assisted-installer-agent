package commands

import (
	"encoding/json"
	"github.com/filanov/bm-inventory/models"
	"github.com/ori-amizur/introspector/src/scanners"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"regexp"
	"strings"
)

type Done struct {}

type Any interface {}

func getOutgoingNics() []string {
	ret := make([]string, 0)
	r := regexp.MustCompile("^(?:eth|ens|eno|enp|wlp)\\d")
	for _, nic := range scanners.ReadNics() {
		if r.MatchString(nic.Name) {
			ret = append(ret, nic.Name)
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


func l3CheckAddressOnNic(address string, outgoingNic string, l3chan chan *models.L3Connectivity){
	ret := &models.L3Connectivity{
		OutgoingNic:outgoingNic,
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

func l3CheckAddress(address string, outgoingNics []string, l3chan, doneChan chan Any){
	defer sendDone(doneChan)
	innerChan := make(chan *models.L3Connectivity, 1000)
	for _, nic := range outgoingNics {
		go l3CheckAddressOnNic(address, nic, innerChan)
	}
	successful := false
	for i := 0; i != len(outgoingNics) ; i++ {
		ret := <- innerChan
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

func l3CheckConnectivity(addresses []string, outgoingNics [] string, l3chan chan Any)  {
	defer sendDone(l3chan)
	doneChan := make(chan Any)
	for _, address := range addresses {
		go l3CheckAddress(address, outgoingNics, l3chan, doneChan)
	}
	for i := 0; i != len(addresses); i++ {
		<- doneChan
	}
}

func l2CheckAddressOnNic(dstAddr string, dstMac string, srcNic string, l2chan chan Any) {
	defer sendDone(l2chan)
	ret := &models.L2Connectivity{
		OutgoingNic:       srcNic,
		RemoteIPAddress:   dstAddr,
		RemoteMac:         "",
		Successful:        false,
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
	replies := 0
	for _, line := range lines[1:] {
		parts = rRegexp.FindStringSubmatch(line)
		if len(parts) != 3 {
			continue
		}
		remoteMac := strings.ToLower(parts[2])
		ret.RemoteMac = remoteMac
		ret.Successful = strings.ToLower(dstMac) == remoteMac
		if !ret.Successful {
			log.Warnf("Unexpected mac address for arping %s on nic %s: %s", dstAddr, srcNic, remoteMac)
		}
		l2chan <- ret
		replies++
	}
	if replies == 0 {
		l2chan <- ret
	}
}

func l2CheckAddress(dstAddr string, dstMac string , sourceNics []string, l2chan chan Any, l2DoneChan chan Any) {
	defer sendDone(l2DoneChan)
	innerChan := make(chan Any, 1000)
	for _, srcNic := range sourceNics {
		go l2CheckAddressOnNic(dstAddr, dstMac, srcNic, innerChan)
	}
	successful := false
	for numDone := 0 ; numDone != len(sourceNics); {
		iret := <- innerChan
		switch ret := iret.(type) {
		case *models.L2Connectivity:
			if ret.Successful {
				successful = true
				l2chan <- ret
			}
		case Done:
			numDone++
		}
	}
	if !successful {
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

func l2CheckConnectivity(destinationNics []*models.ConnectivityCheckNic, sourceNics [] string, l2chan chan Any)  {
	defer sendDone(l2chan)
	doneChan := make(chan Any)
	for _, destNic := range destinationNics {
		for _, address := range destNic.IPAddresses {
			go l2CheckAddress(address, destNic.Mac, sourceNics, l2chan, doneChan)
		}
	}
	for i := 0; i != len(destinationNics) ; i++ {
		<- doneChan
	}
}



func checkNode(outgoingNics []string, node *models.ConnectivityCheckNode, nodeChan chan *models.ConnectivityRemoteNode) {
	ret := &models.ConnectivityRemoteNode{
		NodeID:         node.NodeID,
		L2Connectivity: make([]*models.L2Connectivity, 0),
		L3Connectivity: make([]*models.L3Connectivity, 0),
	}
	r := regexp.MustCompile("^(?:eth|ens|eno|enp)\\d")
	checkedNics := make([]*models.ConnectivityCheckNic, 0)
	for _, nic := range node.Nics {
		if r.MatchString(nic.Name) {
			checkedNics = append(checkedNics, nic)
		}
	}
	addresses := getOutgoingAddresses(checkedNics)
	ch := make(chan Any, 1000)
	go l3CheckConnectivity(addresses, outgoingNics, ch)
	go l2CheckConnectivity(checkedNics, outgoingNics, ch)
	for numDone := 0 ; numDone != 2 ;  {
		iret := <- ch
		switch value := iret.(type) {
		case *models.L2Connectivity:
			ret.L2Connectivity = append(ret.L2Connectivity, value)
		case *models.L3Connectivity:
			ret.L3Connectivity = append(ret.L3Connectivity, value)
		case Done:
			numDone++
		}
	}
	nodeChan <- ret
}

func ConnectivityCheck(input string) (string, error) {
	params := make(models.ConnectivityCheckParams, 0)
	err := json.Unmarshal([]byte(input), &params)
	if err != nil {
		log.Warnf("Error unmarshalling json %s: %s", input, err.Error())
		return "", err
	}
	nics := getOutgoingNics()
	nodeChan := make(chan *models.ConnectivityRemoteNode, 0)
	for _, node := range params {
		go checkNode(nics, node, nodeChan)
	}
	ret := models.ConnectivityReport{RemoteNodes:make([]*models.ConnectivityRemoteNode, 0)}
	for i := 0 ; i != len(params) ; i++ {
		ret.RemoteNodes = append(ret.RemoteNodes, <- nodeChan)
	}
	bytes, err := json.Marshal(&ret)
	if err != nil {
		log.Warnf("Could not marshal json: $s", err.Error())
		return "", err
	}
	return string(bytes), nil
}
