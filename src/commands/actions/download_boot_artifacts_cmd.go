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

type folders struct {
	// bootFolder is the folder where the /boot directory is mounted
	bootFolder string
	// hostArtifactsFolder is the folder where boot artifacts will eventually be moved to in the /boot folder
	hostArtifactsFolder string
	// bootLoaderFolder is where the bootloader config is the will eventually exist in the /boot folder
	bootLoaderFolder string
	// tempDownloadFolder is the folder where the artifacts will be temporarily downloaded to
	tempDownloadFolder string
}

const (
	defaultRetryAmount                   = 5
	defaultRetryDelay                    = 1 * time.Minute
	artifactsFolder               string = "/discovery"
	bootLoaderFolder              string = "/loader/entries"
	tempBootArtifactsFolder       string = "/tmp/boot"
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

	folders, err := createFolders(*req.HostFsMountDir, defaultRetryAmount)
	if err != nil {
		log.Errorf("failed creating folders: %s", err.Error())
		return fmt.Errorf("failed creating folders: %s", err.Error())
	}

	if err := downloadArtifacts(req, caCertPath, folders); err != nil {
		log.Errorf("failed downloading boot artifacts: %s", err.Error())
		return fmt.Errorf("failed downloading boot artifacts: %s", err.Error())
	}
	log.Info("Successfully downloaded boot artifacts")

	if err := createBootLoaderConfig(*req.RootfsURL, folders); err != nil {
		log.Errorf("failed creating bootloader config file on host: %s", err.Error())
		return fmt.Errorf("failed creating bootloader config file on host: %s", err.Error())
	}
	log.Infof("Successfully created bootloader config.")
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
		time.Sleep(defaultRetryDelay)
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

func downloadArtifacts(req models.DownloadBootArtifactsRequest, caCertPath string, folders *folders) error {
	httpClient, err := createHTTPClient(caCertPath)
	if err != nil {
		return fmt.Errorf("failed creating secure assisted service client: %s", err.Error())
	}

	if err := download(httpClient, path.Join(folders.tempDownloadFolder, kernelFile), *req.KernelURL, defaultRetryAmount); err != nil {
		return fmt.Errorf("failed downloading kernel to host: %s", err.Error())
	}

	if err := download(httpClient, path.Join(folders.tempDownloadFolder, initrdFile), *req.InitrdURL, defaultRetryAmount); err != nil {
		return fmt.Errorf("failed downloading initrd to host: %s", err.Error())
	}
	return nil
}

func createBootLoaderConfig(rootfsUrl string, folders *folders) error {
	kernelPath := path.Join("/boot", artifactsFolder, kernelFile)
	initrdPath := path.Join("/boot", artifactsFolder, initrdFile)
	bootLoaderConfigFile := path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName)
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

func createFolders(hostFsMountDir string, retryAmount int) (*folders, error) {
	var err error
	bootFolder := path.Join(hostFsMountDir, "boot")
	folders := &folders{
		bootFolder:          bootFolder,
		hostArtifactsFolder: path.Join(bootFolder, artifactsFolder),
		bootLoaderFolder:    path.Join(bootFolder, bootLoaderFolder),
		tempDownloadFolder:  path.Join(tempBootArtifactsFolder, artifactsFolder),
	}

	for i := 0; i < retryAmount; i++ {
		log.Debugf("Creating folders attempt %d/%d", i, retryAmount)
		err = syscall.Mount(folders.bootFolder, folders.bootFolder, "", syscall.MS_REMOUNT, "")
		if err != nil {
			log.Warnf("failed to mount boot folder [%s]: %s\nRetrying in %s", folders.bootFolder, err.Error(), defaultRetryDelay)
			continue
		}
		syscall.Sync()
		if err = createFolderIfNotExist(folders.hostArtifactsFolder); err != nil {
			log.Warnf("failed to create artifacts folder [%s]: %s\nRetrying in %s", folders.hostArtifactsFolder, err.Error(), defaultRetryDelay)
			continue
		}
		if err = createFolderIfNotExist(folders.bootLoaderFolder); err != nil {
			log.Warnf("failed to create bootloader folder [%s]: %s\nRetrying in %s", folders.bootLoaderFolder, err.Error(), defaultRetryDelay)
			continue
		}
		if err = createFolderIfNotExist(folders.tempDownloadFolder); err != nil {
			log.Warnf("failed to create temp download folder [%s]: %s\nRetrying in %s", folders.tempDownloadFolder, err.Error(), defaultRetryDelay)
			continue
		}
		log.Debug("Folders created successfully")
		return folders, nil
	}
	return nil, fmt.Errorf("failed to create folders: %w", err)
}
