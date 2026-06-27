package config

import (
	"os"
	"path/filepath"
	"ppm/internal/platform"
	"testing"
)

// setupTempHome은 모든 OS에서 ppm 표준 경로가 임시 디렉터리를 가리키도록 환경변수를 설정합니다.
// os.UserHomeDir()는 Unix에서 HOME, Windows에서 USERPROFILE을 사용하므로 둘 다 설정하고,
// APPDATA는 비워서 platform.GetPaths()가 home/AppData/Roaming 으로 파생되게 합니다.
func setupTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)        // Unix
	t.Setenv("USERPROFILE", home) // Windows (os.UserHomeDir)
	t.Setenv("APPDATA", "")       // GetPaths가 home/AppData/Roaming 으로 파생
	return home
}

func TestGenerateAndLoadConfig(t *testing.T) {
	// 임시 HOME 디렉토리 설정 (모든 OS 독립적)
	tmpHome := setupTempHome(t)

	// 1. 설정 파일 생성 테스트
	err := GenerateDefaultConfig()
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}

	paths, err := platform.GetPaths()
	if err != nil {
		t.Fatalf("Failed to get platform paths: %v", err)
	}

	configPath := filepath.Join(paths.ConfigDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	// 2. 설정 파일 파싱 테스트
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.RegistryURL != "https://api.github.com" {
		t.Errorf("Expected registry_url to be https://api.github.com, got %s", cfg.RegistryURL)
	}

	expectedInstallPath := filepath.Join(tmpHome, ".local", "bin")
	if cfg.InstallPath != expectedInstallPath {
		t.Errorf("Expected install_path to be %s, got %s", expectedInstallPath, cfg.InstallPath)
	}
}
