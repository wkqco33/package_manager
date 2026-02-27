package config

import (
	"os"
	"path/filepath"

	"ppm/internal/apperr"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	RegistryURL string `yaml:"registry_url"`
	AuthToken   string `yaml:"auth_token"`
	InstallPath string `yaml:"install_path"`
}

var ErrConfigNotFound = apperr.New(apperr.CodeConfig, "configuration file not found")

// LoadConfig reads the config.yaml file from ~/.config/ppm
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeFileSystem, err, "could not get user home directory")
	}

	configPath := filepath.Join(home, ".config", "ppm", "config.yaml")
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

	// Set default InstallPath if empty
	if cfg.InstallPath == "" {
		cfg.InstallPath = filepath.Join(home, ".local", "bin")
	}

	return &cfg, nil
}

// EnsureConfigDir creates ~/.config/ppm if it doesn't exist
func EnsureConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileSystem, err, "could not get user home directory")
	}
	ppmDir := filepath.Join(home, ".config", "ppm")
	if err := os.MkdirAll(ppmDir, 0755); err != nil {
		return "", apperr.Wrap(apperr.CodeFileSystem, err, "failed to create config directory")
	}
	return ppmDir, nil
}

// GenerateDefaultConfig creates a default configuration file
func GenerateDefaultConfig() error {
	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.yaml")

	// Skip if already exists
	if _, err := os.Stat(configPath); err == nil {
		return apperr.New(apperr.CodeConfig, "config.yaml already exists")
	}

	home, _ := os.UserHomeDir()
	cfg := Config{
		RegistryURL: "https://api.github.com",
		AuthToken:   "", // User needs to fill this
		InstallPath: filepath.Join(home, ".local", "bin"),
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

// GetPackagesDir returns the path to the directory where packages are extracted
func GetPackagesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperr.Wrap(apperr.CodeFileSystem, err, "could not get user home directory")
	}
	return filepath.Join(home, ".config", "ppm", "packages"), nil
}
