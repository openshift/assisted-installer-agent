package main

import (
	"fmt"
	"github.com/ori-amizur/introspector/src/commands"
	"github.com/ori-amizur/introspector/src/util"
)

func main() {
	util.SetLogging("hardware_info")
	fmt.Print(string(commands.CreateHostInfo()))
}
