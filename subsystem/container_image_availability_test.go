package subsystem

import (
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thoas/go-funk"

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
		images = []string{"quay.io/ibm/hello-world", "quay.io/coreos/etcd:latest"}

		for _, image := range images {
			deleteImage(image)
		}

		startImageAvailability(hostID, models.ContainerImageAvailabilityRequest{Images: images, Timeout: pullTimeoutInSeconds})
		checkImageAvailabilityResponse(hostID, images, true, true, pullTimeoutInSeconds)
	})

	It("Already downloaded image", func() {
		images = []string{"quay.io/ibm/hello-world"}

		for _, image := range images {
			deleteImage(image)
			Expect(pullImage(defaultContainerTool, image)).Should(BeTrue())
			Expect(isImageAvailable(defaultContainerTool, image)).Should(BeTrue())
		}

		startImageAvailability(hostID, models.ContainerImageAvailabilityRequest{Images: images, Timeout: pullTimeoutInSeconds})
		checkImageAvailabilityResponse(hostID, images, true, false, pullTimeoutInSeconds)
	})

	It("Invalid image", func() {
		images = []string{"invalid-registry/invalid-repository/image:tag"}
		startImageAvailability(hostID, models.ContainerImageAvailabilityRequest{Images: images, Timeout: pullTimeoutInSeconds})
		checkImageAvailabilityResponse(hostID, images, false, true, pullTimeoutInSeconds)
	})
})

func startImageAvailability(hostId string, request models.ContainerImageAvailabilityRequest) {
	_, err := addRegisterStub(hostId, http.StatusCreated, InfraEnvID)
	Expect(err).ShouldNot(HaveOccurred())
	setImageAvailabilityStub(hostId, request)
	setReplyStartAgent(hostId)
}

func checkImageAvailabilityResponse(hostID string, expectedImages []string,
	expectedResult bool, isRemoteImage bool, pullTimeout int64) {
	Eventually(func() bool {
		return isReplyFound(hostID, &ImageAVailabilityVerifier{expectedImages, expectedResult, isRemoteImage, pullTimeout})
	}, maxTimeout, 5*time.Second).Should(BeTrue())
}

func getImageAvailabilityResponseFromStepReply(actualReply *models.StepReply) *models.ContainerImageAvailabilityResponse {
	var response models.ContainerImageAvailabilityResponse
	if actualReply.Output == "" {
		return &response
	}
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	Expect(err).NotTo(HaveOccurred())
	return &response
}

type ImageAVailabilityVerifier struct {
	expectedImages []string
	expectedResult bool
	isRemoteImage  bool
	pullTimeout    int64
}

func (i *ImageAVailabilityVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 && actualReply.ExitCode != 2 {
		log.Errorf("ImageAvailabilityVerifier returned with exit code %d. error: %s", actualReply.ExitCode, actualReply.Error)
		return false
	}
	if actualReply.StepType != models.StepTypeContainerImageAvailability {
		log.Errorf("ImageAvailabilityVerifier invalid step reply %s", actualReply.StepType)
		return false
	}
	response := getImageAvailabilityResponseFromStepReply(actualReply)

	if response == nil {
		log.Errorf("ImageAvailabilityVerifier response is nil")
		return false
	}

	for _, image := range response.Images {
		if !funk.Contains(i.expectedImages, image.Name) {
			log.Errorf("ImageAvailabilityVerifier image %s wasn't expected in list %s", image.Name, i.expectedImages)
			return false
		}

		if !checkImageStatus(image, i.expectedResult, i.isRemoteImage, i.pullTimeout) {
			log.Errorf("ImageAvailabilityVerifier image %+v wasn't expected to result %v %v", image, i.expectedResult, i.isRemoteImage)
			return false
		}
	}

	return true
}

func checkImageStatus(image *models.ContainerImageAvailability, expectedResult bool, isRemoteImage bool, pullTimeout int64) bool {
	if isImageAvailable(defaultContainerTool, image.Name) != expectedResult {
		return false
	}

	if expectedResult {
		if image.Result != models.ContainerImageAvailabilityResultSuccess {
			return false
		}
	} else {
		if image.Result != models.ContainerImageAvailabilityResultFailure {
			return false
		}
	}

	if expectedResult && isRemoteImage {
		return image.Time > 0 && int64(image.Time) <= pullTimeout && image.DownloadRate > 0 && image.SizeBytes > 0
	} else { // failure or local image
		return image.Time == 0 && image.DownloadRate == 0 && image.SizeBytes == 0
	}
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
