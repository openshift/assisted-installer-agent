package apivip_check

import (
	"crypto/tls"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coreos/ignition/v2/config/v3_2"
	ignition_types "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const ignitionVersion string = "3.2.0"

const responseErrorSizeLimit int = 512

const ignitionDownloadErrorStderr string = "ignition download error"

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
// to detect whether this ignition file originated from a cluster that
// has managed networking or not. This helps the service, for example, to know
// whether it should perform DNS validations for the day-2 hosts trying to join
// that cluster or not (as managed network clusters can be joined without any
// DNS configuration, they do not require such validations).
func copyIgnitionManagedNetworkIndications(originalConfig *ignition_types.Config, filteredConfig *ignition_types.Config) {
	// This is a hack - since we have no official way to know whether a
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

	// Some clusters (like hypershift) don't use a domain name at all to
	// connect to the cluster, the service can tell if this is the case by
	// looking at the IP address of the kubeconfig within the ignition, so the
	// agent should include that file for the service
	copyIgnitionFile(originalConfig, filteredConfig, "/etc/kubernetes/kubeconfig")
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
		return createResponse("<unknown URL due to internal error>", false,
			"", fmt.Sprintf("internal error - failed to deserialize service request: %v", err), log), ignitionDownloadErrorStderr, -1
	}

	if checkAPIRequest.URL == nil {
		return createResponse("<unknown URL due to internal error>", false,
			"", "internal error - service request is missing URL", log), ignitionDownloadErrorStderr, -1
	}

	ignition, err := downloadIgnition(checkAPIRequest)
	if err != nil {
		return createResponse(*checkAPIRequest.URL, false, "",
			errors.Wrap(err, "ignition file download failed").Error(), log), ignitionDownloadErrorStderr, 0
	}

	config, _, err := v3_2.ParseCompatibleVersion([]byte(ignition))
	if err != nil {
		var wrapped error
		if len(ignition) > responseErrorSizeLimit {
			// hopefully if the user is hitting some sort of different server
			// other than MCS (e.g. generic load balancer error page), they can
			// try and infer what they're hitting from the first
			// responseErrorSizeLimit bytes. We don't want to return the entire
			// response body, as it's shown in the UI.
			wrapped = errors.Wrap(err, fmt.Sprintf(`response JSON is not a valid ignition file, first %d bytes are:
%s
parse error is`, responseErrorSizeLimit, ignition[:responseErrorSizeLimit]))
		} else {
			wrapped = errors.Wrap(err, fmt.Sprintf(`response is not a valid ignition file:
%s
parse error is`, ignition))
		}
		return createResponse(*checkAPIRequest.URL, false, "", wrapped.Error(), log), ignitionDownloadErrorStderr, 0
	}

	configBytes, err := json.Marshal(filterIgnition(&config))
	if err != nil {
		return createResponse(*checkAPIRequest.URL, false, "",
			errors.Wrap(err, "internal error - failed to re-serialize filtered ignition config").Error(), log), ignitionDownloadErrorStderr, 0
	}

	return createResponse(*checkAPIRequest.URL, true, string(configBytes), "", log), "", 0
}

func createResponse(url string, success bool, ignition string, downloadError string, log logrus.FieldLogger) string {
	checkAPIResponse := models.APIVipConnectivityResponse{
		Ignition:      ignition,
		IsSuccess:     success,
		URL:           url,
		DownloadError: downloadError,
	}

	if downloadError != "" {
		log.Error(downloadError)
	}

	bytes, err := json.Marshal(checkAPIResponse)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func downloadIgnition(connectivityReq models.APIVipConnectivityRequest) (string, error) {
	var client *http.Client

	if connectivityReq.CaCertificate != nil {
		caCertPool := x509.NewCertPool()

		decodedCaCert, err := b64.StdEncoding.DecodeString(*connectivityReq.CaCertificate)
		if err != nil {
			return "", errors.Wrap(err, "ignition endpoint CA certificate is not valid base64")
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

	req, err := http.NewRequest("GET", *connectivityReq.URL, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request")
	}

	req.Header = http.Header{
		"Accept": []string{fmt.Sprintf("application/vnd.coreos.ignition+json; version=%s", ignitionVersion)},
	}

	if connectivityReq.IgnitionEndpointToken != nil {
		// Kept for backwards compatibility, in case an older assisted-service is used,
		// should be removed once this field has been fully removed
		bearerToken := fmt.Sprintf("Bearer %s", *connectivityReq.IgnitionEndpointToken)
		req.Header.Set("Authorization", bearerToken)
	}

	if connectivityReq.RequestHeaders != nil {
		for _, hdr := range connectivityReq.RequestHeaders {
			req.Header.Set(hdr.Key, hdr.Value)
		}
	}

	res, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "request failed")
	}
	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		if res.ContentLength != -1 {
			return "", errors.Wrap(err, fmt.Sprintf("error while reading response body, read %d out of %d bytes", len(bytes), res.ContentLength))
		}

		return "", errors.Wrap(err, fmt.Sprintf("error while reading response body, read %d bytes", len(bytes)))
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", errors.Errorf("bad status code: %v. server response: %s", res.StatusCode, string(bytes))
	}

	if len(bytes) == 0 {
		return "", errors.Errorf("server responsed with status code %v but the response was empty", res.StatusCode)
	}

	var js json.RawMessage
	if err = json.Unmarshal(bytes, &js); err != nil {
		if len(bytes) > responseErrorSizeLimit {
			// hopefully if the user is hitting some sort of different server
			// other than MCS (e.g. generic load balancer error page), they can
			// try and infer what they're hitting from the first
			// responseErrorSizeLimitbytes bytes. We don't want to return the
			// entire response body, as it's shown in the UI.
			return "", errors.Wrap(err, fmt.Sprintf(`expected ignition but got non-valid json, first %d bytes are:
%s
parse error is`, responseErrorSizeLimit, string(bytes[:responseErrorSizeLimit])))
		}

		return "", errors.Wrap(err, fmt.Sprintf(`response is not valid json:
%s
parse error is`, string(bytes)))
	}

	return string(js), err
}
