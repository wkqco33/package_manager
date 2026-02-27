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
	Short: "Show package information",
	Long:  `Display detailed information about a package from the remote registry.`,
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
