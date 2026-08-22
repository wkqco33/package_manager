package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/app"
	"github.com/wkqco33/package_manager/internal/archive"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/lockfile"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
	"github.com/wkqco33/package_manager/internal/ui"
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
			return registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries)
		},
		NewArchiver: archive.NewArchiver,
	}
}

// installCmd는 install 명령입니다.
var installCmd = newInstallCommand(defaultInstallDependencies())

func newInstallCommand(deps installDependencies) *wcli.Command {
	var allowSourceBuild bool
	var installDryRun bool
	var installLocked bool
	var installOffline bool
	var installAtomic bool
	command := &wcli.Command{
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

			var resolvedPackages []*pkg.Package
			if installOffline && !installLocked {
				return fmt.Errorf("--offline requires --locked and ppm.lock")
			}
			if installLocked {
				locked, lockErr := lockfile.Load("ppm.lock")
				if lockErr != nil {
					return fmt.Errorf("ppm.lock을 읽을 수 없습니다: %w", lockErr)
				}
				resolvedPackages = locked.Packages
				if len(ctx.Args) > 0 {
					if !lockedContainsAll(locked.Packages, ctx.Args) {
						return fmt.Errorf("요청한 패키지가 ppm.lock에 없습니다")
					}
					resolvedPackages = filterLockedPackages(resolvedPackages, ctx.Args)
				}
			} else {
				resolvedPackages, err = pkg.ResolveDependencies(ctx.Args, withMetadataProgress(fetcher))
				if err != nil {
					return err
				}
			}
			if installDryRun {
				logger.Info("설치 계획:")
				for _, p := range resolvedPackages {
					logger.Info("  install %s %s", p.Name, p.Version)
				}
				return nil
			}
			installer := app.PackageInstaller{
				Fetcher:          fetcher,
				InstallPath:      cfg.InstallPath,
				NewArchiver:      deps.NewArchiver,
				AllowSourceBuild: allowSourceBuild,
				Offline:          installOffline,
				Atomic:           installAtomic,
			}
			if err := installer.Install(resolvedPackages); err != nil {
				logger.Error("Installation failed: %v", err)
				return err
			}
			return nil
		},
	}
	command.Flags().BoolVar(&allowSourceBuild, "from-source", "", false, "신뢰할 수 있는 저장소의 소스에서만 로컬 빌드 허용")
	command.Flags().BoolVar(&installDryRun, "dry-run", "", false, "설치하지 않고 설치 계획만 표시")
	command.Flags().BoolVar(&installLocked, "locked", "", false, "ppm.lock의 버전과 체크섬만 사용")
	command.Flags().BoolVar(&installOffline, "offline", "", false, "네트워크를 사용하지 않고 캐시에서만 설치")
	command.Flags().BoolVar(&installAtomic, "atomic", "", false, "모든 패키지 성공 시에만 변경사항 적용")
	return command
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

func lockedContainsAll(packages []*pkg.Package, names []string) bool {
	available := make(map[string]bool, len(packages))
	for _, p := range packages {
		available[p.Name] = true
		available[filepath.Base(p.Name)] = true
	}
	for _, name := range names {
		if !available[name] {
			return false
		}
	}
	return true
}

func filterLockedPackages(packages []*pkg.Package, names []string) []*pkg.Package {
	byName := make(map[string]*pkg.Package, len(packages))
	for _, p := range packages {
		byName[p.Name] = p
		byName[filepath.Base(p.Name)] = p
	}
	selected := make(map[*pkg.Package]bool)
	var selectPackage func(*pkg.Package)
	selectPackage = func(p *pkg.Package) {
		if p == nil || selected[p] {
			return
		}
		selected[p] = true
		for _, dep := range p.Dependencies {
			selectPackage(byName[dep])
		}
	}
	for _, name := range names {
		selectPackage(byName[name])
	}
	result := make([]*pkg.Package, 0, len(packages))
	for _, p := range packages {
		if selected[p] {
			result = append(result, p)
		}
	}
	return result
}

func init() {
	rootCmd.AddCommand(installCmd)
}
