package apivip_check

import (
	"crypto/tls"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/coreos/ignition/v2/config/v3_2"
	ignition_types "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const ignitionVersion string = "3.2.0"

func getIgnitionFile(ignition *ignition_types.Config, path string) *ignition_types.File {
	for _, file := range ignition.Storage.Files {
		if file.Path == path {
			return &file
		}
	}

	return nil
}

func copyIgnitionFile(originalConfig *ignition_types.Config, filteredConfig *ignition_types.Config, path string) {
	if file := getIgnitionFile(originalConfig, path); file != nil {
		filteredConfig.Storage.Files = append(filteredConfig.Storage.Files, *file)
	}
}

// copyIgnitionDiskEncryptionInformation copies disk-encryption related
// information. See filterIgnition for more info.
func copyIgnitionDiskEncryptionInformation(originalConfig *ignition_types.Config, filteredConfig *ignition_types.Config) {
	filteredConfig.Storage.Luks = originalConfig.Storage.Luks
}

// copyIgnitionManagedNetworkIndications copies (see filterIgnition for more
// info) ignition storage file entries that can be used by the service in order
// to detect whether this worker ignition file originated from a cluster that
// has managed networking or not. This helps the service, for example, to know
// whether it should perform DNS validations for the day-2 hosts trying to join
// that cluster or not (as managed network clusters can be joined without any
// DNS configuration, they do not require such validations).
func copyIgnitionManagedNetworkIndications(originalConfig *ignition_types.Config, filteredConfig *ignition_types.Config) {
	// This is a hack - since we have no official way to know whether a worker
	// ignition file originated from a cluster with managed networking or not,
	// we instead rely on the presence of coredns and keepalived pod manifests
	// to indicate that. We only expect those to be present in clusters with
	// managed networking. To be a bit more robust, we consider the presence of
	// any one of them to mean that the cluster has managed networking. This
	// gives us better forwards compatibility if one of them gets renamed /
	// replaced with other technologies in future OCP versions.
	//
	// Another way in which this is hacky is that users could manually create
	// static pods with the same name as part of their machine-configs, in
	// which case we would have a false-positive detection. But that is
	// admittedly very unlikely.
	//
	// Hopefully we can negotiate with the relevant OCP teams to have a more
	// official, stable way to have this detection - like a magic empty file
	// placed somewhere in the ignition that we can check for the presence of.
	// Once we have such file, we can slowly deprecate this detection mechanism
	// and fully move to the new one by inspecting that file instead.
	copyIgnitionFile(originalConfig, filteredConfig, "/etc/kubernetes/manifests/coredns.yaml")
	copyIgnitionFile(originalConfig, filteredConfig, "/etc/kubernetes/manifests/keepalived.yaml")
}

// filterIgnition removes unnecessary sections of the ignition config, leaving
// only those needed by the service. We do this because the ignition config
// tends to be quite large.
func filterIgnition(originalConfig *ignition_types.Config) *ignition_types.Config {
	// Marshal then parse for the sake of cloning
	config, _ := json.Marshal(originalConfig)
	filteredConfig, _, _ := v3_2.Parse(config)

	filteredConfig.Storage.Files = []ignition_types.File{}
	filteredConfig.Systemd.Units = []ignition_types.Unit{}

	copyIgnitionDiskEncryptionInformation(originalConfig, &filteredConfig)
	copyIgnitionManagedNetworkIndications(originalConfig, &filteredConfig)

	// NOTE: add here any additional objects from the config (when needed by the service)

	return &filteredConfig
}

func CheckAPIConnectivity(checkAPIRequestStr string, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var checkAPIRequest models.APIVipConnectivityRequest

	if err := json.Unmarshal([]byte(checkAPIRequestStr), &checkAPIRequest); err != nil {
		wrapped := errors.Wrap(err, "Error unmarshaling APIVipConnectivityRequest")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false, ""), wrapped.Error(), -1
	}

	if checkAPIRequest.URL == nil {
		err := errors.New("Missing URL in checkAPIRequest")
		log.WithError(err).Error(err.Error())
		return createResponse(false, ""), err.Error(), -1
	}

	ignition, err := httpDownload(checkAPIRequest)
	if err != nil {
		wrapped := errors.Wrap(err, "Failed to download worker.ign file")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false, ""), wrapped.Error(), 0
	}

	config, _, err := v3_2.ParseCompatibleVersion([]byte(ignition))
	if err != nil {
		wrapped := errors.Wrap(err, "Invalid ignition format")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false, ""), wrapped.Error(), 0
	}

	configBytes, err := json.Marshal(filterIgnition(&config))
	if err != nil {
		wrapped := errors.Wrap(err, "Failed to filter ignition")
		log.WithError(err).Error(wrapped.Error())
		return createResponse(false, ""), wrapped.Error(), 0
	}

	return createResponse(true, string(configBytes)), "", 0
}

func createResponse(success bool, ignition string) string {
	checkAPIResponse := models.APIVipConnectivityResponse{
		Ignition:  ignition,
		IsSuccess: success,
	}
	bytes, err := json.Marshal(checkAPIResponse)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func httpDownload(connectivityReq models.APIVipConnectivityRequest) (string, error) {
	var client *http.Client

	if connectivityReq.CaCertificate != nil {
		caCertPool := x509.NewCertPool()
		decodedCaCert, err := b64.StdEncoding.DecodeString(*connectivityReq.CaCertificate)
		if err != nil {
			return "", errors.Wrap(err, "Failed to decode CaCertificate")
		}
		if ok := caCertPool.AppendCertsFromPEM(decodedCaCert); !ok {
			return "", errors.Errorf("unable to parse cert")
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
		return "", errors.Wrap(err, "HTTP download failure")
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.Errorf("HTTP download failure. Status Code: %v", res.StatusCode)
	}

	defer res.Body.Close()
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "File read failure")
	}

	if len(bytes) == 0 {
		return "", errors.New("Empty Ignition file")
	}

	var js json.RawMessage
	if err = json.Unmarshal(bytes, &js); err != nil {
		return "", errors.Wrap(err, "Error unmarshaling Ignition string")
	}

	return string(js), err
}
