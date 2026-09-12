package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// Paths는 ppm 표준 디렉터리 경로를 정의합니다.
type Paths struct {
	ConfigDir  string
	BinDir     string
	PackageDir string
	CacheDir   string
}

// GetPaths는 현재 운영체제 기준 표준 경로를 반환합니다.
func GetPaths() (*Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	var configDir string
	var binDir string
	var cacheDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %AppData%/ppm
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		configDir = filepath.Join(appData, "ppm")
		binDir = filepath.Join(home, ".local", "bin") // Windows는 표준 사용자 bin이 없어 .local/bin을 사용

		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		cacheDir = filepath.Join(localAppData, "ppm", "cache")
	case "darwin":
		// macOS: ~/Library/Application Support/ppm
		configDir = filepath.Join(home, "Library", "Application Support", "ppm")
		binDir = filepath.Join(home, ".local", "bin")
		cacheDir = filepath.Join(home, "Library", "Caches", "ppm")
	default:
		// Linux/Unix: ~/.config/ppm
		configDir = filepath.Join(home, ".config", "ppm")
		binDir = filepath.Join(home, ".local", "bin")
		cacheDir = filepath.Join(home, ".cache", "ppm")
	}

	// 명시적 PPM 경로 재정의가 항상 OS 기본값보다 우선합니다.
	if value := os.Getenv("PPM_CONFIG_DIR"); value != "" {
		configDir = value
	} else if value := os.Getenv("XDG_CONFIG_HOME"); value != "" && runtime.GOOS != "windows" {
		configDir = filepath.Join(value, "ppm")
	}
	if value := os.Getenv("PPM_INSTALL_DIR"); value != "" {
		binDir = value
	}
	if value := os.Getenv("PPM_CACHE_DIR"); value != "" {
		cacheDir = value
	} else if value := os.Getenv("XDG_CACHE_HOME"); value != "" && runtime.GOOS != "windows" {
		cacheDir = filepath.Join(value, "ppm")
	}

	return &Paths{
		ConfigDir:  configDir,
		BinDir:     binDir,
		PackageDir: filepath.Join(configDir, "packages"),
		CacheDir:   cacheDir,
	}, nil
}

// ExecutableName은 플랫폼별 실행 파일명을 반환합니다.
func ExecutableName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}
