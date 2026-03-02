package actions

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("download boot artifacts cmd", func() {

	It("succeeds when given all args", func() {
		param := "{\"initrd_url\":\"http://test.com/api/v2/pxe-initrd?api_key=123&arch=x86_64&version=4.10\"," +
			"\"rootfs_url\":\"http://test.com/rootfs?arch=x86_64&version=4.10\"," +
			"\"kernel_url\":\"http://test.com/kernel?arch=x86_64&version=4.10\"," +
			"\"host_fs_mount_dir\":\"/host\"}"
		_, err := New(&config.AgentConfig{}, models.StepTypeDownloadBootArtifacts, []string{param})
		Expect(err).To(BeNil())
	})

	It("fails when no args are given", func() {
		_, err := New(&config.AgentConfig{}, models.StepTypeDownloadBootArtifacts, []string{})
		Expect(err).To(HaveOccurred())

	})

	It("fails when missing some args", func() {
		param := "{\"initrd_url\":\"http://test.com/api/v2/pxe-initrd?api_key=123&arch=x86_64&version=4.10\"," +
			"\"rootfs_url\":\"http://test.com/rootfs?arch=x86_64&version=4.10\"}"
		_, err := New(&config.AgentConfig{}, models.StepTypeDownloadBootArtifacts, []string{param})
		Expect(err).To(HaveOccurred())
	})

	It("fails when given bad input", func() {
		param := "{\"initrd_url\":\"http://test.com/api/v2/pxe-initrd?api_key=123&arch=x86_64&version=4.10\"," +
			"\"rootfs_url\":\"http://test.com/rootfs?arch=x86_64&version=4.10\"," +
			"\"kernel_url\":\"http://test.com/kernel?arch=x86_64&version=4.10\"" +
			"\"host_fs_mount_dir\":\"/host\"}"
		badParamsCommonTests(models.StepTypeDownloadBootArtifacts, []string{param})
	})

})
var _ = Describe("createFolders", func() {
	const (
		defaultTestRetryAmount = 3
	)
	var (
		tempDir string
		hostDir string
		bootDir string
		srcDir  string
	)

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "download-boot-artifacts")
		// createFolders uses syscall.Mount(MS_REMOUNT) which requires a real mount point and root
		if runtime.GOOS != "linux" {
			Skip("createFolders mount test only runs on Linux")
		}
		if syscall.Geteuid() != 0 {
			Skip("createFolders mount test requires root")
		}

		// Create a real mount point: bind-mount a temp dir onto hostDir/boot so MS_REMOUNT succeeds
		hostDir = filepath.Join(os.TempDir(), "createfolders-host")
		bootDir = filepath.Join(hostDir, "boot")
		srcDir = filepath.Join(os.TempDir(), "createfolders-src")
		Expect(os.MkdirAll(srcDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(bootDir, 0755)).To(Succeed())
		Expect(syscall.Mount(srcDir, bootDir, "", syscall.MS_BIND, "")).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		os.RemoveAll(tempBootArtifactsFolder)
		os.RemoveAll(hostDir)
		os.RemoveAll(srcDir)
		Expect(syscall.Unmount(bootDir, 0)).To(Succeed())
	})

	It("Successful folder creation", func() {
		By("artifacts folder should not exist before creation", func() {
			_, err := os.Stat(path.Join(bootDir, "discovery"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		By("boot loader config folder should not exist before creation", func() {
			_, err := os.Stat(path.Join(bootDir, "loader", "entries"))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		By("creating all required folders", func() {
			err := createFolders(hostDir, defaultTestRetryAmount)
			Expect(err).To(BeNil())
			// Verify folders were created at the expected paths
			_, err = os.Stat(getMountedArtifactsFolder(hostDir))
			Expect(err).To(Succeed())
			Expect(getMountedArtifactsFolder(hostDir)).To(Equal(path.Join(bootDir, artifactsFolder)))
			_, err = os.Stat(getMountedBootLoaderFolder(hostDir))
			Expect(err).To(Succeed())
			Expect(getMountedBootLoaderFolder(hostDir)).To(Equal(path.Join(bootDir, "loader", "entries")))
			_, err = os.Stat(getMountedBootFolder(hostDir))
			Expect(err).To(Succeed())
			Expect(getMountedBootFolder(hostDir)).To(Equal(bootDir))
			_, err = os.Stat(tempBootArtifactsFolder)
			Expect(err).To(Succeed())
		})
	})
})
var _ = Describe("createBootLoaderConfig", func() {
	var (
		bootFile string
	)

	const (
		hostFsMountDir = "/test"
		rootfsUrl      = "http://test.com/rootfs?arch=x86_64&version=4.10"
	)

	BeforeEach(func() {
		// Avoid createFolders here since it uses syscall.Mount(MS_REMOUNT) which fails in tests (requires root + real mount).
		Expect(os.MkdirAll(tempBootArtifactsFolder, 0755)).To(Succeed())
		bootFile = path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName)
	})

	AfterEach(func() {
		os.RemoveAll(hostFsMountDir)
		os.RemoveAll(tempBootArtifactsFolder)
	})

	It("Successful bootloader config creation", func() {
		err := createBootLoaderConfigInTempFolder(rootfsUrl)
		Expect(err).To(BeNil())
		bootConfigContents, err := os.ReadFile(bootFile)
		Expect(err).To(BeNil())
		Expect(string(bootConfigContents)).To(ContainSubstring(path.Join(artifactsFolder, kernelFile)))
		Expect(string(bootConfigContents)).To(ContainSubstring(path.Join(artifactsFolder, initrdFile)))
		Expect(string(bootConfigContents)).To(ContainSubstring(rootfsUrl))
	})

	It("Failed bootloader config creation - folder DNE", func() {
		os.RemoveAll(tempBootArtifactsFolder) // remove so write fails
		err := createBootLoaderConfigInTempFolder(rootfsUrl)
		Expect(err).To(HaveOccurred())
	})

	It("Bad bootloader config - incorrect rootfs URL", func() {
		err := createBootLoaderConfigInTempFolder("http://example.com/not-rootfs-url")
		Expect(err).To(BeNil())
		bootConfigContents, err := os.ReadFile(bootFile)
		Expect(err).To(BeNil())
		Expect(string(bootConfigContents)).ToNot(ContainSubstring(rootfsUrl))
	})
})

var _ = Describe("createFolderIfNotExist", func() {
	var tempDir string

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "test-folder-creation")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("creates a folder that does not exist", func() {
		testFolder := path.Join(tempDir, "new-folder")
		err := createFolderIfNotExist(testFolder)
		Expect(err).To(BeNil())

		// Verify the folder exists
		info, err := os.Stat(testFolder)
		Expect(err).To(BeNil())
		Expect(info.IsDir()).To(BeTrue())
	})

	It("succeeds when folder already exists", func() {
		testFolder := path.Join(tempDir, "existing-folder")
		Expect(os.MkdirAll(testFolder, 0755)).To(Succeed())

		err := createFolderIfNotExist(testFolder)
		Expect(err).To(BeNil())
	})

	It("creates nested folders", func() {
		testFolder := path.Join(tempDir, "parent", "child", "grandchild")
		err := createFolderIfNotExist(testFolder)
		Expect(err).To(BeNil())

		// Verify the nested folder exists
		info, err := os.Stat(testFolder)
		Expect(err).To(BeNil())
		Expect(info.IsDir()).To(BeTrue())
	})
})

var _ = Describe("copyFile", func() {
	var tempDir string

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "test-copy-file")
		Expect(os.MkdirAll(tempDir, 0755)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("successfully copies a file", func() {
		srcFile := path.Join(tempDir, "source.txt")
		dstFile := path.Join(tempDir, "destination.txt")
		content := []byte("test content")

		Expect(os.WriteFile(srcFile, content, 0600)).To(Succeed())

		err := copyFile(srcFile, dstFile)
		Expect(err).To(BeNil())

		// Verify content
		copiedContent, err := os.ReadFile(dstFile)
		Expect(err).To(BeNil())
		Expect(copiedContent).To(Equal(content))
	})

	It("preserves file permissions", func() {
		srcFile := path.Join(tempDir, "source.txt")
		dstFile := path.Join(tempDir, "destination.txt")

		Expect(os.WriteFile(srcFile, []byte("test"), 0600)).To(Succeed())

		err := copyFile(srcFile, dstFile)
		Expect(err).To(BeNil())

		// Verify permissions
		srcInfo, _ := os.Stat(srcFile)
		dstInfo, _ := os.Stat(dstFile)
		Expect(dstInfo.Mode()).To(Equal(srcInfo.Mode()))
	})

	It("fails when source file does not exist", func() {
		srcFile := path.Join(tempDir, "nonexistent.txt")
		dstFile := path.Join(tempDir, "destination.txt")

		err := copyFile(srcFile, dstFile)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to open source file"))
	})

	It("fails when destination directory does not exist", func() {
		srcFile := path.Join(tempDir, "source.txt")
		dstFile := path.Join(tempDir, "nonexistent-dir", "destination.txt")

		Expect(os.WriteFile(srcFile, []byte("test"), 0600)).To(Succeed())

		err := copyFile(srcFile, dstFile)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to create destination file"))
	})
})

var _ = Describe("copyFilesToBootFolder", func() {
	var tempDir string
	var bootFolder string

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "test-copy-files-to-boot")
		bootFolder = path.Join(tempDir, "boot")
		Expect(os.MkdirAll(bootFolder, 0755)).To(Succeed())
		// copyFilesToBootFolder expects the same layout as createFolders() already created
		Expect(os.MkdirAll(path.Join(bootFolder, artifactsFolder), 0755)).To(Succeed())
		Expect(os.MkdirAll(path.Join(bootFolder, bootLoaderFolder), 0755)).To(Succeed())
		// Sources are read from tempBootArtifactsFolder; reset it so tests do not see files from a prior case
		Expect(os.RemoveAll(tempBootArtifactsFolder)).To(Succeed())
		Expect(os.MkdirAll(tempBootArtifactsFolder, 0755)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		os.RemoveAll(tempBootArtifactsFolder)
	})

	It("successfully copies files", func() {
		// Create test files
		file1 := path.Join(tempBootArtifactsFolder, kernelFile)
		file2 := path.Join(tempBootArtifactsFolder, initrdFile)
		file3 := path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName)
		Expect(os.WriteFile(file1, []byte("kernel content"), 0600)).To(Succeed())
		Expect(os.WriteFile(file2, []byte("initrd content"), 0600)).To(Succeed())
		Expect(os.WriteFile(file3, []byte("bootloader config"), 0600)).To(Succeed())

		err := copyFilesToBootFolder(tempDir)
		Expect(err).To(BeNil())

		// Verify files exist in destination
		content1, err := os.ReadFile(path.Join(bootFolder, artifactsFolder, kernelFile))
		Expect(err).To(BeNil())
		Expect(string(content1)).To(Equal("kernel content"))

		content2, err := os.ReadFile(path.Join(bootFolder, artifactsFolder, initrdFile))
		Expect(err).To(BeNil())
		Expect(string(content2)).To(Equal("initrd content"))

		bootLoaderContent, err := os.ReadFile(path.Join(bootFolder, bootLoaderFolder, bootLoaderConfigFileName))
		Expect(err).To(BeNil())
		Expect(string(bootLoaderContent)).To(Equal("bootloader config"))
	})

	It("fails when source files do not exist", func() {
		// BeforeEach left tempBootArtifactsFolder empty (no vmlinuz/initrd/config)
		err := copyFilesToBootFolder(tempDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no such file or directory"))
	})
})

var _ = Describe("calculateBootArtifactsSize", func() {
	var tempDir string

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "test-calculate-size")

		Expect(os.MkdirAll(tempBootArtifactsFolder, 0755)).To(Succeed())

		// Create test files with known sizes
		kernelPath := path.Join(tempBootArtifactsFolder, kernelFile)
		initrdPath := path.Join(tempBootArtifactsFolder, initrdFile)
		bootLoaderPath := path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName)

		Expect(os.WriteFile(kernelPath, make([]byte, 1000), 0600)).To(Succeed())
		Expect(os.WriteFile(initrdPath, make([]byte, 2000), 0600)).To(Succeed())
		Expect(os.WriteFile(bootLoaderPath, make([]byte, 100), 0600)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		os.RemoveAll(tempBootArtifactsFolder)
	})

	It("calculates total size correctly", func() {
		totalSize, err := calculateBootArtifactsSize()
		Expect(err).To(BeNil())
		// 1000 (kernel) + 2000 (initrd) + 100 (bootloader config) = 3100
		Expect(totalSize).To(Equal(uint64(3100)))
	})

	It("fails when kernel file is missing", func() {
		os.Remove(path.Join(tempBootArtifactsFolder, kernelFile))
		_, err := calculateBootArtifactsSize()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to stat kernel file"))
	})

	It("fails when initrd file is missing", func() {
		os.Remove(path.Join(tempBootArtifactsFolder, initrdFile))
		_, err := calculateBootArtifactsSize()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to stat initrd file"))
	})

	It("fails when bootloader config file is missing", func() {
		os.Remove(path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName))
		_, err := calculateBootArtifactsSize()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to stat bootloader config file"))
	})
})
