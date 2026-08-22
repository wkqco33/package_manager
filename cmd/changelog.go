package cmd

import (
	"fmt"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
)

var changelogCmd = &wcli.Command{
	Use: "changelog [package]", Short: "패키지 최신 릴리스 노트 표시",
	Run: func(ctx *wcli.Context) error {
		if len(ctx.Args) != 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(ctx.Args))
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		var fetcher pkg.MetadataFetcher = registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries)
		p, err := fetcher.GetMetadata(ctx.Args[0])
		if err != nil {
			return err
		}
		fmt.Printf("%s %s 릴리스 노트\n\n", p.Name, p.Version)
		if p.ReleaseNotes == "" {
			fmt.Println("릴리스 노트가 없습니다.")
		} else {
			fmt.Println(p.ReleaseNotes)
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(changelogCmd) }
