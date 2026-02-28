package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
)

// uninstallCmd는 uninstall 명령입니다.
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [package...]",
	Short: "설치된 패키지 삭제",
	Long:  `설치된 바이너리 및 패키지 데이터 파일을 시스템에서 완전히 제거합니다. (예: ppm uninstall repo1 repo2)`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, 5) // 동시 삭제 최대 5개
		errCh := make(chan error, len(args))

		for _, name := range args {
			wg.Add(1)
			go func(pkgName string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				logger.Info("Uninstalling %s...", pkgName)
				if err := pkg.Uninstall(pkgName, cfg.InstallPath); err != nil {
					errCh <- fmt.Errorf("[%s] %w", pkgName, err)
				}
			}(name)
		}

		wg.Wait()
		close(errCh)

		hasError := false
		for err := range errCh {
			logger.Error("Uninstall failed: %v", err)
			hasError = true
		}

		if hasError {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
