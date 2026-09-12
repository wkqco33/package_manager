package cmd

import (
	"fmt"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/app"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/pkg"
)

type uninstallDependencies struct {
	LoadConfig func() (*config.Config, error)
	Remove     app.RemovePackage
	Confirm    func() error
}

func defaultUninstallDependencies() uninstallDependencies {
	return uninstallDependencies{
		LoadConfig: config.LoadConfig,
		Remove:     pkg.Uninstall,
		Confirm:    func() error { return requireDestructiveConfirmation("uninstall", uninstallYes || uninstallForce) },
	}
}

// uninstallCmd는 uninstall 명령입니다.
var uninstallCmd = newUninstallCommand(defaultUninstallDependencies())

func newUninstallCommand(deps uninstallDependencies) *wcli.Command {
	command := &wcli.Command{
		Use:   "uninstall [package...]",
		Short: "설치된 패키지 삭제",
		Long:  `설치된 바이너리 및 패키지 데이터 파일을 시스템에서 완전히 제거합니다. (예: ppm uninstall repo1 repo2)`,
		Run: func(ctx *wcli.Context) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("requires at least 1 arg(s), only received 0")
			}
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			if deps.Remove == nil {
				return fmt.Errorf("uninstall command requires a remove operation")
			}
			if deps.Confirm != nil {
				if err := deps.Confirm(); err != nil {
					return err
				}
			}
			uninstaller := app.PackageUninstaller{
				InstallPath: cfg.InstallPath,
				Concurrency: 5,
				Remove: func(name, installPath string) error {
					logger.Info("Uninstalling %s...", name)
					return deps.Remove(name, installPath)
				},
			}
			if err := uninstaller.Uninstall(ctx.Args); err != nil {
				logger.Error("Uninstall failed: %v", err)
				return err
			}
			return nil
		},
	}
	command.Flags().BoolVar(&uninstallYes, "yes", "y", false, "확인 없이 실행")
	command.Flags().BoolVar(&uninstallForce, "force", "f", false, "확인 없이 강제 실행")
	return command
}

var (
	uninstallYes   bool
	uninstallForce bool
)

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
