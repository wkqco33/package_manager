package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/wkqco33/package_manager/cmd"
)

//go:embed PACKAGE_GUIDE.md
var packageGuideContent string

func init() {
	cmd.PackageGuide = packageGuideContent
}

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
