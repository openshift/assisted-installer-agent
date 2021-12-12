package apivip_check

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

	if err := httpDownload(*checkAPIRequest.URL); err != nil {
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

func httpDownload(uri string) error {
	client := http.Client{}
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return errors.Wrap(err, "HTTP download - failed to create request")
	}
	req.Header = http.Header{
		"Accept": []string{fmt.Sprintf("application/vnd.coreos.ignition+json; version=%s", ignitionVersion)},
	}

	res, err := client.Do(req)
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
