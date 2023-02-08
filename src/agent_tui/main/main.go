package main

import (
	"fmt"
	"os"

	"github.com/openshift/agent-installer-utils/tools/agent_tui"
	"github.com/openshift/agent-installer-utils/tools/agent_tui/checks"
)

func main() {
	releaseImage := os.Getenv("RELEASE_IMAGE")
	logPath := os.Getenv("AGENT_TUI_LOG_PATH")
	if releaseImage == "" {
		fmt.Println("RELEASE_IMAGE environment variable is not specified.")
		fmt.Println("Unable to perform connectivity checks.")
		fmt.Println("Exiting agent-tui.")
		os.Exit(1)
	}
	if logPath == "" {
		logPath = "/tmp/agent_tui.log"
		fmt.Printf("AGENT_TUI_LOG_PATH is unspecified, logging to: %v\n", logPath)
	}
	config := checks.Config{
		ReleaseImageURL: releaseImage,
		LogPath:         logPath,
	}
	agent_tui.App(nil, config)
}
