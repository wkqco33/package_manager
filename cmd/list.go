package cmd

import (
	"fmt"
	"os"

	"ppm/internal/apperr"
	"ppm/internal/config"
	"ppm/internal/logger"
	"ppm/internal/pkg"
	"ppm/internal/ui"

	"github.com/spf13/cobra"
)

// listCmd는 list 명령입니다.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "설치된 패키지 목록 표시",
	Long:  `현재 ppm을 통해 설치된 모든 패키지의 목록을 표시합니다.`,
	Run: func(cmd *cobra.Command, args []string) {
		packagesDir, err := config.GetPackagesDir()
		if err != nil {
			PrintError(err)
			os.Exit(1)
		}

		installed, err := pkg.ListInstalled(packagesDir)
		if err != nil {
			PrintError(apperr.Wrap(apperr.CodeFileSystem, err, "failed to read packages directory"))
			os.Exit(1)
		}

		count := 0
		logger.Info("설치된 패키지 목록:")

		// 메타데이터 기반 목록 출력
		for _, p := range installed {
			fmt.Printf("  %s %s (%s)\n", ui.Highlight("📦"), ui.Label(p.Name), ui.Muted(p.Version))
			count++
		}

		// 메타데이터 없는 패키지(레거시/수동 설치) 보조 처리
		if count == 0 {
			entries, _ := os.ReadDir(packagesDir)
			for _, e := range entries {
				if e.IsDir() {
					fmt.Printf("  %s %s\n", ui.Highlight("📦"), ui.Label(e.Name()))
					count++
				}
			}
		}

		if count == 0 {
			logger.Info("설치된 패키지가 없습니다.")
		} else {
			fmt.Println()
			logger.Success("%s", fmt.Sprintf("총 %d개의 패키지가 설치되어 있습니다.", count))
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
