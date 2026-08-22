package cmd

import (
	"fmt"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/lockfile"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
)

var lockCmd = &wcli.Command{
	Use:   "lock [package...]",
	Short: "의존성과 버전을 ppm.lock에 저장",
	Long:  "지정한 패키지의 현재 메타데이터, 버전, asset, 체크섬을 재현 가능한 lockfile로 저장합니다.",
	Run: func(ctx *wcli.Context) error {
		if len(ctx.Args) == 0 {
			return fmt.Errorf("requires at least 1 package")
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		fetcher := registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries, cfg.TrustedOwners, cfg.RequireChecksum, cfg.RequireSignature, cfg.SignaturePublicKey)
		packages, err := pkg.ResolveDependencies(ctx.Args, fetcher)
		if err != nil {
			return err
		}
		if err := lockfile.Save("ppm.lock", packages); err != nil {
			return fmt.Errorf("ppm.lock 저장 실패: %w", err)
		}
		fmt.Printf("ppm.lock에 %d개 패키지를 저장했습니다.\n", len(packages))
		return nil
	},
}

func init() { rootCmd.AddCommand(lockCmd) }
