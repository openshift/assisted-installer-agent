package actions

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
	log "github.com/sirupsen/logrus"
)

type downloadBootArtifacts struct {
	args        []string
	agentConfig *config.AgentConfig
}

func (a *downloadBootArtifacts) Validate() error {
	return ValidateCommon("download boot artifacts", 1, a.args, &models.DownloadBootArtifactsRequest{})
}

func (a *downloadBootArtifacts) Run() (stdout, stderr string, exitCode int) {
	err := run(a.agentConfig.InfraEnvID, a.Args()[0], a.agentConfig.CACertificatePath)
	if err != nil {
		return "", err.Error(), -1
	}
	return "Successfully downloaded boot artifacts", "", 0
}

// Unused, but required as part of ActionInterface
func (a *downloadBootArtifacts) Command() string {
	return "download_boot_artifacts"
}

// Unused, but required as part of ActionInterface
func (a *downloadBootArtifacts) Args() []string {
	return a.args
}

const (
	retryDownloadAmount                  = 5
	defaultDownloadRetryDelay            = 1 * time.Minute
	artifactsFolder               string = "/boot/discovery"
	kernelFile                    string = "vmlinuz"
	initrdFile                    string = "initrd"
	bootLoaderConfigFileName      string = "/00-assisted-discovery.conf"
	bootLoaderConfigTemplateS390x string = `title Assisted Installer Discovery
version 999
options random.trust_cpu=on ai.ip_cfg_override=1 ignition.firstboot ignition.platform.id=metal coreos.live.rootfs_url=%s
linux %s
initrd %s`
	bootLoaderConfigTemplate string = `title Assisted Installer Discovery
version 999
options random.trust_cpu=on ignition.firstboot ignition.platform.id=metal 'coreos.live.rootfs_url=%s'
linux %s
initrd %s`
)

func run(infraEnvId, downloaderRequestStr, caCertPath string) error {
	var req models.DownloadBootArtifactsRequest
	if err := json.Unmarshal([]byte(downloaderRequestStr), &req); err != nil {
		return fmt.Errorf("failed unmarshalling download boot artifacts request: %w", err)
	}

	bootFolder := path.Join(*req.HostFsMountDir, "/boot")
	if err := syscall.Mount(bootFolder, bootFolder, "", syscall.MS_REMOUNT, ""); err != nil {
		return fmt.Errorf("failed remounting /host/boot folder as rw: %w", err)
	}

	hostArtifactsFolder := path.Join(*req.HostFsMountDir, artifactsFolder)
	bootLoaderFolder := path.Join(*req.HostFsMountDir, "/boot/loader/entries")
	if err := createFolders(hostArtifactsFolder, bootLoaderFolder); err != nil {
		return fmt.Errorf("failed creating folders: %w", err)
	}

	httpClient, err := createHTTPClient(caCertPath)
	if err != nil {
		return fmt.Errorf("failed creating secure assisted service client: %w", err)
	}

	if err := download(httpClient, path.Join(hostArtifactsFolder, kernelFile), *req.KernelURL, retryDownloadAmount); err != nil {
		return fmt.Errorf("failed downloading kernel to host: %w", err)
	}

	if err := download(httpClient, path.Join(hostArtifactsFolder, initrdFile), *req.InitrdURL, retryDownloadAmount); err != nil {
		return fmt.Errorf("failed downloading initrd to host: %w", err)
	}

	if err := createBootLoaderConfig(*req.RootfsURL, artifactsFolder, bootLoaderFolder); err != nil {
		return fmt.Errorf("failed creating bootloader config file on host: %w", err)
	}

	log.Infof("Successfully downloaded boot artifacts and created bootloader config.")
	return nil
}

func createHTTPClient(caCertPath string) (*http.Client, error) {
	client := &http.Client{}
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open cert file %s, %s", caCertPath, err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append cert %s, %s", caCertPath, err)
		}

		t := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    caCertPool,
				MinVersion: tls.VersionTLS12,
			},
		}
		client.Transport = t
	}
	return client, nil
}

func download(httpClient *http.Client, filePath, url string, retry int) error {
	var downloadErr error
	var res *http.Response
	for attempts := 0; attempts < retry; attempts++ {
		res, downloadErr = httpClient.Get(url)
		if downloadErr == nil && (res.StatusCode >= 200 && res.StatusCode < 300) {
			break
		}
		downloadErr = fmt.Errorf("failed downloading boot artifact from %s, status code received: %d, attempt %d/%d, download error: %w",
			url, res.StatusCode, attempts, retry, downloadErr)
		log.Warn(downloadErr.Error())
		time.Sleep(defaultDownloadRetryDelay)
	}

	if downloadErr != nil {
		return fmt.Errorf("failed getting %s: %w", url, downloadErr)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read body while getting url %s: %w", url, err)
	}
	err = os.WriteFile(filePath, body, 0644) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed writing file %s: %w", filePath, err)
	}
	return nil
}

func createBootLoaderConfig(rootfsUrl, artifactsPath, bootLoaderPath string) error {
	kernelPath := path.Join(artifactsPath, kernelFile)
	initrdPath := path.Join(artifactsPath, initrdFile)
	bootLoaderConfigFile := path.Join(bootLoaderPath, bootLoaderConfigFileName)
	var bootLoaderConfig string
	bootLoaderConfig = fmt.Sprintf(bootLoaderConfigTemplate, rootfsUrl, kernelPath, initrdPath)
	if runtime.GOARCH == "s390x" {
		bootLoaderConfig = fmt.Sprintf(bootLoaderConfigTemplateS390x, rootfsUrl, kernelPath, initrdPath)
	}

	if err := os.WriteFile(bootLoaderConfigFile, []byte(bootLoaderConfig), 0644); err != nil { //nolint:gosec
		return fmt.Errorf("failed writing bootloader config content to %s: %w", bootLoaderConfigFile, err)
	}
	return nil
}

func createFolderIfNotExist(folder string) error {
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		return os.MkdirAll(folder, 0755)
	}
	return nil
}

func createFolders(artifactsPath, bootLoaderPath string) error {
	err := createFolderIfNotExist(artifactsPath)
	if err != nil {
		return fmt.Errorf("failed to create artifacts folder [%s]: %w", artifactsPath, err)
	}
	err = createFolderIfNotExist(bootLoaderPath)
	if err != nil {
		return fmt.Errorf("failed to create bootloader folder [%s]: %w", bootLoaderPath, err)
	}
	return nil
}
