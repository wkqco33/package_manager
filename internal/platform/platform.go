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
