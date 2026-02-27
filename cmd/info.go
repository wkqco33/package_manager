package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/registry"
	"ppm/internal/ui"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info [package]",
	Short: "패키지 정보 표시",
	Long:  `원격 레지스트리에서 패키지에 대한 상세 정보를 표시합니다.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pkgName := args[0]

		cfg, err := config.LoadConfig()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		fetcher := &registry.GitHubRegistry{
			Token: cfg.AuthToken,
			URL:   cfg.RegistryURL,
		}

		spinner := ui.NewSpinner(fmt.Sprintf("Fetching metadata for %s...", pkgName))
		spinner.Start()
		p, err := fetcher.GetMetadata(pkgName)
		spinner.Stop()

		if err != nil {
			logger.Error("Failed to get metadata: %v", err)
			os.Exit(1)
		}

		fmt.Println()
		logger.Success(fmt.Sprintf("패키지 정보를 찾았습니다: %s", ui.Highlight(p.Name)))
		fmt.Println()

		fmt.Printf("  %s %-12s %s\n", ui.Highlight("📦"), ui.Label("Name"), p.Name)
		fmt.Printf("  %s %-12s %s\n", ui.Highlight("🏷️"), ui.Label("Version"), ui.Accent(p.Version))

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
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
