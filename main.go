package main

import (
	"fmt"
	"os"

	"github.com/wkqco33/package_manager/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
