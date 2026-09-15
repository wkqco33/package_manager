package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/app"
	"github.com/wkqco33/package_manager/internal/apperr"
	"github.com/wkqco33/package_manager/internal/apps"
	"github.com/wkqco33/package_manager/internal/archive"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
	"github.com/wkqco33/package_manager/internal/ui"
)

type appsDependencies struct {
	GetPackagesDir func() (string, error)
	ListInstalled  func(string) ([]*pkg.Package, error)
	DefaultApps    func() []apps.DefaultApp
	LoadConfig     func() (*config.Config, error)
	NewFetcher     func(*config.Config) pkg.RegistryFetcher
	NewArchiver    app.ArchiverFactory
}

func defaultAppsDependencies() appsDependencies {
	return appsDependencies{
		GetPackagesDir: config.GetPackagesDir,
		ListInstalled:  pkg.ListInstalled,
		DefaultApps:    func() []apps.DefaultApp { return apps.DefaultApps },
		LoadConfig:     config.LoadConfig,
		NewFetcher: func(cfg *config.Config) pkg.RegistryFetcher {
			return registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries, cfg.TrustedOwners, cfg.RequireChecksum, cfg.RequireSignature, cfg.SignaturePublicKey)
		},
		NewArchiver: archive.NewArchiver,
	}
}

// appsCmd는 기본 앱 패키지 소개 및 자동 설치 명령입니다.
var appsCmd = newAppsCommand(defaultAppsDependencies())

var appsJSON bool

type defaultAppJSON struct {
	Name        string `json:"name"`
	BinName     string `json:"bin_name"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	Installed   bool   `json:"installed"`
}

type appsInstallJSONResult struct {
	Installed []string          `json:"installed"`
	Failed    map[string]string `json:"failed,omitempty"`
}

func newAppsCommand(deps appsDependencies) *wcli.Command {
	var installFlag bool
	var allFlag bool
	var dryRunFlag bool
	var concurrencyFlag int
	var jsonFlag bool

	command := &wcli.Command{
		Use:   "apps [package...]",
		Short: "기본 앱 패키지 소개 및 자동 설치",
		Long: `ppm이 추천하는 기본(공개) 앱 패키지 목록을 소개하거나, --install (-i) 플래그를 통해 미설치된 기본 앱들을 고루틴 기반으로 자동 설치합니다.
(예: ppm apps, ppm apps -i, ppm apps -i -a, ppm apps -i note_cli)`,
		Run: func(ctx *wcli.Context) error {
			packagesDir, err := deps.GetPackagesDir()
			if err != nil {
				return err
			}
			installed, err := deps.ListInstalled(packagesDir)
			if err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to read packages directory")
			}
			installedSet := make(map[string]bool, len(installed))
			var installedNames []string
			for _, p := range installed {
				installedSet[p.Name] = true
				installedNames = append(installedNames, p.Name)
			}

			defaultApps := deps.DefaultApps()
			isJSON := appsJSON || jsonFlag

			// 1. --install (-i) 플래그가 지정된 경우 자동 설치 진행
			if installFlag {
				var targets []apps.DefaultApp
				if len(ctx.Args) > 0 {
					var filterErr error
					targets, filterErr = apps.FilterByName(defaultApps, ctx.Args)
					if filterErr != nil {
						return filterErr
					}
				} else if allFlag {
					targets = append([]apps.DefaultApp(nil), defaultApps...)
				} else {
					targets = apps.FilterUninstalled(defaultApps, installedNames)
				}

				if len(targets) == 0 {
					if isJSON {
						return json.NewEncoder(os.Stdout).Encode(appsInstallJSONResult{Installed: []string{}})
					}
					logger.Info("모든 기본 앱이 이미 설치되어 있습니다.")
					return nil
				}

				if deps.LoadConfig == nil || deps.NewFetcher == nil || deps.NewArchiver == nil {
					return errors.New("apps install requires config, fetcher, and archiver")
				}

				cfg, err := deps.LoadConfig()
				if err != nil {
					return err
				}
				fetcher := deps.NewFetcher(cfg)
				if fetcher == nil {
					return errors.New("install command requires a registry fetcher")
				}

				if dryRunFlag {
					if isJSON {
						var planned []string
						for _, t := range targets {
							planned = append(planned, t.Name)
						}
						return json.NewEncoder(os.Stdout).Encode(appsInstallJSONResult{Installed: planned})
					}
					logger.Info("설치 계획 (%d개):", len(targets))
					for _, t := range targets {
						logger.Info("  install %s (%s)", t.Name, t.BinName)
					}
					return nil
				}

				installer := app.AppsInstaller{
					Fetcher:     fetcher,
					InstallPath: cfg.InstallPath,
					NewArchiver: deps.NewArchiver,
					Concurrency: concurrencyFlag,
				}

				if !isJSON {
					logger.Info("기본 앱 %d개 동시 설치 시작...", len(targets))
				}

				result, installErr := installer.InstallApps(targets, false)

				if isJSON {
					jsonRes := appsInstallJSONResult{
						Installed: make([]string, 0, len(result.Installed)),
						Failed:    make(map[string]string),
					}
					for _, p := range result.Installed {
						jsonRes.Installed = append(jsonRes.Installed, p.Name)
					}
					for name, fErr := range result.Failed {
						jsonRes.Failed[name] = fErr.Error()
					}
					_ = json.NewEncoder(os.Stdout).Encode(jsonRes)
					return installErr
				}

				for _, p := range result.Installed {
					logger.Success("  %s (%s) 설치 완료", p.Name, p.Version)
				}
				for name, fErr := range result.Failed {
					logger.Error("  %s 설치 실패: %v", name, fErr)
				}

				if len(result.Installed) > 0 {
					logger.Success("총 %d개의 기본 앱이 성공적으로 설치되었습니다.", len(result.Installed))
				}
				return installErr
			}

			// 2. 일반 조회 모드
			if isJSON {
				result := make([]defaultAppJSON, 0, len(defaultApps))
				for _, a := range defaultApps {
					result = append(result, defaultAppJSON{
						Name:        a.Name,
						BinName:     a.BinName,
						Description: a.Description,
						Homepage:    a.Homepage,
						Installed:   installedSet[a.Name],
					})
				}
				return json.NewEncoder(os.Stdout).Encode(result)
			}

			logger.Info("ppm 기본 앱 패키지:")
			for _, a := range defaultApps {
				status := ui.Muted("미설치")
				if installedSet[a.Name] {
					status = ui.Success("설치됨")
				}
				fmt.Printf("  %s %s %s\n", ui.Highlight("📦"), ui.Label(a.Name), status)
				fmt.Printf("    %s %s\n", ui.Muted(a.Description), ui.Muted(a.Homepage))
			}
			fmt.Println()
			logger.Success("총 %d개의 기본 앱을 소개합니다. 자동 설치: ppm apps --install (-i)", len(defaultApps))
			return nil
		},
	}

	command.Flags().BoolVar(&installFlag, "install", "i", false, "미설치된 기본 앱들을 자동 설치")
	command.Flags().BoolVar(&allFlag, "all", "a", false, "이미 설치된 앱을 포함하여 모든 기본 앱 설치")
	command.Flags().BoolVar(&dryRunFlag, "dry-run", "", false, "설치하지 않고 설치 계획만 표시")
	command.Flags().IntVar(&concurrencyFlag, "concurrency", "c", 4, "동시 설치 고루틴 수")
	command.Flags().BoolVar(&jsonFlag, "json", "", false, "JSON 형식으로 출력")

	return command
}

func init() {
	rootCmd.AddCommand(appsCmd)
}
