package tang_connectivity_check

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-multierror"
	"github.com/openshift/assisted-service/models"
	"github.com/openshift/assisted-service/pkg/tang"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const TangKeysPath = "/adv/"

func CheckTangConnectivity(tangServersDetails string, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var (
		multiErr                     error
		responses                    []*models.TangServerResponse
		checkTangConnectivityRequest models.TangConnectivityRequest
	)

	if err := json.Unmarshal([]byte(tangServersDetails), &checkTangConnectivityRequest); err != nil {
		wrapped := errors.Wrap(err, "Error unmarshaling TangConnectivityRequest")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false, nil), wrapped.Error(), -1
	}

	if checkTangConnectivityRequest.TangServers == nil {
		err := errors.New("Missing TangServers in checkTangConnectivityRequest")
		log.WithError(err).Error(err.Error())
		return createResponse(false, nil), err.Error(), -1
	}

	tangServers, err := tang.UnmarshalTangServers(*checkTangConnectivityRequest.TangServers)
	if err != nil {
		log.WithError(err)
		return createResponse(false, nil), err.Error(), -1
	}

	for _, ts := range tangServers {
		// Validate that the tang server URL was properly set
		if _, err = url.ParseRequestURI(ts.Url); err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}
		// Validate that the tang server Thumbprint was properly set
		if ts.Thumbprint == "" {
			err1 := errors.New(fmt.Sprintf("Tang thumbprint isn't set for server: %s", ts.Url))
			log.Error(err1)
			multiErr = multierror.Append(multiErr, err1)
			continue
		}
		// Attempt a request
		res, err1 := TangRequest(ts)
		if err1 != nil {
			multiErr = multierror.Append(multiErr, err1)
		} else {
			log.Debugf("tang server %s response is: %+v", ts.Url, res)
			responses = append(responses, res)
		}
	}

	if multiErr != nil {
		return createResponse(false, nil), multiErr.Error(), -1
	}
	return createResponse(true, responses), "", 0
}

func TangRequest(tangServer tang.TangServer) (*models.TangServerResponse, error) {
	client := &http.Client{}

	tangURL := fmt.Sprintf("%s%s%s", tangServer.Url, TangKeysPath, tangServer.Thumbprint)
	req, _ := http.NewRequest("GET", tangURL, nil)
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP GET failure")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("HTTP GET failure. Status Code: %v", res.StatusCode)
	}
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "response read failure")
	}
	if len(bytes) == 0 {
		return nil, errors.New("Empty tang response")
	}
	var tangServerResponse models.TangServerResponse
	if err = json.Unmarshal(bytes, &tangServerResponse); err != nil {
		return nil, errors.Wrap(err, "Error unmarshaling tang response")
	}
	tangServerResponse.TangURL = tangURL
	return &tangServerResponse, nil
}

func createResponse(success bool, tangServersResponse []*models.TangServerResponse) string {
	checkTangResponse := models.TangConnectivityResponse{
		IsSuccess:          success,
		TangServerResponse: tangServersResponse,
	}
	bytes, err := json.Marshal(checkTangResponse)
	if err != nil {
		return ""
	}
	return string(bytes)
}
