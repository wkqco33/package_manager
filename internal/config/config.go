package config

import (
	"os"
	"path/filepath"

	"github.com/wkqco33/package_manager/internal/apperr"
	"github.com/wkqco33/package_manager/internal/platform"

	"gopkg.in/yaml.v3"
)

// Config는 애플리케이션 설정입니다.
type Config struct {
	RegistryURL     string   `yaml:"registry_url"`
	Registries      []string `yaml:"registries,omitempty"`
	TrustedOwners   []string `yaml:"trusted_owners,omitempty"`
	RequireChecksum bool     `yaml:"require_checksum,omitempty"`
	AuthToken       string   `yaml:"auth_token"`
	InstallPath     string   `yaml:"install_path"`
}

var ErrConfigNotFound = apperr.New(apperr.CodeConfig, "configuration file not found")

// DefaultConfig는 플랫폼 기본 경로를 사용하는 기본 설정을 반환합니다.
func DefaultConfig() (*Config, error) {
	paths, err := platform.GetPaths()
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeFileSystem, err, "could not get platform paths")
	}
	return &Config{
		RegistryURL: "https://api.github.com",
		InstallPath: paths.BinDir,
	}, nil
}

// SaveConfig는 설정 디렉터리를 만들고 config.yaml을 안전한 권한으로 저장합니다.
func SaveConfig(cfg *Config) error {
	if cfg == nil {
		return apperr.New(apperr.CodeConfig, "configuration must not be nil")
	}
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return apperr.Wrap(apperr.CodeConfig, err, "failed to format config structure")
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to write config file securely")
	}
	if err := os.Chmod(configPath, 0600); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to secure config file")
	}
	return nil
}

// SetValue는 설정 키 하나의 값을 변경합니다.
func SetValue(cfg *Config, key, value string) error {
	if cfg == nil {
		return apperr.New(apperr.CodeConfig, "configuration must not be nil")
	}
	switch key {
	case "registry_url":
		cfg.RegistryURL = value
	case "auth_token":
		cfg.AuthToken = value
	case "install_path":
		cfg.InstallPath = value
	default:
		return apperr.New(apperr.CodeInvalidInput, "unsupported configuration key: %s", key)
	}
	return nil
}

// LoadConfig는 config.yaml을 읽습니다.
func LoadConfig() (*Config, error) {
	paths, err := platform.GetPaths()
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeFileSystem, err, "could not get platform paths")
	}

	configPath := filepath.Join(paths.ConfigDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, apperr.Wrap(apperr.CodeFileSystem, err, "failed to read config file")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, apperr.Wrap(apperr.CodeConfig, err, "failed to parse config.yaml")
	}

	// 환경 변수는 CI/CD에서 안전하게 주입할 수 있도록 설정 파일보다 우선합니다.
	if token := os.Getenv("PPM_AUTH_TOKEN"); token != "" {
		cfg.AuthToken = token
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" && cfg.AuthToken == "" {
		cfg.AuthToken = token
	}
	if registryURL := os.Getenv("PPM_REGISTRY_URL"); registryURL != "" {
		cfg.RegistryURL = registryURL
	}

	// InstallPath가 비어 있으면 기본값 설정
	if cfg.InstallPath == "" {
		cfg.InstallPath = paths.BinDir
	}

	return &cfg, nil
}

// EnsureConfigDir는 설정 디렉터리가 없으면 생성합니다.
func EnsureConfigDir() (string, error) {
	paths, err := platform.GetPaths()
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileSystem, err, "could not get platform paths")
	}
	if err := os.MkdirAll(paths.ConfigDir, 0755); err != nil {
		return "", apperr.Wrap(apperr.CodeFileSystem, err, "failed to create config directory")
	}
	return paths.ConfigDir, nil
}

// GenerateDefaultConfig는 기본 설정 파일을 생성합니다.
func GenerateDefaultConfig() error {
	configDir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// 이미 존재하면 생성하지 않음
	if _, err := os.Stat(configPath); err == nil {
		return apperr.New(apperr.CodeConfig, "config.yaml already exists")
	} else if !os.IsNotExist(err) {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to inspect config file")
	}

	cfg, err := DefaultConfig()
	if err != nil {
		return err
	}
	return SaveConfig(cfg)
}

// GetPackagesDir는 패키지 압축 해제 디렉터리 경로를 반환합니다.
func GetPackagesDir() (string, error) {
	paths, err := platform.GetPaths()
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileSystem, err, "could not get platform paths")
	}
	return paths.PackageDir, nil
}

// GetCacheDir는 로컬 다운로드 캐시 디렉터리 경로를 반환합니다.
func GetCacheDir() (string, error) {
	paths, err := platform.GetPaths()
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileSystem, err, "could not get platform paths")
	}
	return paths.CacheDir, nil
}
