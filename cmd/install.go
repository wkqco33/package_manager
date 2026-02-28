package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"

	"ppm/internal/archive"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
	"ppm/internal/registry"
)

// installCmd는 install 명령입니다.
var installCmd = &cobra.Command{
	Use:   "install [package...]",
	Short: "프라이빗 패키지 설치",
	Long:  `설정된 레지스트리에서 하나 이상의 프라이빗 패키지를 병렬로 다운로드하고 설치합니다. (예: ppm install user/repo1 user/repo2)`,
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

				// 아카이브 타입 판별을 위해 메타데이터를 먼저 조회
				p, err := fetcher.GetMetadata(name)
				if err != nil {
					errCh <- fmt.Errorf("[%s] %w", name, err)
					return
				}

				safeName := filepath.Base(p.Name)
				archiver := archive.NewArchiver(p.Source, safeName)

				if err := pkg.InstallWithPackage(p, fetcher, archiver, cfg.InstallPath); err != nil {
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
