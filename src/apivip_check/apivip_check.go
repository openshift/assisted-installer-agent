package apivip_check

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	WorkerIgnitionPath = "/config/worker"
)

func CheckAPIConnectivity(checkAPIRequestStr string, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var checkAPIRequest models.APIVipConnectivityRequest

	if err := json.Unmarshal([]byte(checkAPIRequestStr), &checkAPIRequest); err != nil {
		wrapped := errors.Wrap(err, "Error unmarshaling APIVipConnectivityRequest")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false), wrapped.Error(), -1
	}

	if checkAPIRequest.URL == nil {
		err := errors.New("Missing URL in checkAPIRequest")
		log.WithError(err).Error(err.Error())
		return createResponse(false), err.Error(), -1
	}

	if err := httpDownload(*checkAPIRequest.URL + WorkerIgnitionPath); err != nil {
		wrapped := errors.Wrap(err, "Failed to download worker.ign file")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false), wrapped.Error(), 0
	}

	if checkAPIRequest.VerifyCidr {
		if err := verifyCIDR(*checkAPIRequest.URL, log); err != nil {
			wrapped := errors.Wrap(err, "CheckAPIConnectivity: failure verifying CIDR of API VIP")
			log.WithError(err).Error(wrapped.Error())
			return createResponse(false), wrapped.Error(), 0
		}
	}

	return createResponse(true), "", 0
}

func createResponse(success bool) string {
	checkAPIResponse := models.APIVipConnectivityResponse{
		IsSuccess: success,
	}
	bytes, err := json.Marshal(checkAPIResponse)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func httpDownload(uri string) error {
	res, err := http.Get(uri)
	if err != nil {
		return errors.Wrap(err, "HTTP download failure")
	}

	defer res.Body.Close()
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "File read failure")
	}

	if len(bytes) == 0 {
		return errors.New("Empty Ignition file")
	}

	var js json.RawMessage
	if err = json.Unmarshal(bytes, &js); err != nil {
		return errors.Wrap(err, "Error unmarshaling Ignition string")
	}

	return err
}

func verifyCIDR(uri string, log logrus.FieldLogger) error {
	apiVip, err := getIPByURI(uri)
	if err != nil {
		return errors.Wrap(err, "Failed to get VIP API")
	}

	_, err = calculateMachineNetworkCIDR(apiVip, log)
	if err != nil {
		return errors.Wrap(err, "Failed to calculate network CIDR")
	}

	return nil
}

func getIPByURI(uri string) (net.IP, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Wrap(err, "Failed parsing specified URL")
	}

	host, _, _ := net.SplitHostPort(u.Host)
	if ip := net.ParseIP(host); ip != nil {
		return ip, nil
	}

	addr, err := net.LookupIP(host)
	if err != nil {
		return nil, errors.Wrap(err, "Unknown host for specified API VIP")
	}
	return addr[0], nil
}

func calculateMachineNetworkCIDR(apiVip net.IP, log logrus.FieldLogger) (string, error) {

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "Failed to fetch machine's interfaces")
	}

	isVIPv4 := util.IsIPv4Addr(apiVip.String())
	for _, intf := range interfaces {

		addrs, _ := intf.Addrs()
		addrStrs := make([]string, 0)
		for _, addr := range addrs {
			addrStrs = append(addrStrs, addr.String())
		}

		if !isVIPv4 {
			util.SetV6PrefixesForAddress(intf.Name, &util.NetlinkRouteFinder{}, log, addrStrs)
		}

		for _, ipAddr := range addrStrs {

			_, ipNet, err := net.ParseCIDR(ipAddr)
			if err != nil {
				log.WithError(err).Warnf("Error parsing CIDR: %s", err)
				continue
			}

			if ipNet.Contains(apiVip) {
				return ipNet.String(), nil
			}
		}
	}

	return "", errors.Errorf("No suitable matching CIDR found for API VIP %s", apiVip)
}
