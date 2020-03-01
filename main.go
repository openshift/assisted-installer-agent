package main

import (
	"fmt"
	"github.com/ori-amizur/introspector/scanners"
)

func main() {
	fmt.Printf("%+v\n",*scanners.ReadCpus())
	fmt.Printf("%+v\n", scanners.ReadBlockDevices())
	fmt.Printf("%+v\n", scanners.ReadMemory())
}
