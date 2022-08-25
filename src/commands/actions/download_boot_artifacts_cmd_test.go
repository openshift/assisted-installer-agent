package actions

import (
	"os"
	"path"

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
	var (
		tempDir             string
		hostArtifactsFolder string
		bootFolder          string
	)

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "download-boot-artifacts")
		hostArtifactsFolder = path.Join(tempDir, artifactsFolder)
		bootFolder = path.Join(tempDir, "loader")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("Successful folder creation", func() {
		By("artifacts folder should not exist before creation", func() {
			_, err := os.Stat(hostArtifactsFolder)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		By("boot loader config folder should not exist before creation", func() {
			_, err := os.Stat(bootFolder)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		By("creating both folders", func() {
			err := createFolders(hostArtifactsFolder, bootFolder)
			Expect(err).To(BeNil())
			folder, err := os.Stat(hostArtifactsFolder)
			Expect(err).To(BeNil())
			Expect(folder.IsDir()).To(BeTrue())
			folder, err = os.Stat(bootFolder)
			Expect(err).To(BeNil())
			Expect(folder.IsDir()).To(BeTrue())
		})
	})
})
var _ = Describe("createBootLoaderConfig", func() {
	var (
		tempDir    string
		bootFile   string
		bootFolder string
	)

	const (
		rootfsUrl = "http://test.com/rootfs?arch=x86_64&version=4.10"
	)

	BeforeEach(func() {
		tempDir = path.Join(os.TempDir(), "download-boot-artifacts")
		bootFolder = path.Join(tempDir, "loader")
		bootFile = path.Join(bootFolder, "00-assisted-discovery.conf")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("Successful bootloader config creation", func() {
		err := os.MkdirAll(bootFolder, 0777)
		Expect(err).To(BeNil())
		err = createBootLoaderConfig(rootfsUrl, artifactsFolder, bootFolder)
		Expect(err).To(BeNil())
		bootConfigContents, err := os.ReadFile(bootFile)
		Expect(err).To(BeNil())
		Expect(string(bootConfigContents)).To(ContainSubstring(path.Join(artifactsFolder, kernelFile)))
		Expect(string(bootConfigContents)).To(ContainSubstring(path.Join(artifactsFolder, initrdFile)))
		Expect(string(bootConfigContents)).To(ContainSubstring(rootfsUrl))
	})

	It("Failed bootloader config creation - folder DNE", func() {
		err := createBootLoaderConfig(rootfsUrl, artifactsFolder, bootFolder)
		Expect(err).To(HaveOccurred())
	})

	It("Bad bootloader config - incorrect artifacts folder", func() {
		err := os.MkdirAll(bootFolder, 0777)
		Expect(err).To(BeNil())
		incorrectFolder := "/incorrect/folder"
		err = createBootLoaderConfig(rootfsUrl, incorrectFolder, bootFolder)
		Expect(err).To(BeNil())
		bootConfigContents, err := os.ReadFile(bootFile)
		Expect(err).To(BeNil())
		Expect(string(bootConfigContents)).ToNot(ContainSubstring(path.Join(artifactsFolder, kernelFile)))
		Expect(string(bootConfigContents)).ToNot(ContainSubstring(path.Join(artifactsFolder, initrdFile)))
	})

	It("Bad bootloader config - incorrect rootfs URL", func() {
		err := os.MkdirAll(bootFolder, 0777)
		Expect(err).To(BeNil())
		err = createBootLoaderConfig("http://example.com/not-rootfs-url", artifactsFolder, bootFolder)
		Expect(err).To(BeNil())
		bootConfigContents, err := os.ReadFile(bootFile)
		Expect(err).To(BeNil())
		Expect(string(bootConfigContents)).ToNot(ContainSubstring(rootfsUrl))
	})
})
