package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/apperr"
	"github.com/wkqco33/package_manager/internal/apps"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/ui"
)

type appsDependencies struct {
	GetPackagesDir func() (string, error)
	ListInstalled  func(string) ([]*pkg.Package, error)
	DefaultApps    func() []apps.DefaultApp
}

func defaultAppsDependencies() appsDependencies {
	return appsDependencies{
		GetPackagesDir: config.GetPackagesDir,
		ListInstalled:  pkg.ListInstalled,
		DefaultApps:    func() []apps.DefaultApp { return apps.DefaultApps },
	}
}

// appsCmd는 기본 앱 패키지 소개 명령입니다.
var appsCmd = newAppsCommand(defaultAppsDependencies())

var appsJSON bool

type defaultAppJSON struct {
	Name        string `json:"name"`
	BinName     string `json:"bin_name"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	Installed   bool   `json:"installed"`
}

func newAppsCommand(deps appsDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "apps",
		Short: "기본 앱 패키지 소개",
		Long:  `ppm이 추천하는 기본(공개) 앱 패키지 목록을 소개하고 설치 상태를 표시합니다.`,
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
			for _, p := range installed {
				installedSet[p.Name] = true
			}

			defaultApps := deps.DefaultApps()

			if appsJSON {
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
			logger.Success("총 %d개의 기본 앱을 소개합니다. 설치: ppm install <owner/repo>", len(defaultApps))
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(appsCmd)
	appsCmd.Flags().BoolVar(&appsJSON, "json", "", false, "JSON 형식으로 출력")
}
