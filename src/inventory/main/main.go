package main

import (
	"fmt"
	"github.com/ori-amizur/introspector/src/inventory"
	"github.com/ori-amizur/introspector/src/util"
)

func main() {
	util.SetLogging("inventory")
	fmt.Print(string(inventory.CreateInveroryInfo()))

}
