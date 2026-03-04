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
			folders, err := createFolders(hostDir, defaultTestRetryAmount)
			Expect(err).To(BeNil())
			Expect(folders.hostArtifactsFolder).To(Equal(path.Join(bootDir, artifactsFolder)))
			Expect(folders.bootLoaderFolder).To(Equal(path.Join(bootDir, "loader", "entries")))
			Expect(folders.bootFolder).To(Equal(bootDir))
			Expect(folders.tempDownloadFolder).To(Equal(path.Join(tempBootArtifactsFolder, artifactsFolder)))
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
		err := createBootLoaderConfig(rootfsUrl)
		Expect(err).To(BeNil())
		bootConfigContents, err := os.ReadFile(bootFile)
		Expect(err).To(BeNil())
		Expect(string(bootConfigContents)).To(ContainSubstring(path.Join(artifactsFolder, kernelFile)))
		Expect(string(bootConfigContents)).To(ContainSubstring(path.Join(artifactsFolder, initrdFile)))
		Expect(string(bootConfigContents)).To(ContainSubstring(rootfsUrl))
	})

	It("Failed bootloader config creation - folder DNE", func() {
		os.RemoveAll(tempBootArtifactsFolder) // remove so write fails
		err := createBootLoaderConfig(rootfsUrl)
		Expect(err).To(HaveOccurred())
	})

	It("Bad bootloader config - incorrect rootfs URL", func() {
		err := createBootLoaderConfig("http://example.com/not-rootfs-url")
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

var _ = Describe("moveFiles", func() {
	var tempDir string
	var srcDir string
	var dstDir string

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "test-move-files")
		srcDir = path.Join(tempDir, "source")
		dstDir = path.Join(tempDir, "destination")

		Expect(os.MkdirAll(srcDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(dstDir, 0755)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("successfully moves files", func() {
		// Create test files
		file1 := path.Join(srcDir, "file1.txt")
		file2 := path.Join(srcDir, "file2.txt")
		Expect(os.WriteFile(file1, []byte("content1"), 0600)).To(Succeed())
		Expect(os.WriteFile(file2, []byte("content2"), 0600)).To(Succeed())

		err := moveFiles(srcDir, dstDir)
		Expect(err).To(BeNil())

		// Verify files exist in destination
		content1, err := os.ReadFile(path.Join(dstDir, "file1.txt"))
		Expect(err).To(BeNil())
		Expect(string(content1)).To(Equal("content1"))

		content2, err := os.ReadFile(path.Join(dstDir, "file2.txt"))
		Expect(err).To(BeNil())
		Expect(string(content2)).To(Equal("content2"))
	})

	It("skips directories", func() {
		// Create a subdirectory in source
		subDir := path.Join(srcDir, "subdir")
		Expect(os.MkdirAll(subDir, 0755)).To(Succeed())

		// Create a file
		file1 := path.Join(srcDir, "file1.txt")
		Expect(os.WriteFile(file1, []byte("content1"), 0600)).To(Succeed())

		err := moveFiles(srcDir, dstDir)
		Expect(err).To(BeNil())

		// Verify file is moved but directory is not
		_, err = os.Stat(path.Join(dstDir, "file1.txt"))
		Expect(err).To(BeNil())

		_, err = os.Stat(path.Join(dstDir, "subdir"))
		Expect(err).To(HaveOccurred())
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("fails when source directory does not exist", func() {
		err := moveFiles("/nonexistent-dir", dstDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to read"))
	})
})

var _ = Describe("getFreeSpace", func() {
	It("returns free space for an existing directory", func() {
		tempDir := os.TempDir()
		_, err := os.Create(path.Join(tempDir, "test-file.txt"))
		Expect(err).To(BeNil())
		freeSpace, err := getFreeSpace(tempDir)
		Expect(err).To(BeNil())
		Expect(freeSpace).To(BeNumerically(">", 0))
	})

	It("fails when directory does not exist", func() {
		_, err := getFreeSpace("/nonexistent-directory-xyz")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("calculateBootArtifactsSize", func() {
	var tempDir string
	var testFolders *folders

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "test-calculate-size")
		tempDownloadDir := path.Join(tempDir, "download")

		Expect(os.MkdirAll(tempDownloadDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(tempBootArtifactsFolder, 0755)).To(Succeed())

		testFolders = &folders{
			tempDownloadFolder: tempDownloadDir,
		}

		// Create test files with known sizes
		kernelPath := path.Join(tempDownloadDir, kernelFile)
		initrdPath := path.Join(tempDownloadDir, initrdFile)
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
		totalSize, err := calculateBootArtifactsSize(testFolders)
		Expect(err).To(BeNil())
		// 1000 (kernel) + 2000 (initrd) + 100 (bootloader config) = 3100
		Expect(totalSize).To(Equal(uint64(3100)))
	})

	It("fails when kernel file is missing", func() {
		os.Remove(path.Join(testFolders.tempDownloadFolder, kernelFile))
		_, err := calculateBootArtifactsSize(testFolders)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to stat kernel file"))
	})

	It("fails when initrd file is missing", func() {
		os.Remove(path.Join(testFolders.tempDownloadFolder, initrdFile))
		_, err := calculateBootArtifactsSize(testFolders)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to stat initrd file"))
	})

	It("fails when bootloader config file is missing", func() {
		os.Remove(path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName))
		_, err := calculateBootArtifactsSize(testFolders)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to stat bootloader config file"))
	})
})

var _ = Describe("moveFilesToBootFolder", func() {
	var tempDir string
	var testFolders *folders

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "test-move-to-boot")
		tempDownloadDir := path.Join(tempDir, "download")
		hostArtifactsDir := path.Join(tempDir, "host-artifacts")
		bootLoaderDir := path.Join(tempDir, "boot-loader")

		Expect(os.MkdirAll(tempDownloadDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(hostArtifactsDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(bootLoaderDir, 0755)).To(Succeed())
		Expect(os.MkdirAll(tempBootArtifactsFolder, 0755)).To(Succeed())

		testFolders = &folders{
			tempDownloadFolder:  tempDownloadDir,
			hostArtifactsFolder: hostArtifactsDir,
			bootLoaderFolder:    bootLoaderDir,
		}

		// Create test files
		kernelPath := path.Join(tempDownloadDir, kernelFile)
		initrdPath := path.Join(tempDownloadDir, initrdFile)
		bootLoaderPath := path.Join(tempBootArtifactsFolder, bootLoaderConfigFileName)

		Expect(os.WriteFile(kernelPath, []byte("kernel content"), 0600)).To(Succeed())
		Expect(os.WriteFile(initrdPath, []byte("initrd content"), 0600)).To(Succeed())
		Expect(os.WriteFile(bootLoaderPath, []byte("bootloader config"), 0600)).To(Succeed())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		os.RemoveAll(tempBootArtifactsFolder)
	})

	It("successfully moves files to boot folder", func() {
		err := moveFilesToBootFolder(testFolders)
		Expect(err).To(BeNil())

		// Verify artifacts moved
		kernelContent, err := os.ReadFile(path.Join(testFolders.hostArtifactsFolder, kernelFile))
		Expect(err).To(BeNil())
		Expect(string(kernelContent)).To(Equal("kernel content"))

		initrdContent, err := os.ReadFile(path.Join(testFolders.hostArtifactsFolder, initrdFile))
		Expect(err).To(BeNil())
		Expect(string(initrdContent)).To(Equal("initrd content"))

		// Verify bootloader config moved
		bootLoaderContent, err := os.ReadFile(path.Join(testFolders.bootLoaderFolder, bootLoaderConfigFileName))
		Expect(err).To(BeNil())
		Expect(string(bootLoaderContent)).To(Equal("bootloader config"))

		// Verify success marker file created
		_, err = os.Stat(path.Join(tempBootArtifactsFolder, "boot_artifacts_moved"))
		Expect(err).To(BeNil())
	})

	It("fails when source directory does not exist", func() {
		testFolders.tempDownloadFolder = "/nonexistent-dir"
		err := moveFilesToBootFolder(testFolders)
		Expect(err).To(HaveOccurred())
	})
})
