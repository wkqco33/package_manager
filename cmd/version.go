package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version defines the current version of ppm.
// This can be overridden during the build process using -ldflags.
var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ppm",
	Long:  `Print the version number of ppm.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ppm version %s %s/%s\n", Version, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
