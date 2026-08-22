package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wkqco33/package_manager/internal/apperr"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/platform"
	"github.com/wkqco33/package_manager/internal/ui"
)

// Package는 패키지 메타데이터를 나타냅니다.
type Package struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Source       string   `json:"source"`
	RegistryURL  string   `json:"registry_url,omitempty"`
	AssetID      int64    `json:"asset_id"` // 프라이빗 에셋 다운로드용 ID
	Checksum     string   `json:"checksum"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	Homepage     string   `json:"homepage"`
	BinName      string   `json:"bin_name"` // 실제 바이너리 이름 (레포지토리 이름과 다를 경우)
	Dependencies []string `json:"dependencies,omitempty"`
}

// Validate는 패키지 구조체의 필수 필드들의 유효성을 검증합니다.
func (p *Package) Validate() error {
	if p.Name == "" {
		return apperr.New(apperr.CodeInvalidInput, "package name is empty")
	}
	if p.Version == "" {
		return apperr.New(apperr.CodeInvalidInput, "package version is empty")
	}
	if p.Source == "" {
		return apperr.New(apperr.CodeInvalidInput, "package source URL is empty")
	}
	return nil
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

// InstallOptions controls potentially unsafe installation fallbacks.
type InstallOptions struct {
	// AllowSourceBuild permits building an untrusted source archive locally when
	// it does not contain a suitable pre-built executable.
	AllowSourceBuild bool
}

// InstallWithPackage preserves the library's historical behavior and allows
// source builds. CLI callers should use InstallWithPackageOptions explicitly.
func InstallWithPackage(p *Package, fetcher RegistryFetcher, archiver Archiver, installPath string) error {
	return InstallWithPackageOptions(p, fetcher, archiver, installPath, InstallOptions{AllowSourceBuild: true})
}

// InstallWithPackageOptions는 이미 조회한 메타데이터로 설치를 수행합니다.
func InstallWithPackageOptions(p *Package, fetcher RegistryFetcher, archiver Archiver, installPath string, options InstallOptions) error {
	if err := p.Validate(); err != nil {
		return err
	}

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

	var cacheFilePath string
	var useCache bool

	cacheDir, err := config.GetCacheDir()
	if err == nil {
		_ = os.MkdirAll(cacheDir, 0755)
		ext := ".tar.gz"
		if strings.HasSuffix(strings.ToLower(p.Source), ".zip") {
			ext = ".zip"
		} else if strings.HasSuffix(strings.ToLower(p.Source), ".tgz") {
			ext = ".tgz"
		}
		cacheFilePath = filepath.Join(cacheDir, fmt.Sprintf("%s-%s%s", safeName, p.Version, ext))

		if _, err := os.Stat(cacheFilePath); err == nil {
			useCache = true
		}
	}

	var archiveFile *os.File
	var archiveSize int64

	if useCache {
		logger.Info("Using cached archive: %s", cacheFilePath)
		file, err := os.Open(cacheFilePath)
		if err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to open cache file")
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to stat cache file")
		}
		archiveFile = file
		archiveSize = info.Size()
	} else {
		logger.Info("Downloading %s version %s...", p.Name, p.Version)
		body, size, err := fetcher.DownloadSource(p)
		if err != nil {
			return apperr.Wrap(apperr.CodeNetwork, err, "download error")
		}
		defer body.Close()

		// 임시 캐시 파일을 생성하여 다운로드 스트리밍
		tmpCacheFile, err := os.CreateTemp(cacheDir, "ppm-cache-*.tmp")
		if err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to create temp cache file")
		}
		tmpCachePath := tmpCacheFile.Name()
		defer func() {
			tmpCacheFile.Close()
			_ = os.Remove(tmpCachePath)
		}()

		bar := ui.NewProgressBar(size, 40, "Downloading")
		progressBody := &ui.ProgressReader{Reader: body, Bar: bar}

		archiveSize, err = io.Copy(tmpCacheFile, progressBody)
		progressBody.Close()
		if err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to write cache file")
		}
		tmpCacheFile.Close()

		// 다운로드 완료 시 최종 캐시 파일로 이동
		if err := os.Rename(tmpCachePath, cacheFilePath); err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to finalize cache file")
		}

		file, err := os.Open(cacheFilePath)
		if err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to open finalized cache file")
		}
		archiveFile = file
	}
	defer archiveFile.Close()

	if err := verifyChecksum(archiveFile, p.Checksum); err != nil {
		if cacheFilePath != "" {
			_ = os.Remove(cacheFilePath)
		}
		return apperr.Wrap(apperr.CodeArchive, err, "archive checksum verification failed")
	}

	// 최종 파일에 대해 압축 해제 렌더링
	bar := ui.NewProgressBar(archiveSize, 40, "Extracting")
	progressReader := &ui.ProgressReader{Reader: archiveFile, Bar: bar}

	logger.Debug("Extracting archive", "dest", extractDir)
	extractErr := archiver.Extract(progressReader, extractDir)
	progressReader.Close()
	if extractErr != nil {
		return apperr.Wrap(apperr.CodeArchive, extractErr, "extraction error")
	}

	// 업데이트 기능을 위해 메타데이터 저장
	if err := saveMetadata(extractDir, p); err != nil {
		logger.Warn("Failed to save metadata: %v", err)
	}

	// 바이너리 링크 생성
	logger.Debug("Linking binary", "src", extractDir, "link", targetLink)

	// 기본값: 아카이브 내부 바이너리 이름은 저장소명(safeName)과 동일하다고 가정
	if err := linkInstalledBinary(archiver, extractDir, binName, targetLink, options.AllowSourceBuild); err != nil {
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

func linkInstalledBinary(archiver Archiver, extractDir, binName, targetLink string, allowSourceBuild bool) error {
	if err := archiver.Link(extractDir, binName, targetLink); err == nil {
		return nil
	} else if !allowSourceBuild {
		return apperr.New(apperr.CodeArchive, "packaged executable %s not found; source builds are disabled (use --from-source only for trusted repositories)", binName)
	} else {
		attempted, buildErr := buildGoSourceFallback(extractDir, binName)
		if !attempted {
			return err
		}
		if buildErr != nil {
			return apperr.New(apperr.CodeArchive, "failed to find packaged binary and build from source: link error: %v; build error: %v", err, buildErr)
		}
		if err := archiver.Link(extractDir, binName, targetLink); err != nil {
			return apperr.Wrap(apperr.CodeFileSystem, err, "failed to link built binary")
		}
		return nil
	}
}

func buildGoSourceFallback(extractDir, binName string) (bool, error) {
	buildDir, ok := findGoBuildDir(extractDir)
	if !ok {
		return false, nil
	}

	outputPath := filepath.Join(extractDir, binName)
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	cmd.Dir = buildDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return true, fmt.Errorf("go build failed in %s: %s", buildDir, msg)
	}

	if _, err := os.Stat(outputPath); err != nil {
		return true, fmt.Errorf("go build did not produce %s: %w", outputPath, err)
	}

	return true, nil
}

func findGoBuildDir(root string) (string, bool) {
	if hasPackageMain(root) {
		return root, true
	}

	type candidate struct {
		path  string
		depth int
	}
	var candidates []candidate

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || rel == "." {
			return nil
		}

		depth := len(strings.Split(filepath.ToSlash(rel), "/"))
		if depth > 4 {
			return filepath.SkipDir
		}

		if hasPackageMain(path) {
			candidates = append(candidates, candidate{path: path, depth: depth})
			return filepath.SkipDir
		}
		return nil
	})

	if len(candidates) == 0 {
		return "", false
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].depth != candidates[j].depth {
			return candidates[i].depth < candidates[j].depth
		}
		return candidates[i].path < candidates[j].path
	})

	return candidates[0].path, true
}

func hasPackageMain(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "package main") {
			return true
		}
	}

	return false
}

func verifyChecksum(file *os.File, expected string) error {
	if expected == "" {
		return nil
	}

	expected = strings.TrimPrefix(strings.TrimSpace(expected), "sha256:")
	if len(expected) != sha256.Size*2 {
		return fmt.Errorf("invalid SHA-256 checksum length")
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("invalid SHA-256 checksum: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to hash archive: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to rewind archive: %w", err)
	}
	return nil
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
