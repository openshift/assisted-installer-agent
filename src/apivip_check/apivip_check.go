package apivip_check

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

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
		return createRepsonse(false), wrapped.Error(), -1
	}

	if checkAPIRequest.URL == nil {
		err := errors.New("Missing URL in checkAPIRequest")
		log.WithError(err).Error(err.Error())
		return createRepsonse(false), err.Error(), -1
	}
	
	if err := httpDownload(*checkAPIRequest.URL + WorkerIgnitionPath, log); err != nil {
		wrapped := errors.Wrap(err, "Failed to download worker.ign file")
		log.WithError(err).Error(wrapped.Error())
		return createRepsonse(false), wrapped.Error(), -1
	}

	return createRepsonse(true), "", 0
}

func createRepsonse(success bool) string {
	checkAPIResponse := models.APIVipConnectivityResponse{
		IsSuccess: success,
	}
	bytes, err := json.Marshal(checkAPIResponse)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func httpDownload(uri string, log logrus.FieldLogger) error {
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
