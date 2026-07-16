package config

import (
	"os"
	"path/filepath"

	"ppm/internal/apperr"
	"ppm/internal/platform"

	"gopkg.in/yaml.v3"
)

// Config는 애플리케이션 설정입니다.
type Config struct {
	RegistryURL string `yaml:"registry_url"`
	AuthToken   string `yaml:"auth_token"`
	InstallPath string `yaml:"install_path"`
}

var ErrConfigNotFound = apperr.New(apperr.CodeConfig, "configuration file not found")

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
	}

	paths, err := platform.GetPaths()
	if err != nil {
		return err
	}

	cfg := Config{
		RegistryURL: "https://api.github.com",
		AuthToken:   "", // 사용자가 직접 입력
		InstallPath: paths.BinDir,
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return apperr.Wrap(apperr.CodeConfig, err, "failed to format config structure")
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return apperr.Wrap(apperr.CodeFileSystem, err, "failed to write config file securely")
	}
	return nil
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
