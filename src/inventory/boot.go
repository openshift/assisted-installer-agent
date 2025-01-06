package inventory

import (
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/sirupsen/logrus"
)

// secureBootEfivarsPath is the full path of the file that contains the value of the `SecureBoot` EFI variable. Note
// that we have to add the `/host` prefix because this runs inside a container where the host filesystems are mounted
// in that `/host` directory.
const secureBootEfivarsPath = "/host/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"

type boot struct {
	dependencies util.IDependencies
}

func newBoot(dependencies util.IDependencies) *boot {
	return &boot{dependencies: dependencies}
}

func (b *boot) getPxeInterface() string {
	cmdline, err := b.dependencies.ReadFile("/proc/cmdline")
	if err != nil {
		return ""
	}
	prefix := "BOOTIF="
	for _, part := range strings.Split(strings.TrimSpace(string(cmdline)), " ") {
		if strings.HasPrefix(part, prefix) {
			return part[len(prefix):]
		}
	}
	return ""
}

func (b *boot) getCurrentBootMode() (ret string) {
	ret = "bios"
	mode, err := b.dependencies.Stat("/sys/firmware/efi")
	if err != nil {
		return
	}
	if mode.IsDir() {
		ret = "uefi"
	}
	return
}

func (b *boot) getCommandLine() string {
	cmdline, err := b.dependencies.ReadFile("/proc/cmdline")
	if err != nil {
		return ""
	}
	return string(cmdline)
}

// getSecureBootState gets the value of the `SecureBoot` EFI variable.
func (b *boot) getSecureBootState() models.SecureBootState {
	// We assume that the `efivars` filesystem is mounted under `/sys/firmware/efi/efivars`. The format of the
	// files inside that is the name of the variable followed by a unique UUID. For the `SecureBoot` variable the
	// UUID is `8be4df61-93ca-11d2-aa0d-00e098032b8c`. The content of the file is 4 bytes that represent the
	// attributes of the variable (read only, read write, etc) and a variable number of bytes for the value. For the
	// `SecureBoot` variable the content is one byte, and the value of that byte is 0 if secure boot is disabled or
	// 1 if it is enabled.
	logger := logrus.WithFields(logrus.Fields{
		"path": secureBootEfivarsPath,
	})
	content, err := b.dependencies.ReadFile(secureBootEfivarsPath)
	if errors.Is(err, os.ErrNotExist) {
		logger.WithError(err).Info("Secure boot EFI variable file doesn't exist")
		return models.SecureBootStateNotSupported
	}
	if err != nil {
		logger.WithError(err).Info("Failed to read secure boot EFI variable file")
		return models.SecureBootStateUnknown
	}
	logger = logger.WithFields(logrus.Fields{
		"content": hex.EncodeToString(content),
		"length":  len(content),
	})
	if len(content) != 5 {
		logger.Error("Expected secure boot EFI variable file content to have exactly 5 bytes")
		return models.SecureBootStateUnknown
	}
	state := content[4]
	logger = logger.WithFields(logrus.Fields{
		"state": state,
	})
	switch state {
	case 0:
		logger.Info("Secure boot is supported and disabled")
		return models.SecureBootStateDisabled
	case 1:
		logger.Info("Secure boot is supported and enabled")
		return models.SecureBootStateEnabled
	default:
		logger.Error("Secure boot state is supported, but value is unknown")
		return models.SecureBootStateUnknown
	}
}

func (b *boot) getDeviceType() string {
	stdOut, _, code := b.dependencies.ExecutePrivileged("findmnt", "--noheadings", "--output", "SOURCE", "/sysroot")
	if code != 0 {
		return ""
	}
	if strings.HasPrefix(stdOut, "/dev/loop") {
		return models.BootDeviceTypeEphemeral
	}
	return models.BootDeviceTypePersistent
}

func (b *boot) getBoot() *models.Boot {
	ret := models.Boot{
		CurrentBootMode: b.getCurrentBootMode(),
		PxeInterface:    b.getPxeInterface(),
		CommandLine:     b.getCommandLine(),
		SecureBootState: b.getSecureBootState(),
		DeviceType:      b.getDeviceType(),
	}
	return &ret
}

func GetBoot(dependencies util.IDependencies) *models.Boot {
	return newBoot(dependencies).getBoot()
}
