package util

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Status File Validation", func() {
	var (
		tempDir             string
		originalWaitingLoop time.Duration
	)

	BeforeEach(func() {
		var err error

		originalWaitingLoop = waitingLoop
		waitingLoop = 50 * time.Millisecond

		tempDir, err = os.MkdirTemp("", "status_file_test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		waitingLoop = originalWaitingLoop
		os.RemoveAll(tempDir)
	})

	Context("when status file path is empty", func() {
		It("should skip the check and return nil", func() {
			err := ValidateStatusFile("")
			Expect(err).To(BeNil())
		})
	})

	Context("when status file does not exist", func() {
		It("should skip the check and return nil (feature not implemented)", func() {
			statusFilePath := filepath.Join(tempDir, "nonexistent.json")
			err := ValidateStatusFile(statusFilePath)
			Expect(err).To(BeNil())
		})
	})

	Context("when status file is empty", func() {
		It("should wait until status changes", func() {
			statusFilePath := filepath.Join(tempDir, "status.json")

			// Create empty file
			Expect(os.WriteFile(statusFilePath, []byte(""), 0600)).To(Succeed())

			go func() {
				time.Sleep(20 * time.Millisecond)
				content := StatusFileContent{Status: StatusWaitingForAgent}
				data, err := json.Marshal(content)
				Expect(err).ToNot(HaveOccurred())
				Expect(os.WriteFile(statusFilePath, data, 0600)).To(Succeed())
			}()

			err := ValidateStatusFile(statusFilePath)
			Expect(err).To(BeNil())
		})
	})

	Context("when status is initializing", func() {
		It("should wait until status changes", func() {
			statusFilePath := filepath.Join(tempDir, "status.json")

			content := StatusFileContent{Status: StatusInitializing}
			data, _ := json.Marshal(content)
			Expect(os.WriteFile(statusFilePath, data, 0600)).To(Succeed())

			go func() {
				time.Sleep(20 * time.Millisecond)
				newContent := StatusFileContent{Status: StatusWaitingForAgent}
				newData, _ := json.Marshal(newContent)
				Expect(os.WriteFile(statusFilePath, newData, 0600)).To(Succeed())
			}()

			err := ValidateStatusFile(statusFilePath)
			Expect(err).To(BeNil())
		})
	})

	Context("when status is an empty string", func() {
		It("should wait until status changes", func() {
			statusFilePath := filepath.Join(tempDir, "status.json")

			content := StatusFileContent{Status: ""}
			data, _ := json.Marshal(content)
			Expect(os.WriteFile(statusFilePath, data, 0600)).To(Succeed())

			go func() {
				time.Sleep(20 * time.Millisecond)
				newContent := StatusFileContent{Status: StatusWaitingForAgent}
				newData, _ := json.Marshal(newContent)
				Expect(os.WriteFile(statusFilePath, newData, 0600)).To(Succeed())
			}()

			err := ValidateStatusFile(statusFilePath)
			Expect(err).To(BeNil())
		})
	})

	Context("when status is waiting for assisted agent", func() {
		It("should return nil immediately", func() {
			statusFilePath := filepath.Join(tempDir, "status.json")

			content := StatusFileContent{Status: StatusWaitingForAgent}
			data, _ := json.Marshal(content)
			Expect(os.WriteFile(statusFilePath, data, 0600)).To(Succeed())

			err := ValidateStatusFile(statusFilePath)
			Expect(err).To(BeNil())
		})
	})

	Context("when status is finalizing", func() {
		It("should return a finalizing status error", func() {
			statusFilePath := filepath.Join(tempDir, "status.json")

			content := StatusFileContent{Status: StatusFinalizing}
			data, _ := json.Marshal(content)
			Expect(os.WriteFile(statusFilePath, data, 0600)).To(Succeed())

			err := ValidateStatusFile(statusFilePath)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("ironic agent is already finalizing"))
		})
	})

	Context("when status has unexpected value", func() {
		It("should return error with unexpected status", func() {
			statusFilePath := filepath.Join(tempDir, "status.json")

			content := StatusFileContent{Status: "unknown_status"}
			data, _ := json.Marshal(content)
			Expect(os.WriteFile(statusFilePath, data, 0600)).To(Succeed())

			err := ValidateStatusFile(statusFilePath)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("unexpected ironic status"))
		})
	})

	Context("when file contains invalid JSON", func() {
		It("should return error", func() {
			statusFilePath := filepath.Join(tempDir, "status.json")

			// Write invalid JSON
			Expect(os.WriteFile(statusFilePath, []byte("{invalid json}"), 0600)).ToNot(HaveOccurred())

			err := ValidateStatusFile(statusFilePath)
			Expect(err).ToNot(BeNil())
		})
	})
})
