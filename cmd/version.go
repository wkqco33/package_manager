package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version은 ppm의 현재 버전입니다.
// 빌드 시 -ldflags로 덮어쓸 수 있습니다.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "ppm의 버전 번호 출력",
	Long:  `ppm의 버전 번호를 출력합니다.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ppm version %s %s/%s\n", Version, runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
