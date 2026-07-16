package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"ppm/internal/archive"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
	"ppm/internal/registry"
	"ppm/internal/ui"
)

// installCmd는 install 명령입니다.
var installCmd = &cobra.Command{
	Use:   "install [package...]",
	Short: "프라이빗 패키지 설치",
	Long:  `설정된 레지스트리에서 하나 이상의 프라이빗 패키지를 병렬로 다운로드하고 설치합니다. (예: ppm install user/repo1 user/repo2)`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		fetcher := &registry.GitHubRegistry{
			Token: cfg.AuthToken,
			URL:   cfg.RegistryURL,
		}

		// 의존 관계를 포함한 모든 패키지의 설치 순서 해석 (위상 정렬)
		resolvedPackages, err := resolveDependencies(args, fetcher)
		if err != nil {
			logger.Error("Failed to resolve dependencies: %v", err)
			os.Exit(1)
		}

		hasError := false
		for _, p := range resolvedPackages {
			safeName := filepath.Base(p.Name)
			binName := safeName
			if p.BinName != "" {
				binName = p.BinName
			}
			archiver := archive.NewArchiver(p.Source, binName)

			if err := pkg.InstallWithPackage(p, fetcher, archiver, cfg.InstallPath); err != nil {
				logger.Error("Installation failed: [%s] %v", p.Name, err)
				hasError = true
			}
		}

		if hasError {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

// resolveDependencies는 위상 정렬(Topology Sort) 알고리즘을 사용해 의존성 순서대로 정렬된 패키지 목록을 구합니다.
func resolveDependencies(pkgNames []string, fetcher *registry.GitHubRegistry) ([]*pkg.Package, error) {
	var resolved []*pkg.Package
	visited := make(map[string]bool)
	tempVisited := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if tempVisited[name] {
			return fmt.Errorf("circular dependency detected: %s", name)
		}
		if visited[name] {
			return nil
		}

		tempVisited[name] = true

		spinner := ui.NewSpinner("Fetching metadata for " + name + "...")
		spinner.Start()
		p, err := fetcher.GetMetadata(name)
		spinner.Stop()
		if err != nil {
			return err
		}

		for _, dep := range p.Dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}

		tempVisited[name] = false
		visited[name] = true
		resolved = append(resolved, p)
		return nil
	}

	for _, name := range pkgNames {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}
