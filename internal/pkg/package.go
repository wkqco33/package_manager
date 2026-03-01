package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"ppm/internal/apperr"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/platform"
	"ppm/internal/ui"
)

// Package는 패키지 메타데이터를 나타냅니다.
type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	AssetID     int64  `json:"asset_id"` // 프라이빗 에셋 다운로드용 ID
	Checksum    string `json:"checksum"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
	BinName     string `json:"bin_name"` // 실제 바이너리 이름 (레포지토리 이름과 다를 경우)
}

// RegistryFetcher는 레지스트리 메타데이터 조회/소스 다운로드 인터페이스입니다.
type RegistryFetcher interface {
	GetMetadata(pkgName string) (*Package, error)
	DownloadSource(pkg *Package) (io.ReadCloser, int64, error) // 소스 아카이브의 리더와 크기 반환
}

// Archiver는 아카이브 해제/바이너리 링크 인터페이스입니다.
type Archiver interface {
	Extract(r io.Reader, destDir string) error
	Link(extractedDir, binName, targetLink string) error
}

// Install은 메타데이터를 먼저 조회한 뒤 설치를 수행합니다.
func Install(pkgName string, fetcher RegistryFetcher, archiver Archiver, installPath string) error {
	spinner := ui.NewSpinner("Fetching metadata for " + pkgName + "...")
	spinner.Start()
	p, err := fetcher.GetMetadata(pkgName)
	spinner.Stop()
	if err != nil {
		return apperr.Wrap(apperr.CodeRegistry, err, "metadata fetch error")
	}
	logger.Success("Metadata fetched successfully")

	return InstallWithPackage(p, fetcher, archiver, installPath)
}

// InstallWithPackage는 이미 조회한 메타데이터로 설치를 수행합니다.
func InstallWithPackage(p *Package, fetcher RegistryFetcher, archiver Archiver, installPath string) error {
	// 이미 설치됐는지 확인
	packagesDir, err := config.GetPackagesDir()
	if err != nil {
		return err
	}
	safeName := filepath.Base(p.Name)
	extractDir := filepath.Join(packagesDir, fmt.Sprintf("%s-%s", safeName, p.Version))

	baseBinName := safeName
	if p.BinName != "" {
		baseBinName = p.BinName
	}
	binName := platform.ExecutableName(baseBinName)
	targetLink := filepath.Join(installPath, binName)

	if _, err := os.Stat(extractDir); err == nil {
		// 같은 버전이 이미 있으면 링크만 점검
		logger.Info("%s version %s is already installed. Ensuring link...", p.Name, p.Version)
		if err := archiver.Link(extractDir, binName, targetLink); err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "linking error")
		}
		logger.Success("%s is already up to date.", p.Name)
		return nil
	}

	logger.Info("Downloading and extracting %s version %s...", p.Name, p.Version)
	body, size, err := fetcher.DownloadSource(p)
	if err != nil {
		return apperr.Wrap(apperr.CodeNetwork, err, "download error")
	}

	bar := ui.NewProgressBar(size, 40, "Downloading")
	progressBody := &ui.ProgressReader{Reader: body, Bar: bar}
	defer progressBody.Close()

	logger.Debug("Extracting archive", "dest", extractDir)
	if err := archiver.Extract(progressBody, extractDir); err != nil {
		return apperr.Wrap(apperr.CodeArchive, err, "extraction error")
	}

	// 업데이트 기능을 위해 메타데이터 저장
	if err := saveMetadata(extractDir, p); err != nil {
		logger.Warn("Failed to save metadata: %v", err)
	}

	// 바이너리 링크 생성
	logger.Debug("Linking binary", "src", extractDir, "link", targetLink)

	// 기본값: 아카이브 내부 바이너리 이름은 저장소명(safeName)과 동일하다고 가정
	if err := archiver.Link(extractDir, binName, targetLink); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "linking error")
	}

	logger.Success("Successfully installed %s!", p.Name)
	return nil
}

// Uninstall은 패키지와 관련 바이너리/링크를 제거합니다.
func Uninstall(pkgName, installPath string) error {
	packagesDir, err := config.GetPackagesDir()
	if err != nil {
		return err
	}

	safeName := filepath.Base(pkgName)

	// 설치된 패키지들의 메타데이터를 확인하여 BinName 파악
	baseBinName := safeName
	entries, err := os.ReadDir(packagesDir)
	if err == nil {
		prefix := safeName + "-"
		for _, entry := range entries {
			if entry.IsDir() && (entry.Name() == safeName || strings.HasPrefix(entry.Name(), prefix)) {
				metaPath := filepath.Join(packagesDir, entry.Name(), "ppm-meta.json")
				data, readErr := os.ReadFile(metaPath)
				if readErr == nil {
					var p Package
					if json.Unmarshal(data, &p) == nil && p.BinName != "" {
						baseBinName = p.BinName
						break
					}
				}
			}
		}
	}

	binName := platform.ExecutableName(baseBinName)
	targetLink := filepath.Join(installPath, binName)

	// 1) installPath에서 바이너리 또는 심볼릭 링크 제거
	if _, err := os.Stat(targetLink); err == nil {
		logger.Debug("Removing binary/link", "path", targetLink)
		if err := os.Remove(targetLink); err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to remove binary/link")
		}
	} else if !os.IsNotExist(err) {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to check binary/link status")
	}

	// 2) packagesDir에서 해당 패키지의 모든 버전 제거
	if err != nil && os.IsNotExist(err) && entries == nil {
		return nil // 제거할 항목 없음
	}

	// 디렉토리를 다시 읽어 삭제 진행 (위에 err == nil 일때만 entries를 구했음)
	if entries == nil {
		entries, err = os.ReadDir(packagesDir)
		if err != nil && os.IsNotExist(err) {
			return nil
		}
	}

	removedCount := 0
	prefix := safeName + "-"
	for _, entry := range entries {
		if entry.IsDir() && (entry.Name() == safeName || strings.HasPrefix(entry.Name(), prefix)) {
			targetDir := filepath.Join(packagesDir, entry.Name())
			logger.Debug("Removing package directory", "path", targetDir)
			if err := os.RemoveAll(targetDir); err != nil {
				logger.Warn("Failed to remove directory: %s (%v)", targetDir, err)
			} else {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		logger.Success("Successfully uninstalled %s (removed %d version(s))", pkgName, removedCount)
	} else {
		logger.Warn("No installed files found for %s, but ensured binary/link is removed.", pkgName)
	}

	return nil
}

func saveMetadata(dir string, p *Package) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "ppm-meta.json"), data, 0644)
}

// ListInstalled는 packages 디렉터리를 스캔해 설치 목록을 반환합니다.
func ListInstalled(packagesDir string) ([]*Package, error) {
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Package{}, nil
		}
		return nil, err
	}

	var packages []*Package
	for _, entry := range entries {
		if entry.IsDir() {
			metaPath := filepath.Join(packagesDir, entry.Name(), "ppm-meta.json")
			data, err := os.ReadFile(metaPath)
			if err != nil {
				// 메타데이터 없는 레거시 패키지
				packages = append(packages, &Package{
					Name: entry.Name(),
				})
				continue
			}

			var p Package
			if err := json.Unmarshal(data, &p); err == nil {
				packages = append(packages, &p)
			} else {
				// 메타데이터 손상 시 레거시로 포함
				packages = append(packages, &Package{
					Name: entry.Name(),
				})
			}
		}
	}
	return packages, nil
}
