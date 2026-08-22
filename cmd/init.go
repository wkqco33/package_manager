package cmd

import (
	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/platform"
)

type initDependencies struct {
	GenerateConfig func() error
	GetPaths       func() (*platform.Paths, error)
}

func defaultInitDependencies() initDependencies {
	return initDependencies{
		GenerateConfig: config.GenerateDefaultConfig,
		GetPaths:       platform.GetPaths,
	}
}

// initCmd는 init 명령입니다.
var initCmd = newInitCommand(defaultInitDependencies())

func newInitCommand(deps initDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "init",
		Short: "ppm 설정 초기화",
		Long:  `ppm 설정 파일 및 디렉토리를 플랫폼에 맞게 초기화합니다.`,
		Run: func(ctx *wcli.Context) error {
			logger.Info("Initializing ppm configuration...")
			if err := deps.GenerateConfig(); err != nil {
				return err
			}
			paths, err := deps.GetPaths()
			if err != nil {
				return err
			}
			logger.Success("Configuration file created successfully at %s/config.yaml", paths.ConfigDir)
			logger.Info("Please edit the file to add your AuthToken.")
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
