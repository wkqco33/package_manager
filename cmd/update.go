package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"ppm/internal/archive"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
	"ppm/internal/registry"
)

// updateCmd는 update 명령입니다.
var updateCmd = &cobra.Command{
	Use:   "update [package...]",
	Short: "설치된 패키지 업데이트",
	Long:  `설치된 패키지를 최신 버전으로 업데이트합니다. 인자가 없으면 모든 패키지를 업데이트합니다.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		packagesDir, err := config.GetPackagesDir()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		var targetPackages []string
		if len(args) > 0 {
			targetPackages = args
		} else {
			installed, err := pkg.ListInstalled(packagesDir)
			if err != nil {
				PrintError(err)
				os.Exit(1)
			}
			for _, p := range installed {
				// 전체 이름(owner/repo) 형식만 자동 업데이트
				if strings.Contains(p.Name, "/") {
					targetPackages = append(targetPackages, p.Name)
				} else {
					logger.Warn("레거시 패키지 '%s'는 자동 업데이트를 지원하지 않습니다. 'ppm install owner/repo'로 다시 설치해주세요.", p.Name)
				}
			}
		}

		if len(targetPackages) == 0 {
			logger.Info("업데이트할 패키지가 없습니다.")
			return
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, 5)
		errCh := make(chan error, len(targetPackages))

		for _, pkgName := range targetPackages {
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

				// TODO: 설치 전 버전 비교로 중복 다운로드 방지
				if err := pkg.Install(name, fetcher, archiver, cfg.InstallPath); err != nil {
					errCh <- fmt.Errorf("[%s] %w", name, err)
				}
			}(pkgName)
		}

		wg.Wait()
		close(errCh)

		hasError := false
		for err := range errCh {
			logger.Error("Update failed: %v", err)
			hasError = true
		}

		if !hasError {
			logger.Success("모든 패키지가 최신 상태입니다.")
		} else {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
