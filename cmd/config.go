package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wkqco33/wcli"
	"gopkg.in/yaml.v3"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
)

type configDependencies struct {
	LoadConfig    func() (*config.Config, error)
	DefaultConfig func() (*config.Config, error)
	SaveConfig    func(*config.Config) error
	SetValue      func(*config.Config, string, string) error
}

func defaultConfigDependencies() configDependencies {
	return configDependencies{
		LoadConfig:    config.LoadConfig,
		DefaultConfig: config.DefaultConfig,
		SaveConfig:    config.SaveConfig,
		SetValue:      config.SetValue,
	}
}

var configCmd = newConfigCommand(defaultConfigDependencies())

func newConfigCommand(deps configDependencies) *wcli.Command {
	showCmd := newConfigShowCommand(deps)
	setCmd := newConfigSetCommand(deps)
	command := &wcli.Command{
		Use:   "config",
		Short: "ppm 설정 관리",
		Long:  "ppm 설정을 확인하거나 변경합니다.",
	}
	command.AddCommand(showCmd, setCmd)
	return command
}

func newConfigShowCommand(deps configDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "show",
		Short: "현재 설정 확인",
		Run: func(ctx *wcli.Context) error {
			if len(ctx.Args) != 0 {
				return fmt.Errorf("config show accepts no arguments")
			}
			cfg, err := deps.LoadConfig()
			if err != nil {
				if errors.Is(err, config.ErrConfigNotFound) {
					return fmt.Errorf("설정 파일을 찾을 수 없습니다. 'ppm init' 또는 'ppm config set <key> <value>'를 먼저 실행해주세요: %w", err)
				}
				return err
			}
			return printConfig(cfg)
		},
	}
}

func newConfigSetCommand(deps configDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "set <key> <value>",
		Short: "설정값 변경",
		Run: func(ctx *wcli.Context) error {
			if len(ctx.Args) != 2 {
				return fmt.Errorf("usage: ppm config set <key> <value>")
			}

			cfg, err := deps.LoadConfig()
			if err != nil {
				if !errors.Is(err, config.ErrConfigNotFound) {
					return err
				}
				cfg, err = deps.DefaultConfig()
				if err != nil {
					return err
				}
			}
			if err := deps.SetValue(cfg, ctx.Args[0], ctx.Args[1]); err != nil {
				return err
			}
			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}
			logger.Success("설정이 변경되었습니다: %s", ctx.Args[0])
			return nil
		},
	}
}

func printConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration must not be nil")
	}
	masked := *cfg
	if masked.AuthToken != "" {
		masked.AuthToken = strings.Repeat("*", 8)
	}
	data, err := yaml.Marshal(&masked)
	if err != nil {
		return fmt.Errorf("failed to format configuration: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
