package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
	"github.com/wkqco33/package_manager/internal/ui"
)

type infoDependencies struct {
	LoadConfig func() (*config.Config, error)
	NewFetcher func(*config.Config) pkg.MetadataFetcher
}

func defaultInfoDependencies() infoDependencies {
	return infoDependencies{
		LoadConfig: config.LoadConfig,
		NewFetcher: func(cfg *config.Config) pkg.MetadataFetcher {
			return registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries, cfg.TrustedOwners, cfg.RequireChecksum, cfg.RequireSignature, cfg.SignaturePublicKey)
		},
	}
}

// infoCmd는 info 명령입니다.
var infoCmd = newInfoCommand(defaultInfoDependencies())

var infoJSON bool

func newInfoCommand(deps infoDependencies) *wcli.Command {
	return &wcli.Command{
		Use:   "info [package]",
		Short: "패키지 정보 표시",
		Long:  `원격 레지스트리에서 패키지에 대한 상세 정보를 표시합니다.`,
		Run: func(ctx *wcli.Context) error {
			if len(ctx.Args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(ctx.Args))
			}
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			fetcher := deps.NewFetcher(cfg)
			if fetcher == nil {
				return fmt.Errorf("info command requires a metadata fetcher")
			}
			pkgName := ctx.Args[0]
			spinner := ui.NewSpinner(fmt.Sprintf("Fetching metadata for %s...", pkgName))
			spinner.Start()
			p, err := fetcher.GetMetadata(pkgName)
			spinner.Stop()
			if err != nil {
				return err
			}
			if infoJSON {
				return json.NewEncoder(os.Stdout).Encode(p)
			}

			fmt.Println()
			logger.Success("패키지 정보를 찾았습니다: %s", ui.Highlight(p.Name))
			fmt.Println()
			fmt.Printf("  %s %-12s %s\n", ui.Highlight("📦"), ui.Label("Name"), p.Name)
			fmt.Printf("  %s %-12s %s\n", ui.Highlight("🏷️"), ui.Label("Version"), ui.Accent(p.Version))
			if p.BinName != "" {
				fmt.Printf("  %s %-12s %s\n", ui.Highlight("⚙️ "), ui.Label("Binary"), p.BinName)
			}
			if p.Description != "" {
				fmt.Printf("  %s %-12s %s\n", ui.Highlight("📝"), ui.Label("Description"), p.Description)
			}
			if p.Author != "" {
				fmt.Printf("  %s %-12s %s\n", ui.Highlight("👤"), ui.Label("Author"), p.Author)
			}
			if p.Homepage != "" {
				fmt.Printf("  %s %-12s %s\n", ui.Highlight("🔗"), ui.Label("Homepage"), ui.Path(p.Homepage))
			}
			fmt.Printf("  %s %-12s %s\n", ui.Highlight("🚀"), ui.Label("Source"), ui.Muted(p.Source))
			fmt.Println()
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().BoolVar(&infoJSON, "json", "", false, "JSON 형식으로 출력")
}
