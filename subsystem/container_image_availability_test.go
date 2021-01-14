package subsystem

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
)

var _ = Describe("Image availability tests", func() {
	var (
		hostID               string
		pullTimeoutInSeconds int64
		images               []string
	)

	deleteImage := func(image string) {
		_ = removeImage(defaultContainerTool, image)
		Expect(isImageAvailable(defaultContainerTool, image)).Should(BeFalse())
	}

	checkImageStatus := func(image *models.ContainerImageAvailability, expectedResult bool, isRemoteImage bool) {
		Expect(images).Should(ContainElement(image.Name))
		Expect(isImageAvailable(defaultContainerTool, image.Name)).Should(Equal(expectedResult))

		if expectedResult {
			Expect(image.Result).Should(Equal(models.ContainerImageAvailabilityResultSuccess))
		} else {
			Expect(image.Result).Should(Equal(models.ContainerImageAvailabilityResultFailure))
		}

		if expectedResult && isRemoteImage {
			Expect(image.Time).Should(BeNumerically(">", 0))
			Expect(image.DownloadRate).Should(BeNumerically(">", 0))
			Expect(image.SizeBytes).Should(BeNumerically(">", 0))
		} else { // failure or local image
			Expect(image.DownloadRate).Should(BeZero())
			Expect(image.Time).Should(BeZero())
			Expect(image.SizeBytes).Should(BeZero())
		}
	}

	BeforeEach(func() {
		resetAll()
		hostID = nextHostID()

		pullTimeoutInSeconds = 60
	})

	AfterEach(func() {
		for _, image := range images {
			deleteImage(image)
		}
	})

	It("Valid new images", func() {
		images = []string{"quay.io/aptible/hello-world", "quay.io/coreos/etcd:latest"}

		for _, image := range images {
			deleteImage(image)
		}

		startImageAvailability(hostID, models.ContainerImageAvailabilityRequest{Images: images, Timeout: pullTimeoutInSeconds})

		response := getImageAvailabilityResponse(hostID)
		Expect(response).ShouldNot(BeNil())
		Expect(response.Images).Should(HaveLen(len(images)))
		for _, image := range response.Images {
			checkImageStatus(image, true, true)
			Expect(image.Time).Should(BeNumerically("<", pullTimeoutInSeconds))
		}
	})

	It("Already downloaded image", func() {
		images = []string{"quay.io/aptible/hello-world"}

		for _, image := range images {
			deleteImage(image)
			Expect(pullImage(defaultContainerTool, image)).Should(BeTrue())
			Expect(isImageAvailable(defaultContainerTool, image)).Should(BeTrue())
		}

		startImageAvailability(hostID, models.ContainerImageAvailabilityRequest{Images: images, Timeout: pullTimeoutInSeconds})

		response := getImageAvailabilityResponse(hostID)
		Expect(response).ShouldNot(BeNil())
		Expect(response.Images).Should(HaveLen(len(images)))
		for _, image := range response.Images {
			checkImageStatus(image, true, false)
			Expect(image.Time).Should(BeNumerically("<", pullTimeoutInSeconds))
		}
	})

	It("Small timeout", func() {
		images = []string{"quay.io/coreos/etcd:latest"}
		pullTimeoutInSeconds = 2

		for _, image := range images {
			deleteImage(image)
		}

		startImageAvailability(hostID, models.ContainerImageAvailabilityRequest{Images: images, Timeout: pullTimeoutInSeconds})

		response := getImageAvailabilityResponse(hostID)
		Expect(response).ShouldNot(BeNil())
		Expect(response.Images).Should(HaveLen(len(images)))
		for _, image := range response.Images {
			checkImageStatus(image, false, true)
		}
	})

	It("Invalid image", func() {
		images = []string{"invalid-registry/invalid-repository/image:tag"}
		startImageAvailability(hostID, models.ContainerImageAvailabilityRequest{Images: images, Timeout: pullTimeoutInSeconds})

		response := getImageAvailabilityResponse(hostID)
		Expect(response).ShouldNot(BeNil())
		Expect(response.Images).Should(HaveLen(len(images)))
		for _, image := range response.Images {
			checkImageStatus(image, false, true)
			Expect(image.Time).Should(BeNumerically("<=", 5))
		}
	})
})

func startImageAvailability(hostId string, request models.ContainerImageAvailabilityRequest) {
	_, err := addRegisterStub(hostId, http.StatusCreated, ClusterID)
	Expect(err).ShouldNot(HaveOccurred())
	setImageAvailabilityStub(hostId, request)
	setReplyStartAgent(hostId)
}

func setImageAvailabilityStub(hostID string, request models.ContainerImageAvailabilityRequest) {
	b, err := json.Marshal(&request)
	Expect(err).ShouldNot(HaveOccurred())

	step := generateContainerStep(models.StepTypeContainerImageAvailability,
		[]string{
			"-v", "/usr/bin/docker:/usr/bin/podman", // Overrides podman with docker
			"-v", "/var/run/docker.sock:/var/run/docker.sock",
		},
		[]string{"/usr/bin/container_image_availability", "--request", string(b)})
	_, err = addNextStepStub(hostID, 10, "", step)
	Expect(err).NotTo(HaveOccurred())
}

func getImageAvailabilityResponse(hostID string) *models.ContainerImageAvailabilityResponse {
	Eventually(func() bool {
		return isReplyFound(hostID, &ImageAVailabilityVerifier{})
	}, maxTimeout, 5*time.Second).Should(BeTrue())

	stepReply := getSpecificStep(hostID, &ImageAVailabilityVerifier{})
	return getImageAvailabilityResponseFromStepReply(stepReply)
}

func getImageAvailabilityResponseFromStepReply(actualReply *models.StepReply) *models.ContainerImageAvailabilityResponse {
	var response models.ContainerImageAvailabilityResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	Expect(err).NotTo(HaveOccurred())
	return &response
}

type ImageAVailabilityVerifier struct{}

func (i *ImageAVailabilityVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		log.Errorf("ImageAVailabilityVerifier returned with exit code %d. error: %s", actualReply.ExitCode, actualReply.Error)
		return false
	}
	if actualReply.StepType != models.StepTypeContainerImageAvailability {
		log.Errorf("ImageAVailabilityVerifier invalid step reply %s", actualReply.StepType)
		return false
	}

	return true
}

func pullImage(containerTool, image string) bool {
	_, _, exitCode := util.Execute(containerTool, "pull", image)
	return exitCode == 0
}

func isImageAvailable(containerTool, image string) bool {
	stdout, _, exitCode := util.Execute(containerTool, "images", "--quiet", image)
	return exitCode == 0 && stdout != ""
}

func removeImage(containerTool, image string) bool {
	_, _, exitCode := util.Execute(containerTool, "rmi", "--force", image)
	return exitCode == 0
}
