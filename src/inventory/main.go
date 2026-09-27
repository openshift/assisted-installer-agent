package inventory

import (
	"fmt"

	"github.com/openshift/assisted-installer-agent/src/config"
	"github.com/openshift/assisted-installer-agent/src/util"
)

func Main() {
	inventoryConfig := config.ProcessInventoryConfigArgs()
	util.SetLogging("inventory", inventoryConfig.TextLogging, inventoryConfig.JournalLogging, inventoryConfig.StdoutLogging, inventoryConfig.ForcedHostID)
	fmt.Print(string(CreateInventoryInfo(inventoryConfig)))
}
