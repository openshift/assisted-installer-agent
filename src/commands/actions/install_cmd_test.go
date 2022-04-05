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

func getInstall(args models.InstallCmdRequest, filesystem afero.Fs, errorShouldOccur bool) *install {
	b, err := json.Marshal(&args)
	Expect(err).NotTo(HaveOccurred())
	action := &install{args: []string{string(b)}, filesystem: filesystem}
	err = action.Validate()
	if errorShouldOccur {
		Expect(err).To(HaveOccurred())
		return nil
	}

	Expect(err).NotTo(HaveOccurred())
	return action
}

var _ = Describe("installer test", func() {
	var params string
	var args models.InstallCmdRequest
	var oldConfig config.ConnectivityConfig
	var filesystem afero.Fs

	BeforeEach(func() {
		filesystem = afero.NewMemMapFs()
		Expect(copier.Copy(&oldConfig, &config.GlobalAgentConfig.ConnectivityConfig)).To(BeNil())
		config.GlobalAgentConfig.AgentVersion = "quay.io/edge-infrastructure/assisted-installer-agent:latest"
		config.GlobalAgentConfig.InsecureConnection = true
		config.GlobalAgentConfig.TargetURL = "http://10.1.178.26:6000"
		clusterId := strfmt.UUID("cd781f46-f32a-4154-9670-6442a367ab81")
		hostId := strfmt.UUID("f7ac1860-92cf-4ed8-aeec-2d9f20b35bab")
		infraEnvId := strfmt.UUID("456eecf6-7aec-402d-b453-f609b19783cb")
		role := models.HostRoleBootstrap
		bootDevice := "/dev/disk/by-path/pci-0000:00:06.0"
		err := filesystem.MkdirAll("/dev/disk/by-path", 0755)
		Expect(err).NotTo(HaveOccurred())
		err = afero.WriteFile(filesystem, bootDevice, []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		args = models.InstallCmdRequest{
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
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		params = string(b)

	})
	AfterEach(func() {
		Expect(copier.Copy(&config.GlobalAgentConfig.ConnectivityConfig, &oldConfig)).To(BeNil())
	})

	It("install bootstrap", func() {
		action := install{args: []string{params}, filesystem: filesystem}
		err := action.Validate()
		Expect(err).NotTo(HaveOccurred())

		command, args := action.CreateCmd()
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
			"--url http://10.1.178.26:6000 --high-availability-mode Full --controller-image localhost:5000/edge-infrastructure/assisted-installer-controller:latest " +
			"--agent-image quay.io/edge-infrastructure/assisted-installer-agent:latest " +
			"--mco-image quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4 " +
			"--must-gather-image '{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\",\"ocp\":\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}' --openshift-version 4.9.24 --insecure --check-cluster-version --installer-args '[\"--append-karg\",\"ip=ens3:dhcp\"]'"))
	})
	It("install ca cert", func() {
		caPath := "/ca_cert"
		config.GlobalAgentConfig.CACertificatePath = caPath
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		paths := []string{
			"/ca_cert",
		}
		verifyPaths(strings.Join(args, " "), paths)
		Expect(strings.Join(args, " ")).To(ContainSubstring("--cacert /ca_cert"))
	})

	It("install no mco, must-gather and openshift version", func() {
		args.McoImage = ""
		args.MustGatherImage = ""
		args.OpenshiftVersion = ""
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("mco-image"))
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("must-gather-image"))
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("openshift-version"))
	})

	It("install with single must-gather", func() {
		args.MustGatherImage = "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:d10836be954321a7583d7388498807872bf804d3f2840cbec9100264dd01c165"
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("must-gather-image"))
	})

	It("install with disks to format", func() {
		err := afero.WriteFile(filesystem, "/dev/sda", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		err = afero.WriteFile(filesystem, "/dev/sdb", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		args.DisksToFormat = []string{"/dev/sda", "/dev/sdb"}
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--format-disk /dev/sda"))
		Expect(strings.Join(args, " ")).To(ContainSubstring("--format-disk /dev/sdb"))
	})

	It("install with bad disks", func() {
		By("No dev as prefix")
		err := afero.WriteFile(filesystem, "/dev/sda", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		err = afero.WriteFile(filesystem, "/dev/sdb", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		args.DisksToFormat = []string{"sda", "/dev/sdb"}
		_ = getInstall(args, filesystem, true)

		By("pipe after disk")
		err = afero.WriteFile(filesystem, "/dev/sdb", []byte("a file"), 0755)
		Expect(err).NotTo(HaveOccurred())
		args.BootDevice = swag.String("/dev/sdb|echo")
		_ = getInstall(args, filesystem, true)

		By("disk path doesn't exists")
		args.DisksToFormat = []string{"/dev/sdb"}
		_ = getInstall(args, filesystem, true)

	})

	It("install insecure and cvo is false", func() {
		config.GlobalAgentConfig.InsecureConnection = false
		args.CheckCvo = swag.Bool(false)
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--insecure"))
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--check-cluster-version"))
	})

	It("install no installer args", func() {
		args.InstallerArgs = ""
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--installer-args"))
	})

	It("install bad installer args", func() {
		args.InstallerArgs = "[\"--append-karg\",\"ip=ens3:dhcp|dsadsa\"]"
		_ = getInstall(args, filesystem, true)
	})

	It("install with service ips", func() {
		args.ServiceIps = []string{"192.168.2.1", "192.168.3.1"}
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--service-ips 192.168.2.1,192.168.3.1"))
	})

	It("install with bad service ips", func() {
		By("ip with pipe")
		args.ServiceIps = []string{"192.168.2.1|aaaa", "192.168.3.1"}
		_ = getInstall(args, filesystem, true)

		By("bad ip")
		args.ServiceIps = []string{"aaaa", "192.168.3.1"}
		_ = getInstall(args, filesystem, true)
	})

	It("install with proxy", func() {
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--http-proxy http://192.0.0.1 --https-proxy http://192.0.0.2 --no-proxy domain.org,127.0.0.2,127.0.0.1,localhost"))
	})

	It("install without https proxy", func() {
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String(""),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--http-proxy http://192.0.0.1 --no-proxy domain.org,127.0.0.2,127.0.0.1,localhost"))
	})

	It("install with bad proxy", func() {
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(args, filesystem, true)

		By("Bad https proxy")
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(args, filesystem, true)

		By("Bad no proxy")
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("https://192.0.0.2"),
			NoProxy:    swag.String("domain.org,echo,127.0.0.1,localhost"),
		}
		_ = getInstall(args, filesystem, true)

	})

	It("install with bad proxy with pipe", func() {
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1|ecoh"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(args, filesystem, true)

		By("Bad https proxy with pipe")
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2|ecoh"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		_ = getInstall(args, filesystem, true)

		By("Bad no proxy with pipe")
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("http://192.0.0.1"),
			HTTPSProxy: swag.String("http://192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost|ecoh"),
		}
		_ = getInstall(args, filesystem, true)
	})

	It("bad must-gather image - bad operator", func() {
		args.MustGatherImage = "{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\"," +
			"\"rm\":\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}"
		_ = getInstall(args, filesystem, true)
	})

	It("bad must-gather image - bad image", func() {
		args.MustGatherImage = "{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\"," +
			"\"ocp\":\"echo aaaa\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}"
		_ = getInstall(args, filesystem, true)
	})

	It("bad must-gather image - bad image with pipe", func() {
		args.MustGatherImage = "{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\"," +
			"\"ocp\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8|rm\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}"
		_ = getInstall(args, filesystem, true)
	})

	It("bad HighAvailability", func() {
		args.HighAvailabilityMode = swag.String("some string")
		_ = getInstall(args, filesystem, true)
	})

	It("bad openshift version - some string", func() {
		args.OpenshiftVersion = "some string"
		_ = getInstall(args, filesystem, true)
	})

	It("bad openshift version - with pipe", func() {
		args.OpenshiftVersion = "4.10|"
		_ = getInstall(args, filesystem, true)
	})

	It("good version", func() {
		args.OpenshiftVersion = "4.10-rc-6-prelease"
		action := getInstall(args, filesystem, false)
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--openshift-version 4.10-rc-6-prelease"))
	})

	It("bad images mco", func() {
		args.McoImage = "echo"
		_ = getInstall(args, filesystem, true)
	})

	It("bad images installer", func() {
		args.InstallerImage = swag.String("quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:aaa;echo")
		_ = getInstall(args, filesystem, true)
	})

	It("bad images controller", func() {
		args.ControllerImage = swag.String("echo:111|rm")
		_ = getInstall(args, filesystem, true)

	})

})
