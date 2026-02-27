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
	"github.com/openshift/assisted-installer-agent/src/util"
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

// Helper functions to build mounted folder paths from hostFsMountDir
func getMountedBootFolder(hostFsMountDir string) string {
	return path.Join(hostFsMountDir, "boot")
}

func getMountedArtifactsFolder(hostFsMountDir string) string {
	return path.Join(hostFsMountDir, "boot", artifactsFolder)
}

func getMountedBootLoaderFolder(hostFsMountDir string) string {
	return path.Join(hostFsMountDir, "boot", bootLoaderFolder)
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

	err := createFolders(*req.HostFsMountDir, defaultRetryAmount)
	if err != nil {
		log.Errorf("failed creating folders: %s", err.Error())
		return fmt.Errorf("failed creating folders: %s", err.Error())
	}

	if err := downloadArtifactsToTempFolder(req, caCertPath); err != nil {
		log.Errorf("failed downloading boot artifacts: %s", err.Error())
		return fmt.Errorf("failed downloading boot artifacts: %s", err.Error())
	}
	log.Info("Successfully downloaded boot artifacts")

	if err := createBootLoaderConfigInTempFolder(*req.RootfsURL); err != nil {
		log.Errorf("failed creating bootloader config file on host: %s", err.Error())
		return fmt.Errorf("failed creating bootloader config file on host: %s", err.Error())
	}
	log.Infof("Successfully wrote bootloader config to %s", path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName))

	if err := ensureBootHasSpace(getMountedBootFolder(*req.HostFsMountDir)); err != nil {
		log.Errorf("failed to ensure boot folder has enough space: %s", err.Error())
		return fmt.Errorf("failed to ensure boot folder has enough space: %s", err.Error())
	}

	if err := copyFilesToBootFolder(*req.HostFsMountDir); err != nil {
		log.Errorf("failed to move files to boot folder: %s", err.Error())
		return fmt.Errorf("failed to move files to boot folder: %s", err.Error())
	}

	log.Infof("Download boot artifacts completed successfully.")
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

func downloadArtifactsToTempFolder(req models.DownloadBootArtifactsRequest, caCertPath string) error {
	httpClient, err := createHTTPClient(caCertPath)
	if err != nil {
		return fmt.Errorf("failed creating secure assisted service client: %s", err.Error())
	}

	if err := download(httpClient, path.Join(tempBootArtifactsFolder, kernelFile), *req.KernelURL, defaultRetryAmount); err != nil {
		return fmt.Errorf("failed downloading kernel to host: %s", err.Error())
	}

	if err := download(httpClient, path.Join(tempBootArtifactsFolder, initrdFile), *req.InitrdURL, defaultRetryAmount); err != nil {
		return fmt.Errorf("failed downloading initrd to host: %s", err.Error())
	}
	return nil
}

func createBootLoaderConfigInTempFolder(rootfsUrl string) error {
	// These are the actual paths to the kernel and initrd in the actual host
	kernelPath := path.Join("/boot", artifactsFolder, kernelFile)
	initrdPath := path.Join("/boot", artifactsFolder, initrdFile)

	var bootLoaderConfig string
	bootLoaderConfig = fmt.Sprintf(bootLoaderConfigTemplate, rootfsUrl, kernelPath, initrdPath)
	if runtime.GOARCH == "s390x" {
		bootLoaderConfig = fmt.Sprintf(bootLoaderConfigTemplateS390x, rootfsUrl, kernelPath, initrdPath)
	}

	bootLoaderConfigFile := path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName)
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

func createFolders(hostFsMountDir string, retryAmount int) error {
	var err error
	mountedBootFolder := getMountedBootFolder(hostFsMountDir)
	mountedArtifactsFolder := getMountedArtifactsFolder(hostFsMountDir)
	mountedBootLoaderFolder := getMountedBootLoaderFolder(hostFsMountDir)

	for i := 0; i < retryAmount; i++ {
		log.Debugf("Creating folders attempt %d/%d", i, retryAmount)
		err = syscall.Mount(mountedBootFolder, mountedBootFolder, "", syscall.MS_REMOUNT, "")
		if err != nil {
			log.Warnf("failed to mount boot folder [%s]: %s\nRetrying in %s", mountedBootFolder, err.Error(), defaultRetryDelay)
			time.Sleep(defaultRetryDelay)
			continue
		}
		syscall.Sync()
		if err = createFolderIfNotExist(mountedArtifactsFolder); err != nil {
			log.Warnf("failed to create artifacts folder [%s]: %s\nRetrying in %s", mountedArtifactsFolder, err.Error(), defaultRetryDelay)
			time.Sleep(defaultRetryDelay)
			continue
		}
		if err = createFolderIfNotExist(mountedBootLoaderFolder); err != nil {
			log.Warnf("failed to create bootloader folder [%s]: %s\nRetrying in %s", mountedBootLoaderFolder, err.Error(), defaultRetryDelay)
			time.Sleep(defaultRetryDelay)
			continue
		}
		if err = createFolderIfNotExist(tempBootArtifactsFolder); err != nil {
			log.Warnf("failed to create temp boot artifacts folder [%s]: %s\nRetrying in %s", tempBootArtifactsFolder, err.Error(), defaultRetryDelay)
			time.Sleep(defaultRetryDelay)
			continue
		}
		log.Debug("All folders created successfully")
		return nil
	}
	return fmt.Errorf("failed to create folders: %w", err)
}

func ensureBootHasSpace(hostFsMountDir string) error {
	mountedBootFolder := getMountedBootFolder(hostFsMountDir)
	artifactsSize, err := calculateBootArtifactsSize()
	if err != nil {
		return fmt.Errorf("failed to calculate size of boot artifacts: %w", err)
	}
	log.Debugf("Boot artifacts total size: %d bytes", artifactsSize)

	freeSpace, err := getFreeSpace(mountedBootFolder)
	if err != nil {
		return fmt.Errorf("failed to get free space of boot folder [%s]: %w", mountedBootFolder, err)
	}
	log.Debugf("Free space in boot folder: %d bytes", freeSpace)

	if freeSpace > artifactsSize {
		return nil
	}

	log.Warnf("Boot folder does not have enough space. Wanted: %d bytes, Available: %d bytes. Attempting to reclaim space", artifactsSize, freeSpace)
	if err = reclaimBootFolderSpace(); err != nil {
		return fmt.Errorf("failed to reclaim boot folder space: %w", err)
	}

	freeSpace, err = getFreeSpace(mountedBootFolder)
	if err != nil {
		return fmt.Errorf("failed to get free space of boot folder [%s]: %w", mountedBootFolder, err)
	}

	if freeSpace < artifactsSize {
		return fmt.Errorf("boot folder does not have enough space, artifacts size: %d, free space: %d\nRetrying...", artifactsSize, freeSpace)
	}
	return nil
}

// getFreeSpace returns the available space in the given folder
func getFreeSpace(folder string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(folder, &stat); err != nil {
		return 0, fmt.Errorf("failed to statfs %s: %w", folder, err)
	}
	// G115: Integer overflow is not possible for realistic filesystem sizes (<< uint64 max)
	return stat.Bavail * uint64(stat.Bsize), nil // #nosec G115
}

// calculateBootArtifactsSize calculates the total size of downloaded boot artifacts and bootloader config
func calculateBootArtifactsSize() (uint64, error) {
	var totalSize uint64

	// Calculate size of kernel file
	kernelPath := path.Join(tempBootArtifactsFolder, kernelFile)
	kernelInfo, err := os.Stat(kernelPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat kernel file %s: %w", kernelPath, err)
	}
	// G115: Integer overflow is not possible for realistic filesystem sizes (<< uint64 max)
	totalSize += uint64(kernelInfo.Size()) // #nosec G115

	// Calculate size of initrd file
	initrdPath := path.Join(tempBootArtifactsFolder, initrdFile)
	initrdInfo, err := os.Stat(initrdPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat initrd file %s: %w", initrdPath, err)
	}
	// G115: Integer overflow is not possible for realistic filesystem sizes (<< uint64 max)
	totalSize += uint64(initrdInfo.Size()) // #nosec G115

	// Calculate size of bootloader config file
	bootLoaderConfigPath := path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName)
	bootLoaderInfo, err := os.Stat(bootLoaderConfigPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat bootloader config file %s: %w", bootLoaderConfigPath, err)
	}
	// G115: Integer overflow is not possible for realistic filesystem sizes (<< uint64 max)
	totalSize += uint64(bootLoaderInfo.Size()) // #nosec G115

	return totalSize, nil
}

func reclaimBootFolderSpace() error {
	stdout, stderr, exitCode := util.ExecutePrivileged("rpm-ostree", "cleanup", "--os=rhcos", "-r")
	log.Debugf("Cleanup RHCOS stdout: %s\nstderr: %s\nexitCode: %d", stdout, stderr, exitCode)
	if exitCode != 0 {
		return fmt.Errorf("Cleanup command for RHCOS failed: %s: %s", stdout, stderr)
	}
	log.Info("Successfully cleaned up RHCOS")
	return nil
}

func copyFilesToBootFolder(hostFsMountDir string) error {
	mountedArtifactsFolder := getMountedArtifactsFolder(hostFsMountDir)
	mountedBootLoaderFolder := getMountedBootLoaderFolder(hostFsMountDir)
	if err := copyFile(path.Join(tempBootArtifactsFolder, kernelFile), path.Join(mountedArtifactsFolder, kernelFile)); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", path.Join(tempBootArtifactsFolder, kernelFile), path.Join(mountedArtifactsFolder, kernelFile), err)
	}
	if err := copyFile(path.Join(tempBootArtifactsFolder, initrdFile), path.Join(mountedArtifactsFolder, initrdFile)); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", path.Join(tempBootArtifactsFolder, initrdFile), path.Join(mountedArtifactsFolder, initrdFile), err)
	}
	if err := copyFile(path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName), path.Join(mountedBootLoaderFolder, bootLoaderConfigFileName)); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName), path.Join(mountedBootLoaderFolder, bootLoaderConfigFileName), err)
	}
	log.Infof("Successfully moved files to /boot folder.")
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", src, dst, err)
	}

	// Preserve file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", src, err)
	}
	if err = os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dst, err)
	}
	if err = destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file %s: %w", dst, err)
	}
	return nil
}
