package cmd

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wkqco33/package_manager/internal/app"
	"github.com/wkqco33/package_manager/internal/apperr"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/ui"

	"github.com/wkqco33/wcli"
)

var (
	cleanAll bool
)

type cleanDependencies struct {
	LoadConfig     func() (*config.Config, error)
	GetPackagesDir func() (string, error)
}

func defaultCleanDependencies() cleanDependencies {
	return cleanDependencies{
		LoadConfig:     config.LoadConfig,
		GetPackagesDir: config.GetPackagesDir,
	}
}

// cleanCmd는 clean 명령입니다.
var cleanCmd = newCleanCommand(defaultCleanDependencies())

func newCleanCommand(deps cleanDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "clean",
		Short: "패키지 캐시 및 사용하지 않는 버전 삭제",
		Long:  `설치된 패키지 중 현재 사용하지 않는 구버전이나 캐시 파일들을 정리합니다.`,
		Run: func(ctx *wcli.Context) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				if err == config.ErrConfigNotFound {
					return apperr.New(apperr.CodeConfig, "설정 파일을 찾을 수 없습니다. 'ppm init'을 먼저 실행해주세요.")
				} else {
					return err
				}
			}

			packagesDir, err := deps.GetPackagesDir()
			if err != nil {
				return err
			}

			if cleanAll {
				return performCleanAll(packagesDir, cfg.InstallPath)
			}
			return performCleanUnused(packagesDir, cfg.InstallPath)
		},
	}
}

func performCleanAll(packagesDir, installPath string) error {
	logger.Info("모든 패키지 및 링크를 삭제합니다...")
	spinner := ui.NewSpinner("패키지 및 링크를 정리하는 중...")
	spinner.Start()
	defer spinner.Stop()
	cleaner := app.PackageCleaner{
		ReadDir:   os.ReadDir,
		RemoveAll: os.RemoveAll,
		Remove:    os.Remove,
		Readlink:  os.Readlink,
	}
	result, err := cleaner.CleanAll(packagesDir, installPath)
	spinner.Stop()
	if err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "패키지 정리 실패")
	}
	if result.PackagesRemoved {
		logger.Success("패키지 디렉토리가 성공적으로 삭제되었습니다.")
	}
	logger.Success("정리가 완료되었습니다. 링크 %d개를 삭제했습니다.", result.LinksRemoved)
	return nil
}

func performCleanUnused(packagesDir, installPath string) error {
	logger.Info("사용하지 않는 구버전 패키지를 정리합니다...")
	spinner := ui.NewSpinner("사용하지 않는 패키지를 정리하는 중...")
	spinner.Start()
	defer spinner.Stop()
	cleaner := app.PackageCleaner{
		ReadDir:           os.ReadDir,
		RemoveAll:         os.RemoveAll,
		CollectActiveDirs: collectActiveDirs,
	}
	removedCount, err := cleaner.CleanUnused(packagesDir, installPath)
	spinner.Stop()
	if err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "패키지 정리 실패")
	}
	if removedCount == 0 {
		logger.Success("정리할 사용하지 않는 패키지가 없습니다.")
	} else {
		logger.Success("총 %d개의 사용하지 않는 패키지를 정리했습니다.", removedCount)
	}
	return nil
}

// collectActiveDirs는 installPath에 설치된 바이너리들을 기준으로 현재 활성 상태인
// packagesDir의 직계 하위 디렉터리 이름 집합을 반환합니다.
//
// Unix에서는 설치 바이너리가 packagesDir를 가리키는 심볼릭 링크이므로 링크 대상으로 판별하고,
// Windows에서는 심볼릭 링크 대신 파일을 복사하므로(see archive.*.Link) 링크가 존재하지 않습니다.
// 따라서 복사본의 경우 설치된 바이너리와 내용이 동일한 파일을 가진 패키지 디렉터리를 활성으로 판별합니다.
func collectActiveDirs(packagesDir, installPath string, pkgEntries []os.DirEntry) map[string]bool {
	activeDirs := make(map[string]bool)

	entries, err := os.ReadDir(installPath)
	if err != nil {
		// 설치 경로를 읽을 수 없으면 활성 버전을 신뢰성 있게 판별할 수 없습니다.
		return activeDirs
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fullPath := filepath.Join(installPath, entry.Name())

		// 1) 심볼릭 링크인 경우(Unix): 링크 대상으로 활성 디렉터리를 판별
		if linkTarget, err := os.Readlink(fullPath); err == nil {
			if name := activeDirFromLinkTarget(packagesDir, linkTarget); name != "" {
				activeDirs[name] = true
				logger.Debug("활성 패키지 감지(링크): %s", name)
			}
			continue
		}

		// 2) 일반 파일인 경우(Windows 복사본 등): 내용 비교로 활성 디렉터리를 판별
		info, err := entry.Info()
		if err != nil {
			continue
		}
		installedHash, err := hashFile(fullPath)
		if err != nil {
			continue
		}
		for _, pe := range pkgEntries {
			if !pe.IsDir() || activeDirs[pe.Name()] {
				continue
			}
			pkgDir := filepath.Join(packagesDir, pe.Name())
			if dirContainsMatchingFile(pkgDir, info.Size(), installedHash) {
				activeDirs[pe.Name()] = true
				logger.Debug("활성 패키지 감지(복사본): %s", pe.Name())
				break
			}
		}
	}

	return activeDirs
}

// activeDirFromLinkTarget는 심볼릭 링크 대상이 packagesDir 내부를 가리키는 경우
// 해당하는 packagesDir의 직계 하위 디렉터리 이름을 반환합니다.
// 예: .../packages/repo-v1.0.0/repo -> repo-v1.0.0
func activeDirFromLinkTarget(packagesDir, linkTarget string) string {
	if !strings.Contains(linkTarget, packagesDir) {
		return ""
	}
	rel, err := filepath.Rel(packagesDir, linkTarget)
	if err != nil {
		return ""
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) == 0 || parts[0] == "" || parts[0] == ".." {
		return ""
	}
	return parts[0]
}

// dirContainsMatchingFile는 dir 하위에 주어진 크기와 SHA-256 해시가 일치하는 파일이 있으면 true를 반환합니다.
func dirContainsMatchingFile(dir string, size int64, target [sha256.Size]byte) bool {
	found := false
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// 크기로 먼저 거른 뒤 해시를 비교합니다.
		if info.Size() != size {
			return nil
		}
		if h, err := hashFile(path); err == nil && h == target {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// hashFile은 파일의 SHA-256 해시를 계산합니다.
func hashFile(path string) ([sha256.Size]byte, error) {
	var sum [sha256.Size]byte
	f, err := os.Open(path)
	if err != nil {
		return sum, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return sum, err
	}
	copy(sum[:], h.Sum(nil))
	return sum, nil
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanAll, "all", "a", false, "모든 설치된 패키지 및 링크 삭제")
}
