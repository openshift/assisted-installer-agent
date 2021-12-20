package apivip_check

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	b64 "encoding/base64"

	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const ignitionVersion string = "3.2.0"

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

	if err := httpDownload(checkAPIRequest); err != nil {
		wrapped := errors.Wrap(err, "Failed to download worker.ign file")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false), wrapped.Error(), 0
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

func httpDownload(connectivityReq models.APIVipConnectivityRequest) error {
	var client *http.Client

	if connectivityReq.CaCertificate != nil {
		caCertPool := x509.NewCertPool()
		decodedCaCert, err := b64.StdEncoding.DecodeString(*connectivityReq.CaCertificate)
		if err != nil {
			return errors.Wrap(err, "Failed to decode CaCertificate")
		}
		if ok := caCertPool.AppendCertsFromPEM(decodedCaCert); !ok {
			return errors.Errorf("unable to parse cert")
		}
		client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			},
		}
	} else {
		client = &http.Client{}
	}

	req, _ := http.NewRequest("GET", *connectivityReq.URL, nil)

	req.Header = http.Header{
		"Accept": []string{fmt.Sprintf("application/vnd.coreos.ignition+json; version=%s", ignitionVersion)},
	}

	if connectivityReq.IgnitionEndpointToken != nil {
		bearerToken := fmt.Sprintf("Bearer %s", *connectivityReq.IgnitionEndpointToken)
		req.Header.Set("Authorization", bearerToken)
	}

	res, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "HTTP download failure")
	}

	if res.StatusCode != http.StatusOK {
		return errors.Errorf("HTTP download failure. Status Code: %v", res.StatusCode)
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
