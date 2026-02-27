package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"ppm/internal/archive"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
	"ppm/internal/registry"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [package...]",
	Short: "Install private packages",
	Long:  `Download and install one or more private packages from the configured registry in parallel. (e.g. ppm install user/repo1 user/repo2)`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, 5) // 최대 5개 동시 설치
		errCh := make(chan error, len(args))

		for _, pkgName := range args {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				fetcher := &registry.GitHubRegistry{
					Token: cfg.AuthToken,
					URL:   cfg.RegistryURL,
				}
				archiver := &archive.TarArchiver{}

				if err := pkg.Install(name, fetcher, archiver, cfg.InstallPath); err != nil {
					errCh <- fmt.Errorf("[%s] %w", name, err)
				}
			}(pkgName)
		}

		wg.Wait()
		close(errCh)

		hasError := false
		for err := range errCh {
			logger.Error("Installation failed: %v", err)
			hasError = true
		}

		if hasError {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
