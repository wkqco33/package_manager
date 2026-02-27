package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/registry"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info [package]",
	Short: "패키지 정보 표시",
	Long:  `원격 레지스트리에서 패키지에 대한 상세 정보를 표시합니다.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pkgName := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		fetcher := &registry.GitHubRegistry{
			Token: cfg.AuthToken,
			URL:   cfg.RegistryURL,
		}

		logger.Info("Fetching metadata for %s...", pkgName)
		p, err := fetcher.GetMetadata(pkgName)
		if err != nil {
			logger.Error("Failed to get metadata: %v", err)
			os.Exit(1)
		}

		logger.Info("Package: %s", p.Name)
		logger.Info("Latest Version: %s", p.Version)
		if p.Description != "" {
			logger.Info("Description: %s", p.Description)
		}
		if p.Author != "" {
			logger.Info("Author: %s", p.Author)
		}
		if p.Homepage != "" {
			logger.Info("Homepage: %s", p.Homepage)
		}
		logger.Info("Source URL: %s", p.Source)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
