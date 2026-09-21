package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wkqco33/package_manager/internal/platform"
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
	for _, name := range []string{
		"PPM_CONFIG_DIR",
		"PPM_INSTALL_DIR",
		"PPM_CACHE_DIR",
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
	} {
		t.Setenv(name, "")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
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

// ReadConfigToken은 config.yaml의 원시 auth_token만 반환해야 합니다.
// LoadConfig는 환경 변수와 credential store·gh CLI까지 반영하므로,
// ppm auth status가 소스별 감지 여부를 구분할 수 없습니다.
func TestReadConfigTokenIgnoresResolvedSources(t *testing.T) {
	setupTempHome(t)

	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatalf("DefaultConfig() error = %v", err)
	}
	if err := SetValue(cfg, "auth_token", "config-token"); err != nil {
		t.Fatalf("SetValue() error = %v", err)
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	t.Setenv("PPM_AUTH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "environment-token")

	token, err := ReadConfigToken()
	if err != nil {
		t.Fatalf("ReadConfigToken() error = %v", err)
	}
	if token != "config-token" {
		t.Errorf("ReadConfigToken() = %q, want config-token", token)
	}

	// LoadConfig는 환경 변수를 우선하므로 해석된 값이 달라집니다.
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.AuthToken != "environment-token" {
		t.Errorf("LoadConfig().AuthToken = %q, want environment-token", loaded.AuthToken)
	}
}

func TestReadConfigTokenWithoutConfigFile(t *testing.T) {
	setupTempHome(t)

	if _, err := ReadConfigToken(); !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("ReadConfigToken() error = %v, want ErrConfigNotFound", err)
	}
}
