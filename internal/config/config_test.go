package config

import (
	"os"
	"path/filepath"
	"ppm/internal/platform"
	"testing"
)

func TestGenerateAndLoadConfig(t *testing.T) {
	// 임시 HOME 디렉토리 설정
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome) // Go 1.17+ 에서는 t.Setenv로 환경변수 안전한 조작 가능

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
