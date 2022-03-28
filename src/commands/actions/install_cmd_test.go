package actions

import (
	"encoding/json"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/copier"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("installer test", func() {
	var params string
	var args models.InstallCmdRequest
	var oldConfig config.ConnectivityConfig

	BeforeEach(func() {
		Expect(copier.Copy(&oldConfig, &config.GlobalAgentConfig.ConnectivityConfig)).To(BeNil())
		config.GlobalAgentConfig.AgentVersion = "quay.io/edge-infrastructure/assisted-installer-agent:latest"
		config.GlobalAgentConfig.InsecureConnection = true
		config.GlobalAgentConfig.TargetURL = "http://10.1.178.26:6000"
		clusterId := strfmt.UUID("cd781f46-f32a-4154-9670-6442a367ab81")
		hostId := strfmt.UUID("f7ac1860-92cf-4ed8-aeec-2d9f20b35bab")
		infraEnvId := strfmt.UUID("456eecf6-7aec-402d-b453-f609b19783cb")
		role := models.HostRoleBootstrap
		args = models.InstallCmdRequest{
			BootDevice:           swag.String("/dev/disk/by-path/pci-0000:00:06.0"),
			CheckCvo:             swag.Bool(true),
			ClusterID:            &clusterId,
			HostID:               &hostId,
			InfraEnvID:           &infraEnvId,
			ControllerImage:      swag.String("quay.io/edge-infrastructure/assisted-installer-controller:latest"),
			DisksToFormat:        []string{},
			HighAvailabilityMode: swag.String(models.ClusterHighAvailabilityModeFull),
			InstallerArgs:        "[\"--append-karg\",\"ip=ens3:dhcp\"]",
			InstallerImage:       swag.String("quay.io/edge-infrastructure/assisted-installer:latest"),
			McoImage:             "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:aaa",
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
		action, err := New(models.StepTypeInstall, []string{params})
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
			"--url http://10.1.178.26:6000 --high-availability-mode Full --controller-image quay.io/edge-infrastructure/assisted-installer-controller:latest " +
			"--agent-image quay.io/edge-infrastructure/assisted-installer-agent:latest " +
			"--mco-image quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:aaa " +
			"--must-gather-image '{\"cnv\":\"registry.redhat.io/container-native-virtualization/cnv-must-gather-rhel8:v2.6.5\"," +
			"\"lso\":\"registry.redhat.io/openshift4/ose-local-storage-mustgather-rhel8\",\"ocp\":\"quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:3c30f115dc95c3fef94ea5185f386aa1af8a4b5f07ce8f41a17007d54004e1c4\"," +
			"\"ocs\":\"registry.redhat.io/ocs4/ocs-must-gather-rhel8\"}' --openshift-version 4.9.24 --insecure --check-cluster-version --installer-args '[\"--append-karg\",\"ip=ens3:dhcp\"]'"))
	})
	It("install ca cert", func() {
		caPath := "/ca_cert"
		config.GlobalAgentConfig.CACertificatePath = caPath
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())

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
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("mco-image"))
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("must-gather-image"))
		Expect(strings.Join(args, " ")).ToNot(ContainSubstring("openshift-version"))
	})

	It("install with disks to format", func() {
		args.DisksToFormat = []string{"/dev/sda", "/dev/sdb"}
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--format-disk /dev/sda"))
		Expect(strings.Join(args, " ")).To(ContainSubstring("--format-disk /dev/sdb"))
	})

	It("install insecure and cvo is false", func() {
		config.GlobalAgentConfig.InsecureConnection = false
		args.CheckCvo = swag.Bool(false)
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--insecure"))
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--check-cluster-version"))
	})

	It("install no installer args", func() {
		args.InstallerArgs = ""
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).NotTo(ContainSubstring("--installer-args"))
	})

	It("install with service ips", func() {
		args.ServiceIps = []string{"192.168.2.1", "192.168.3.1"}
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--service-ips 192.168.2.1,192.168.3.1"))
	})

	It("install with proxy", func() {
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("192.0.0.1"),
			HTTPSProxy: swag.String("192.0.0.2"),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--http-proxy 192.0.0.1 --https-proxy 192.0.0.2 --no-proxy domain.org,127.0.0.2,127.0.0.1,localhost"))
	})

	It("install without https proxy", func() {
		args.Proxy = &models.Proxy{
			HTTPProxy:  swag.String("192.0.0.1"),
			HTTPSProxy: swag.String(""),
			NoProxy:    swag.String("domain.org,127.0.0.2,127.0.0.1,localhost"),
		}
		b, err := json.Marshal(&args)
		Expect(err).NotTo(HaveOccurred())
		action, err := New(models.StepTypeInstall, []string{string(b)})
		Expect(err).NotTo(HaveOccurred())
		_, args := action.CreateCmd()
		Expect(strings.Join(args, " ")).To(ContainSubstring("--http-proxy 192.0.0.1 --no-proxy domain.org,127.0.0.2,127.0.0.1,localhost"))
	})
})