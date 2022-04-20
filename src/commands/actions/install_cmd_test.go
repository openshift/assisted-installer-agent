package actions

import (
	"encoding/json"
	"strings"

	"github.com/spf13/afero"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/copier"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("installer test", func() {
	var installCommandLineString string
	var installCommandRequest models.InstallCmdRequest
	var oldConfig config.ConnectivityConfig
	var filesystem afero.Fs
	var agentConfig *config.AgentConfig

	getInstall := func(request models.InstallCmdRequest, filesystem afero.Fs, errorShouldOccur bool) *install {
		b, err := json.Marshal(&request)
		Expect(err).NotTo(HaveOccurred())
		action := &install{args: []string{string(b)}, filesystem: filesystem, agentConfig: agentConfig}
		err = action.Validate()
		if errorShouldOccur {
			Expect(err).To(HaveOccurred())
			return nil
		}
		Expect(err).NotTo(HaveOccurred())
		return action
	}

	getInstallCommandRequest := func() models.InstallCmdRequest {
		role := models.HostRoleBootstrap
		clusterId := strfmt.UUID("cd781f46-f32a-4154-9670-6442a367ab81")
		infraEnvId := strfmt.UUID("456eecf6-7aec-402d-b453-f609b19783cb")
		hostId := strfmt.UUID("f7ac1860-92cf-4ed8-aeec-2d9f20b35bab")
		bootDevice := "/dev/disk/by-path/pci-0000:00:06.0"
		return models.InstallCmdRequest{
			BootDevice:           swag.String(bootDevice),
			CheckCvo:             swag.Bool(true),
			ClusterID:            &clusterId,
			HostID:               &hostId,
			InfraEnvID:           &infraEnvId,
			ControllerImage:      swag.String("localhost:5000/edge-infrastructure/assisted-installer-controller:latest"),
			DisksToFormat:        []string{},
			HighAvailabilityMode: swag.String(models.ClusterHighAvailabilityModeFull),
			InstallerArgs:        "[\"--append-karg\",\"ip=ens3:dhcp\"]",
			InstallerImage:       swag.String("quay.io/edge-infrastructure/assisted-installer:latest"),
			McoImage:             "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4",
			MustGatherImage: "{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
				"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\"," +
				"\"ocp\":\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4\"," +
				"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}",
			OpenshiftVersion: "4.9.24",
			Role:             &role,
		}
	}

	getInstallCommandLineString := func(request models.InstallCmdRequest) string {
		b, err := json.Marshal(&request)
		Expect(err).NotTo(HaveOccurred())
		return string(b)
	}

	BeforeEach(func() {
		filesystem = afero.NewMemMapFs()
		agentConfig = &config.AgentConfig{}
		Expect(copier.Copy(&oldConfig, &agentConfig.ConnectivityConfig)).To(BeNil())
		agentConfig.AgentVersion = "quay.io/edge-infrastructure/assisted-installer-agent:latest"
		agentConfig.InsecureConnection = true
		agentConfig.TargetURL = "http://10.1.178.26:6000"
		bootDevice := "/dev/disk/by-path/pci-0000:00:06.0"
		err := filesystem.MkdirAll("/dev/disk/by-path", 0755)
		Expect(err).NotTo(HaveOccurred())
		err = afero.WriteFile(filesystem, bootDevice, []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		installCommandRequest = getInstallCommandRequest()
		installCommandLineString = getInstallCommandLineString(installCommandRequest)

	})
	AfterEach(func() {
		Expect(copier.Copy(&agentConfig.ConnectivityConfig, &oldConfig)).To(BeNil())
	})

	It("install bootstrap", func() {
		action := install{args: []string{installCommandLineString}, filesystem: filesystem, agentConfig: agentConfig}
		err := action.Validate()
		Expect(err).NotTo(HaveOccurred())

		args := action.Args()
		command := action.Command()
		Expect(command).To(Equal("sh"))
		paths := []string{
			"/var/log",
			"/run/systemd/journal/socket",
			"/dev",
			"/opt",
			"/etc/pki",
		}

		argsAsString := strings.Join(args, " ")
		verifyPaths(argsAsString, paths)
		Expect(argsAsString).To(ContainSubstring("--env=PULL_SECRET_TOKEN"))
		Expect(argsAsString).To(ContainSubstring("--role bootstrap --infra-env-id 456eecf6-7aec-402d-b453-f609b19783cb " +
			"--cluster-id cd781f46-f32a-4154-9670-6442a367ab81 --host-id f7ac1860-92cf-4ed8-aeec-2d9f20b35bab --boot-device /dev/disk/by-path/pci-0000:00:06.0 " +
			"--url http://10.1.178.26:6000 --controller-image localhost:5000/edge-infrastructure/assisted-installer-controller:latest " +
			"--agent-image quay.io/edge-infrastructure/assisted-installer-agent:latest " +
			"--high-availability-mode Full " +
			"--mco-image quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4 " +
			"--must-gather-image '{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\",\"ocp\":\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}' --openshift-version 4.9.24 --insecure --check-cluster-version --installer-args '[\"--append-karg\",\"ip=ens3:dhcp\"]'"))
	})

	It("ha mode is nil, parameter should be omitted", func() {
		installCommandRequest.HighAvailabilityMode = nil
		installCommandLineString = getInstallCommandLineString(installCommandRequest)

		action := install{args: []string{installCommandLineString}, filesystem: filesystem, agentConfig: agentConfig}
		validationError := action.Validate()
		Expect(validationError).NotTo(HaveOccurred())

		args := action.Args()
		argsAsString := strings.Join(args, " ")
		Expect(argsAsString).NotTo(ContainSubstring("--high-availability-mode"))
	})

	It("ha mode is Full, command line parameter should indicate this", func() {
		installCommandRequest.HighAvailabilityMode = swag.String(models.ClusterHighAvailabilityModeFull)
		installCommandLineString = getInstallCommandLineString(installCommandRequest)

		action := install{args: []string{installCommandLineString}, filesystem: filesystem, agentConfig: agentConfig}
		validationError := action.Validate()
		Expect(validationError).NotTo(HaveOccurred())

		args := action.Args()
		argsAsString := strings.Join(args, " ")
		Expect(argsAsString).To(ContainSubstring("--high-availability-mode Full"))
	})

	It("ha mode is None, command line parameter should indicate this", func() {
		installCommandRequest.HighAvailabilityMode = swag.String(models.ClusterHighAvailabilityModeNone)
		installCommandLineString = getInstallCommandLineString(installCommandRequest)

		action := install{args: []string{installCommandLineString}, filesystem: filesystem, agentConfig: agentConfig}
		validationError := action.Validate()
		Expect(validationError).NotTo(HaveOccurred())

		args := action.Args()
		argsAsString := strings.Join(args, " ")
		Expect(argsAsString).To(ContainSubstring("--high-availability-mode None"))
	})

	It("install ca cert", func() {
		caPath := "/ca_cert"
		agentConfig.CACertificatePath = caPath
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		paths := []string{
			"/ca_cert",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(strings.Join(args, " ")).To(ContainSubstring("--cacert /ca_cert"))
	})

	It("install no mco, must-gather and openshift version", func() {
		installCommandRequest.McoImage = ""
		installCommandRequest.MustGatherImage = ""
		installCommandRequest.OpenshiftVersion = ""
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("mco-image"))
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("must-gather-image"))
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("openshift-version"))
	})

	It("install with single must-gather", func() {
		installCommandRequest.MustGatherImage = "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:d10836be954321a7583d7388498807872bf804d3f2840cbec9100264dd01c165"
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).To(ContainSubstring("must-gather-image"))
	})

	It("install with disks to format", func() {
		err := afero.WriteFile(filesystem, "/dev/sda", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		err = afero.WriteFile(filesystem, "/dev/sdb", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		installCommandRequest.DisksToFormat = []string{"/dev/sda", "/dev/sdb"}
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--format-disk /dev/sda"))
		Expect(strings.Join(args, " ")).To(ContainSubstring("--format-disk /dev/sdb"))
	})

	It("install with bad disks", func() {
		By("No dev as prefix")
		err := afero.WriteFile(filesystem, "/dev/sda", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		err = afero.WriteFile(filesystem, "/dev/sdb", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		installCommandRequest.DisksToFormat = []string{"sda", "/dev/sdb"}
		_ = getInstall(installCommandRequest, filesystem, true)

		By("pipe after disk")
		err = afero.WriteFile(filesystem, "/dev/sdb", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		installCommandRequest.BootDevice = swag.String("/dev/sdb|echo")
		_ = getInstall(installCommandRequest, filesystem, true)

		By("disk path doesn't exists")
		installCommandRequest.DisksToFormat = []string{"/dev/sdb"}
		_ = getInstall(installCommandRequest, filesystem, true)

	})

	It("install insecure and cvo is false", func() {
		agentConfig.InsecureConnection = false
		installCommandRequest.CheckCvo = swag.Bool(false)
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--insecure"))
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--check-cluster-version"))
	})

	It("install no installer args", func() {
		installCommandRequest.InstallerArgs = ""
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--installer-args"))
	})

	It("install bad installer args", func() {
		installCommandRequest.InstallerArgs = "[\"--append-karg\",\"ip=ens3:dhcp|dsadsa\"]"
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("install with service ips", func() {
		installCommandRequest.ServiceIps = []string{"192.168.2.1", "192.168.3.1"}
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--service-ips 192.168.2.1,192.168.3.1"))
	})

	It("install with bad service ips", func() {
		By("ip with pipe")
		installCommandRequest.ServiceIps = []string{"192.168.2.1|aaaa", "192.168.3.1"}
		_ = getInstall(installCommandRequest, filesystem, true)

		By("bad ip")
		installCommandRequest.ServiceIps = []string{"aaaa", "192.168.3.1"}
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("install with proxy", func() {
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--http-proxy http://192.0.0.1 --https-proxy http://192.0.0.2 --no-proxy domain.org,127.0.0.2,127.0.0.1,localhost"))
	})

	It("install without https proxy", func() {
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String(""),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--http-proxy http://192.0.0.1 --no-proxy domain.org,127.0.0.2,127.0.0.1,localhost"))
	})

	It("install with bad proxy", func() {
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(installCommandRequest, filesystem, true)

		By("Bad https proxy")
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(installCommandRequest, filesystem, true)

		By("Bad no proxy")
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("https://192.0.0.2"),
			NoProxy:    swag.String("domain.org,echo,127.0.0.1,localhost"),
		}
		_ = getInstall(installCommandRequest, filesystem, true)

	})

	It("install with bad proxy with pipe", func() {
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1|ecoh"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(installCommandRequest, filesystem, true)

		By("Bad https proxy with pipe")
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2|ecoh"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(installCommandRequest, filesystem, true)

		By("Bad no proxy with pipe")
		installCommandRequest.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost|ecoh"),
		}
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("bad must-gather image - bad operator", func() {
		installCommandRequest.MustGatherImage = "{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\"," +
			"\"rm\":\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}"
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("bad must-gather image - bad image", func() {
		installCommandRequest.MustGatherImage = "{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\"," +
			"\"ocp\":\"echo aaaa\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}"
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("bad must-gather image - bad image with pipe", func() {
		installCommandRequest.MustGatherImage = "{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\"," +
			"\"ocp\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8|rm\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}"
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("bad HighAvailability", func() {
		installCommandRequest.HighAvailabilityMode = swag.String("some string")
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("bad openshift version - some string", func() {
		installCommandRequest.OpenshiftVersion = "some string"
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("bad openshift version - with pipe", func() {
		installCommandRequest.OpenshiftVersion = "4.10|"
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("good version", func() {
		installCommandRequest.OpenshiftVersion = "4.10-rc-6-prelease"
		action := getInstall(installCommandRequest, filesystem, false)
		args := action.Args()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--openshift-version 4.10-rc-6-prelease"))
	})

	It("bad images installer", func() {
		installCommandRequest.InstallerImage = swag.String("quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:aaa;echo")
		_ = getInstall(installCommandRequest, filesystem, true)
	})

	It("bad images controller", func() {
		installCommandRequest.ControllerImage = swag.String("echo:111|rm")
		_ = getInstall(installCommandRequest, filesystem, true)

	})

})
