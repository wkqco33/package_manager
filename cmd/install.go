package cmd

import (
	"fmt"

	"github.com/wkqco33/wcli"

	"ppm/internal/app"
	"ppm/internal/archive"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
	"ppm/internal/registry"
	"ppm/internal/ui"
)

type installDependencies struct {
	LoadConfig  func() (*config.Config, error)
	NewFetcher  func(*config.Config) pkg.RegistryFetcher
	NewArchiver app.ArchiverFactory
}

func defaultInstallDependencies() installDependencies {
	return installDependencies{
		LoadConfig: config.LoadConfig,
		NewFetcher: func(cfg *config.Config) pkg.RegistryFetcher {
			return &registry.GitHubRegistry{Token: cfg.AuthToken, URL: cfg.RegistryURL}
		},
		NewArchiver: archive.NewArchiver,
	}
}

// installCmd는 install 명령입니다.
var installCmd = newInstallCommand(defaultInstallDependencies())

func newInstallCommand(deps installDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "install [package...]",
		Short: "프라이빗 패키지 설치",
		Long:  `설정된 레지스트리에서 하나 이상의 프라이빗 패키지를 다운로드하고 설치합니다. (예: ppm install user/repo1 user/repo2)`,
		Run: func(ctx *wcli.Context) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("requires at least 1 arg(s), only received 0")
			}
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			fetcher := deps.NewFetcher(cfg)
			if fetcher == nil {
				return fmt.Errorf("install command requires a registry fetcher")
			}

			resolvedPackages, err := pkg.ResolveDependencies(ctx.Args, withMetadataProgress(fetcher))
			if err != nil {
				return err
			}
			installer := app.PackageInstaller{
				Fetcher:     fetcher,
				InstallPath: cfg.InstallPath,
				NewArchiver: deps.NewArchiver,
			}
			if err := installer.Install(resolvedPackages); err != nil {
				logger.Error("Installation failed: %v", err)
				return err
			}
			return nil
		},
	}
}

type metadataProgressFetcher struct {
	fetcher pkg.MetadataFetcher
}

func (f metadataProgressFetcher) GetMetadata(name string) (*pkg.Package, error) {
	spinner := ui.NewSpinner("Fetching metadata for " + name + "...")
	spinner.Start()
	defer spinner.Stop()
	return f.fetcher.GetMetadata(name)
}

func withMetadataProgress(fetcher pkg.MetadataFetcher) pkg.MetadataFetcher {
	return metadataProgressFetcher{fetcher: fetcher}
}

func init() {
	rootCmd.AddCommand(installCmd)
}
