package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"ppm/internal/apperr"
	"ppm/internal/config"
	"ppm/internal/logger"

	"github.com/spf13/cobra"
)

var (
	cleanAll bool
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "패키지 캐시 및 사용하지 않는 버전 삭제",
	Long:  `설치된 패키지 중 현재 사용하지 않는 구버전이나 캐시 파일들을 정리합니다.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			if err == config.ErrConfigNotFound {
				PrintError(apperr.New(apperr.CodeConfig, "설정 파일을 찾을 수 없습니다. 'ppm init'을 먼저 실행해주세요."))
			} else {
				PrintError(err)
			}
			os.Exit(1)
		}

		packagesDir, err := config.GetPackagesDir()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		if cleanAll {
			performCleanAll(packagesDir, cfg.InstallPath)
		} else {
			performCleanUnused(packagesDir, cfg.InstallPath)
		}
	},
}

func performCleanAll(packagesDir, installPath string) {
	logger.Info("모든 패키지 및 링크를 삭제합니다...")

	// 1. Remove all packages
	if err := os.RemoveAll(packagesDir); err != nil {
		PrintError(apperr.Wrap(apperr.CodeFileSystem, err, "패키지 디렉토리 삭제 실패"))
	} else {
		logger.Success("패키지 디렉토리가 성공적으로 삭제되었습니다.")
	}

	// 2. Remove related symlinks in installPath
	entries, err := os.ReadDir(installPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("설치 경로(%s)를 읽을 수 없습니다.", installPath)
		}
		return
	}

	for _, entry := range entries {
		fullPath := filepath.Join(installPath, entry.Name())
		// Check if it's a symlink pointing to our packagesDir
		linkTarget, err := os.Readlink(fullPath)
		if err == nil {
			if strings.Contains(linkTarget, packagesDir) {
				logger.Debug("심볼릭 링크 삭제: %s -> %s", fullPath, linkTarget)
				if err := os.Remove(fullPath); err != nil {
					logger.Warn("링크 삭제 실패: %s", fullPath)
				}
			}
		}
	}
	logger.Success("정리가 완료되었습니다.")
}

func performCleanUnused(packagesDir, installPath string) {
	logger.Info("사용하지 않는 구버전 패키지를 정리합니다...")

	// 1. Find all active versions by checking symlinks in installPath
	activeDirs := make(map[string]bool)
	entries, err := os.ReadDir(installPath)
	if err == nil {
		for _, entry := range entries {
			fullPath := filepath.Join(installPath, entry.Name())
			linkTarget, err := os.Readlink(fullPath)
			if err == nil {
				// If linkTarget is inside packagesDir, mark it active
				if strings.Contains(linkTarget, packagesDir) {
					// We need the direct subdirectory of packagesDir
					// e.g., /home/user/.config/ppm/packages/repo-v1.0.0/repo -> /home/user/.config/ppm/packages/repo-v1.0.0
					rel, err := filepath.Rel(packagesDir, linkTarget)
					if err == nil {
						parts := strings.Split(rel, string(os.PathSeparator))
						if len(parts) > 0 {
							activeDirs[parts[0]] = true
							logger.Debug("활성 패키지 감지: %s", parts[0])
						}
					}
				}
			}
		}
	}

	// 2. Remove directories in packagesDir that are not active
	pkgEntries, err := os.ReadDir(packagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("정리할 패키지가 없습니다.")
			return
		}
		PrintError(apperr.Wrap(apperr.CodeFileSystem, err, "패키지 디렉토리 읽기 실패"))
		return
	}

	removedCount := 0
	for _, entry := range pkgEntries {
		if !entry.IsDir() {
			continue
		}
		if !activeDirs[entry.Name()] {
			target := filepath.Join(packagesDir, entry.Name())
			logger.Info("삭제 중: %s", entry.Name())
			if err := os.RemoveAll(target); err != nil {
				logger.Warn("삭제 실패: %s (%v)", target, err)
			} else {
				removedCount++
			}
		}
	}

	if removedCount == 0 {
		logger.Success("정리할 사용하지 않는 패키지가 없습니다.")
	} else {
		logger.Success("총 %d개의 사용하지 않는 패키지를 정리했습니다.", removedCount)
	}
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVarP(&cleanAll, "all", "a", false, "모든 설치된 패키지 및 링크 삭제")
}
