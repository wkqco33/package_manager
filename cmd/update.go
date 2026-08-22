package cmd

import (
	"fmt"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/app"
	"github.com/wkqco33/package_manager/internal/archive"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
	"github.com/wkqco33/package_manager/internal/version"
)

type updateDependencies struct {
	LoadConfig     func() (*config.Config, error)
	GetPackagesDir func() (string, error)
	ListInstalled  func(string) ([]*pkg.Package, error)
	NewFetcher     func(*config.Config) pkg.RegistryFetcher
	NewArchiver    app.ArchiverFactory
}

func defaultUpdateDependencies() updateDependencies {
	return updateDependencies{
		LoadConfig:     config.LoadConfig,
		GetPackagesDir: config.GetPackagesDir,
		ListInstalled:  pkg.ListInstalled,
		NewFetcher: func(cfg *config.Config) pkg.RegistryFetcher {
			return registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries)
		},
		NewArchiver: archive.NewArchiver,
	}
}

// updateCmd는 update 명령입니다.
var updateCmd = newUpdateCommand(defaultUpdateDependencies())

var updateCheck bool

func newUpdateCommand(deps updateDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "update [package...]",
		Short: "설치된 패키지 업데이트",
		Long:  `설치된 패키지를 최신 버전으로 업데이트합니다. 인자가 없으면 모든 패키지를 업데이트합니다.`,
		Run: func(ctx *wcli.Context) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			packagesDir, err := deps.GetPackagesDir()
			if err != nil {
				return err
			}
			installed, err := deps.ListInstalled(packagesDir)
			if err != nil {
				return err
			}
			fetcher := deps.NewFetcher(cfg)
			if fetcher == nil {
				return fmt.Errorf("update command requires a registry fetcher")
			}
			if updateCheck {
				for _, installedPackage := range installed {
					latest, fetchErr := fetcher.GetMetadata(installedPackage.Name)
					if fetchErr != nil {
						return fetchErr
					}
					if version.Compare(installedPackage.Version, latest.Version) < 0 {
						logger.Info("%s: %s -> %s", installedPackage.Name, installedPackage.Version, latest.Version)
					}
				}
				return nil
			}
			updater := app.PackageUpdater{
				Fetcher:     fetcher,
				InstallPath: cfg.InstallPath,
				NewArchiver: deps.NewArchiver,
			}
			result, err := updater.Update(installed, ctx.Args)
			if err != nil {
				logger.Error("Update failed: %v", err)
				return err
			}
			if result.Legacy > 0 {
				logger.Warn("%d개의 레거시 패키지는 자동 업데이트를 지원하지 않습니다.", result.Legacy)
			}
			if result.Updated == 0 && result.Skipped == 0 {
				logger.Info("업데이트할 패키지가 없습니다.")
				return nil
			}
			logger.Success("업데이트가 완료되었습니다. (설치: %d, 최신 상태: %d)", result.Updated, result.Skipped)
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateCheck, "check", "", false, "업데이트하지 않고 최신 버전만 확인")
}
