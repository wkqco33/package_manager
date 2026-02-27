package cmd

import (
	"os"

	"ppm/internal/config"
	"ppm/internal/logger"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ppm configuration",
	Long:  `Initialize ppm configuration file and directories (e.g. ~/.config/ppm).`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Initializing ppm configuration...")
		err := config.GenerateDefaultConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}
		logger.Success("Configuration file created successfully at ~/.config/ppm/config.yaml")
		logger.Info("Please edit the file to add your AuthToken.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
