package cmd

import (
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"ppm/internal/apperr"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
	"ppm/internal/ui"
)

type listDependencies struct {
	GetPackagesDir func() (string, error)
	ListInstalled  func(string) ([]*pkg.Package, error)
	ReadDir        func(string) ([]os.DirEntry, error)
}

func defaultListDependencies() listDependencies {
	return listDependencies{
		GetPackagesDir: config.GetPackagesDir,
		ListInstalled:  pkg.ListInstalled,
		ReadDir:        os.ReadDir,
	}
}

// listCmd는 list 명령입니다.
var listCmd = newListCommand(defaultListDependencies())

func newListCommand(deps listDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "list",
		Short: "설치된 패키지 목록 표시",
		Long:  `현재 ppm을 통해 설치된 모든 패키지의 목록을 표시합니다.`,
		Run: func(ctx *wcli.Context) error {
			packagesDir, err := deps.GetPackagesDir()
			if err != nil {
				return err
			}
			installed, err := deps.ListInstalled(packagesDir)
			if err != nil {
				return apperr.Wrap(apperr.CodeFileSystem, err, "failed to read packages directory")
			}

			count := 0
			logger.Info("설치된 패키지 목록:")
			for _, p := range installed {
				fmt.Printf("  %s %s (%s)\n", ui.Highlight("📦"), ui.Label(p.Name), ui.Muted(p.Version))
				count++
			}
			if count == 0 {
				entries, err := deps.ReadDir(packagesDir)
				if err != nil {
					return apperr.Wrap(apperr.CodeFileSystem, err, "failed to read packages directory")
				}
				for _, entry := range entries {
					if entry.IsDir() {
						fmt.Printf("  %s %s\n", ui.Highlight("📦"), ui.Label(entry.Name()))
						count++
					}
				}
			}
			if count == 0 {
				logger.Info("설치된 패키지가 없습니다.")
			} else {
				fmt.Println()
				logger.Success("총 %d개의 패키지가 설치되어 있습니다.", count)
			}
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
}
