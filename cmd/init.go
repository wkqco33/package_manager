package cmd

import (
	"os"

	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/platform"

	"github.com/spf13/cobra"
)

// initCmd는 init 명령입니다.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "ppm 설정 초기화",
	Long:  `ppm 설정 파일 및 디렉토리를 플랫폼에 맞게 초기화합니다.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Initializing ppm configuration...")
		err := config.GenerateDefaultConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}
		paths, _ := platform.GetPaths()
		logger.Success("Configuration file created successfully at %s/config.yaml", paths.ConfigDir)
		logger.Info("Please edit the file to add your AuthToken.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
