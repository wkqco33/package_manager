package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/pkg"
	"github.com/wkqco33/package_manager/internal/registry"
	"github.com/wkqco33/package_manager/internal/version"
)

type outdatedItem struct {
	Name    string `json:"name"`
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

var outdatedJSON bool
var outdatedCmd = &wcli.Command{Use: "outdated", Short: "업데이트 가능한 패키지 표시", Run: func(ctx *wcli.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	dir, err := config.GetPackagesDir()
	if err != nil {
		return err
	}
	installed, err := pkg.ListInstalled(dir)
	if err != nil {
		return err
	}
	fetcher := registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries, cfg.TrustedOwners, cfg.RequireChecksum, cfg.RequireSignature, cfg.SignaturePublicKey)
	items := make([]outdatedItem, 0)
	for _, p := range installed {
		if p == nil || p.Name == "" || p.Version == "" {
			continue
		}
		latest, fetchErr := fetcher.GetMetadata(p.Name)
		if fetchErr != nil {
			return fmt.Errorf("%s 확인 실패: %w", p.Name, fetchErr)
		}
		if version.Compare(p.Version, latest.Version) < 0 {
			items = append(items, outdatedItem{p.Name, p.Version, latest.Version})
		}
	}
	if outdatedJSON {
		return json.NewEncoder(os.Stdout).Encode(items)
	}
	if len(items) == 0 {
		fmt.Println("모든 패키지가 최신 버전입니다.")
		return nil
	}
	fmt.Println("업데이트 가능한 패키지:")
	for _, item := range items {
		fmt.Printf("  %s %s -> %s\n", item.Name, item.Current, item.Latest)
	}
	return nil
}}

func init() {
	rootCmd.AddCommand(outdatedCmd)
	outdatedCmd.Flags().BoolVar(&outdatedJSON, "json", "", false, "JSON 형식으로 출력")
}
