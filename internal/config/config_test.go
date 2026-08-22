package config

import (
	"github.com/wkqco33/package_manager/internal/platform"
	"os"
	"path/filepath"
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

func TestSetValueAndSaveConfig(t *testing.T) {
	setupTempHome(t)

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error = %v", err)
	}
	if err := SetValue(cfg, "auth_token", "secret-token"); err != nil {
		t.Fatalf("SetValue() error = %v", err)
	}
	if err := SetValue(cfg, "registry_url", "https://registry.example.com"); err != nil {
		t.Fatalf("SetValue() error = %v", err)
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.RegistryURL != "https://registry.example.com" {
		t.Errorf("RegistryURL = %q, want updated URL", loaded.RegistryURL)
	}
	if loaded.AuthToken != "secret-token" {
		t.Errorf("AuthToken = %q, want updated token", loaded.AuthToken)
	}
	if loaded.InstallPath != cfg.InstallPath {
		t.Errorf("InstallPath = %q, want %q", loaded.InstallPath, cfg.InstallPath)
	}
}

func TestSetValueRejectsUnknownKey(t *testing.T) {
	if err := SetValue(&Config{}, "unknown", "value"); err == nil {
		t.Fatal("SetValue() error = nil, want unsupported key error")
	}
}

func TestSaveConfigUsesSecurePermissions(t *testing.T) {
	setupTempHome(t)
	if err := SaveConfig(&Config{RegistryURL: "https://api.github.com"}); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	paths, err := platform.GetPaths()
	if err != nil {
		t.Fatalf("GetPaths() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(paths.ConfigDir, "config.yaml"))
	if err != nil {
		t.Fatalf("Stat(config.yaml) error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Errorf("config permissions = %o, want 600", got)
	}
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
