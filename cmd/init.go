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
	Short: "ppm 설정 초기화",
	Long:  `ppm 설정 파일 및 디렉토리를 초기화합니다 (예: ~/.config/ppm).`,
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
