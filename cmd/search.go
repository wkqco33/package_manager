package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/config"
	"github.com/wkqco33/package_manager/internal/registry"
)

var searchJSON bool
var searchCmd = &wcli.Command{
	Use: "search [query]", Short: "GitHub 패키지 저장소 검색",
	Run: func(ctx *wcli.Context) error {
		if len(ctx.Args) == 0 {
			return fmt.Errorf("requires at least 1 search term")
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		results, err := registry.NewGitHubRegistry(cfg.AuthToken, cfg.RegistryURL, cfg.Registries, cfg.TrustedOwners, cfg.RequireChecksum, cfg.RequireSignature, cfg.SignaturePublicKey).Search(ctx.Args[0])
		if err != nil {
			return err
		}
		if searchJSON {
			return json.NewEncoder(os.Stdout).Encode(results)
		}
		if len(results) == 0 {
			fmt.Println("검색 결과가 없습니다.")
			return nil
		}
		for _, result := range results {
			fmt.Printf("%-35s %s\n", result.Name, result.Description)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().BoolVar(&searchJSON, "json", "", false, "JSON 형식으로 출력")
}
