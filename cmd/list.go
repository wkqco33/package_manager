package cmd

import (
	"os"
	"path/filepath"

	"ppm/internal/apperr"
	"ppm/internal/logger"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "설치된 패키지 목록 표시",
	Long:  `현재 ppm을 통해 설치된 모든 패키지의 목록을 표시합니다.`,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			PrintError(apperr.Wrap(apperr.CodeFileSystem, err, "could not get user home directory"))
			os.Exit(1)
		}
		packagesDir := filepath.Join(home, ".config", "ppm", "packages")

		entries, err := os.ReadDir(packagesDir)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Info("No packages installed yet.")
				return
			}
			PrintError(apperr.Wrap(apperr.CodeFileSystem, err, "failed to read packages directory"))
			os.Exit(1)
		}

		count := 0
		logger.Info("Installed packages:")
		for _, e := range entries {
			if e.IsDir() {
				logger.Info(" - %s", e.Name())
				count++
			}
		}

		if count == 0 {
			logger.Info("No packages installed yet.")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
