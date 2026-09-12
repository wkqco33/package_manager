package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
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
	PasswordStdin io.Reader
	ReadFile      func(string) ([]byte, error)
}

func defaultConfigDependencies() configDependencies {
	return configDependencies{
		LoadConfig:    config.LoadConfig,
		DefaultConfig: config.DefaultConfig,
		SaveConfig:    config.SaveConfig,
		SetValue:      config.SetValue,
		PasswordStdin: os.Stdin,
		ReadFile:      os.ReadFile,
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
	var passwordStdin bool
	var passwordFile string
	command := &wcli.Command{
		Use:   "set <key> [<value>]",
		Short: "설정값 변경",
		Run: func(ctx *wcli.Context) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("usage: ppm config set <key> <value>")
			}
			key := ctx.Args[0]
			value, err := configValue(ctx.Args, key, passwordStdin, passwordFile, deps)
			if err != nil {
				return err
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
			if err := deps.SetValue(cfg, key, value); err != nil {
				return err
			}
			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}
			logger.Success("설정이 변경되었습니다: %s", key)
			return nil
		},
	}
	command.Flags().BoolVar(&passwordStdin, "password-stdin", "", false, "auth_token을 표준 입력에서 읽기")
	command.Flags().StringVar(&passwordFile, "password-file", "", "", "auth_token을 파일에서 읽기")
	return command
}

func configValue(args []string, key string, passwordStdin bool, passwordFile string, deps configDependencies) (string, error) {
	if key != "auth_token" {
		if passwordStdin || passwordFile != "" {
			return "", fmt.Errorf("--password-stdin과 --password-file은 auth_token에만 사용할 수 있습니다")
		}
		if len(args) != 2 {
			return "", fmt.Errorf("usage: ppm config set <key> <value>")
		}
		return args[1], nil
	}
	if passwordStdin && passwordFile != "" {
		return "", fmt.Errorf("--password-stdin과 --password-file은 함께 사용할 수 없습니다")
	}
	if passwordStdin {
		if len(args) != 1 {
			return "", fmt.Errorf("--password-stdin은 auth_token 값과 함께 사용할 수 없습니다")
		}
		reader := deps.PasswordStdin
		if reader == nil {
			reader = os.Stdin
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("auth_token 표준 입력을 읽을 수 없습니다: %w", err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("auth_token 표준 입력이 비어 있습니다")
		}
		return value, nil
	}
	if passwordFile != "" {
		if len(args) != 1 {
			return "", fmt.Errorf("--password-file은 auth_token 값과 함께 사용할 수 없습니다")
		}
		readFile := deps.ReadFile
		if readFile == nil {
			readFile = os.ReadFile
		}
		data, err := readFile(passwordFile)
		if err != nil {
			return "", fmt.Errorf("auth_token 파일을 읽을 수 없습니다: %w", err)
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return "", fmt.Errorf("auth_token 파일이 비어 있습니다")
		}
		return value, nil
	}
	if len(args) != 2 {
		return "", fmt.Errorf("usage: ppm config set auth_token <value> (또는 --password-stdin/--password-file)")
	}
	logger.Warn("auth_token을 명령줄 인자로 전달하면 셸 기록과 프로세스 목록에 노출될 수 있습니다. --password-stdin 또는 --password-file 사용을 권장합니다.")
	return args[1], nil
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
