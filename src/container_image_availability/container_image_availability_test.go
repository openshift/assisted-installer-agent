package container_image_availability

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
	mock "github.com/stretchr/testify/mock"
)

const (
	defaultTestImage              = "image"
	defaultTestPullTimeoutSeconds = 5
	defaultTestImageSizeInBytes   = int64(333000000)
)

var _ = Describe("Image availability", func() {
	var (
		imageAvailabilityDependencies *MockImageAvailabilityDependencies
		log                           *logrus.Logger
	)

	BeforeEach(func() {
		imageAvailabilityDependencies = &MockImageAvailabilityDependencies{}
		log = logrus.New()

	})

	AfterEach(func() {
		imageAvailabilityDependencies.AssertExpectations(GinkgoT())
	})

	convertStringArryToInterfaceArray := func(args []string) []interface{} {
		args_as_interface := make([]interface{}, len(args))
		for i := range args {
			args_as_interface[i] = args[i]
		}

		return args_as_interface
	}

	generatePullCommand := func(image string) []interface{} {
		cmd := fmt.Sprintf(templatePull, image)
		cmd = fmt.Sprintf(templateTimeout, mock.Anything, cmd)
		return convertStringArryToInterfaceArray(strings.Split(cmd, " "))
	}

	generateGetCommand := func(image string) []interface{} {
		cmd := fmt.Sprintf(templateGetImage, image)
		return convertStringArryToInterfaceArray(strings.Split(cmd, " "))
	}

	generateInspectCommand := func(image string) []interface{} {
		cmd := fmt.Sprintf(templateInspect, image)
		return convertStringArryToInterfaceArray(strings.Split(cmd, " "))
	}

	Context("pullImage", func() {
		It("image_was_pulled", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", 0).Once()

			err := pullImage(imageAvailabilityDependencies, defaultTestPullTimeoutSeconds, defaultTestImage)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("image_is_unavailable", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", 1).Once()

			err := pullImage(imageAvailabilityDependencies, defaultTestPullTimeoutSeconds, defaultTestImage)
			Expect(err).Should(HaveOccurred())
		})

		It("timeout", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", util.TimeoutExitCode).Once()

			err := pullImage(imageAvailabilityDependencies, defaultTestPullTimeoutSeconds, defaultTestImage)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal(fmt.Sprintf("podman pull was timed out after %d seconds", defaultTestPullTimeoutSeconds)))

		})
	})

	Context("getImageSizeInBytes", func() {
		It("image_exist", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateInspectCommand(defaultTestImage)...).Return(strconv.FormatInt(defaultTestImageSizeInBytes, 10), "", 0).Once()

			size, err := getImageSizeInBytes(imageAvailabilityDependencies, defaultTestImage)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(size).Should(Equal(float64(defaultTestImageSizeInBytes)))
		})

		It("trim_output", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateInspectCommand(defaultTestImage)...).Return(strconv.FormatInt(defaultTestImageSizeInBytes, 10)+"\n", "", 0).Once()

			size, err := getImageSizeInBytes(imageAvailabilityDependencies, defaultTestImage)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(size).Should(Equal(float64(defaultTestImageSizeInBytes)))
		})

		It("image_doesnt_exist", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateInspectCommand(defaultTestImage)...).Return("", "", 1).Once()

			size, err := getImageSizeInBytes(imageAvailabilityDependencies, defaultTestImage)
			Expect(err).Should(HaveOccurred())
			Expect(size).Should(BeZero())
		})

		It("malform_output", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateInspectCommand(defaultTestImage)...).Return("not_a_real_size", "", 0).Once()

			size, err := getImageSizeInBytes(imageAvailabilityDependencies, defaultTestImage)
			Expect(err).Should(HaveOccurred())
			Expect(size).Should(BeZero())
		})
	})

	Context("isImageAvailable", func() {
		It("image_found", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("123", "", 0).Once()
			Expect(isImageAvailable(imageAvailabilityDependencies, defaultTestImage)).Should(BeTrue())
		})

		It("image_not_found", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("", "", 0).Once()
			Expect(isImageAvailable(imageAvailabilityDependencies, defaultTestImage)).Should(BeFalse())
		})

		It("ExecutePrivileged_failure", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("", "", 1).Once()
			Expect(isImageAvailable(imageAvailabilityDependencies, defaultTestImage)).Should(BeFalse())
		})
	})

	Context("handleImageAvailability", func() {
		It("image_pulled", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("", "", 0).Once()
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", 0).Once()
			imageAvailabilityDependencies.On("ExecutePrivileged", generateInspectCommand(defaultTestImage)...).Return(strconv.FormatInt(defaultTestImageSizeInBytes, 10), "", 0).Once()
			output := handleImageAvailability(imageAvailabilityDependencies, log, defaultTestPullTimeoutSeconds, defaultTestImage)

			Expect(output.Name).Should(Equal(defaultTestImage))
			checkImageAvailability(output, true, true)
		})

		It("image_already_exist", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("123", "", 0).Once()
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", 0).Once()
			output := handleImageAvailability(imageAvailabilityDependencies, log, defaultTestPullTimeoutSeconds, defaultTestImage)

			checkImageAvailability(output, true, false)
		})

		It("failed_to_pull", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("", "", 0).Once()
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", 1).Once()
			output := handleImageAvailability(imageAvailabilityDependencies, log, defaultTestPullTimeoutSeconds, defaultTestImage)

			Expect(output.Name).Should(Equal(defaultTestImage))
			checkImageAvailability(output, false, true)
		})

		It("failed_to_pull_timeout", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("", "", 0).Once()
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", util.TimeoutExitCode).Run(func(args mock.Arguments) {
				time.Sleep(defaultTestPullTimeoutSeconds * time.Second)
			}).Once()
			output := handleImageAvailability(imageAvailabilityDependencies, log, defaultTestPullTimeoutSeconds, defaultTestImage)

			Expect(output.Name).Should(Equal(defaultTestImage))
			checkImageAvailability(output, false, true)
		})

		It("failed_to_get_size", func() {
			imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(defaultTestImage)...).Return("", "", 0).Once()
			imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(defaultTestImage)...).Return("", "", 0).Once()
			imageAvailabilityDependencies.On("ExecutePrivileged", generateInspectCommand(defaultTestImage)...).Return("", "", 1).Once()
			output := handleImageAvailability(imageAvailabilityDependencies, log, defaultTestPullTimeoutSeconds, defaultTestImage)

			Expect(output.Name).Should(Equal(defaultTestImage))
			checkImageAvailability(output, false, true)
		})
	})

	Context("Run", func() {
		It("multiple_images", func() {
			images := []string{"image1", "image2", "image3"}
			request := models.ContainerImageAvailabilityRequest{
				Images:  images,
				Timeout: defaultTestPullTimeoutSeconds,
			}
			b, err := json.Marshal(request)
			Expect(err).ShouldNot(HaveOccurred())
			remaining := defaultTestPullTimeoutSeconds
			prevTimeout := remaining
			for _, image := range images {
				imageAvailabilityDependencies.On("ExecutePrivileged", generateGetCommand(image)...).Return("", "", 0).Once()
				imageAvailabilityDependencies.On("ExecutePrivileged", generatePullCommand(image)...).Return("", "", 0).Once().Run(func(args mock.Arguments) {
					remainingTimeoutStr, ok := args.Get(1).(string)
					Expect(ok).To(BeTrue())
					currentTimeout, err := strconv.Atoi(remainingTimeoutStr)
					Expect(err).ToNot(HaveOccurred())
					Expect(currentTimeout).To(BeNumerically(">", 0))
					Expect(currentTimeout).To(BeNumerically("<", remaining))
					Expect(currentTimeout).To(BeNumerically("<", prevTimeout))
					prevTimeout = currentTimeout
					time.Sleep(time.Second)
					remaining--
				})
				imageAvailabilityDependencies.On("ExecutePrivileged", generateInspectCommand(image)...).Return(strconv.FormatInt(defaultTestImageSizeInBytes, 10), "", 0).Once()
			}

			stdout, stderr, exitCode := Run(string(b), imageAvailabilityDependencies, log)

			Expect(exitCode).Should(BeZero())
			Expect(stderr).Should(BeEmpty())

			var response models.ContainerImageAvailabilityResponse

			Expect(json.Unmarshal([]byte(stdout), &response)).ShouldNot(HaveOccurred())
			Expect(response.Images).Should(HaveLen(len(images)))

			for _, image := range response.Images {
				Expect(images).Should(ContainElement(image.Name))
				checkImageAvailability(image, true, true)
			}
		})
	})
})

func checkImageAvailability(image *models.ContainerImageAvailability, expectedResult, isRemoteImage bool) {
	if expectedResult {
		Expect(image.Result).Should(Equal(models.ContainerImageAvailabilityResultSuccess))
	} else {
		Expect(image.Result).Should(Equal(models.ContainerImageAvailabilityResultFailure))
	}

	if expectedResult && isRemoteImage {
		Expect(image.Time).Should(BeNumerically(">", 0))
		Expect(image.DownloadRate).Should(BeNumerically(">", 0))
		Expect(image.SizeBytes).Should(Equal(float64(defaultTestImageSizeInBytes)))
	} else { // failure or local image
		Expect(image.DownloadRate).Should(BeZero())
		Expect(image.Time).Should(BeZero())
		Expect(image.SizeBytes).Should(BeZero())
	}
}

func TestUnitests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Image availability unit tests")
}
