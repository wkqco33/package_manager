package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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

		installed, err := pkg.ListInstalled(packagesDir)
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		installedVersions := make(map[string]map[string]struct{})
		for _, p := range installed {
			if p.Name == "" || p.Version == "" {
				continue
			}
			if _, ok := installedVersions[p.Name]; !ok {
				installedVersions[p.Name] = make(map[string]struct{})
			}
			installedVersions[p.Name][p.Version] = struct{}{}
		}

		var targetPackages []string
		if len(args) > 0 {
			targetPackages = args
		} else {
			seen := make(map[string]struct{})
			for _, p := range installed {
				// 전체 이름(owner/repo) 형식만 자동 업데이트
				if strings.Contains(p.Name, "/") {
					if _, exists := seen[p.Name]; exists {
						continue
					}
					seen[p.Name] = struct{}{}
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

				latest, err := fetcher.GetMetadata(name)
				if err != nil {
					errCh <- fmt.Errorf("[%s] %w", name, err)
					return
				}

				if versions, ok := installedVersions[name]; ok {
					if _, exists := versions[latest.Version]; exists {
						logger.Info("%s는 이미 최신 버전(%s)입니다.", name, latest.Version)
						return
					}
				}

				safeName := filepath.Base(latest.Name)
				archiver := archive.NewArchiver(latest.Source, safeName)
				if err := pkg.InstallWithPackage(latest, fetcher, archiver, cfg.InstallPath); err != nil {
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
