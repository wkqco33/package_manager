package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/app"
	"github.com/wkqco33/package_manager/internal/archive"
	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
)

const selfUpdatePackage = "wkqco33/package_manager"

var selfUpdateCmd = &wcli.Command{
	Use:   "self-update",
	Short: "ppm 자신을 최신 버전으로 업데이트",
	Long:  "GitHub Release에서 현재 운영체제에 맞는 ppm 바이너리를 내려받아 교체합니다.",
	Run: func(ctx *wcli.Context) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		current, err := os.Executable()
		if err != nil {
			return fmt.Errorf("현재 ppm 실행 경로를 확인할 수 없습니다: %w", err)
		}
		// 설정된 설치 경로를 우선 사용합니다. Unix에서 os.Executable()은
		// 심볼릭 링크의 실제 대상을 반환할 수 있어 그대로 사용하면 패키지
		// 저장소 내부에 새 링크를 만들 위험이 있습니다.
		installPath := cfg.InstallPath
		if installPath == "" {
			installPath = filepath.Dir(current)
		}
		fetcher := registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries, cfg.TrustedOwners, cfg.RequireChecksum, cfg.RequireSignature, cfg.SignaturePublicKey)
		p, err := fetcher.GetMetadata(selfUpdatePackage)
		if err != nil {
			return err
		}
		if resolveVersion() == p.Version {
			logger.Info("이미 최신 버전입니다: %s", p.Version)
			return nil
		}
		logger.Info("ppm 업데이트: %s -> %s", resolveVersion(), p.Version)
		installer := app.PackageInstaller{Fetcher: fetcher, InstallPath: installPath, NewArchiver: archive.NewArchiver}
		if err := installer.Install([]*pkg.Package{p}); err != nil {
			return fmt.Errorf("ppm 업데이트 실패: %w", err)
		}
		logger.Success("ppm을 %s 버전으로 업데이트했습니다.", p.Version)
		return nil
	},
}

func init() { rootCmd.AddCommand(selfUpdateCmd) }
