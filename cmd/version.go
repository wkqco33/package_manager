package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/wkqco33/wcli"
)

// Version은 ppm의 현재 버전입니다.
// 빌드 시 -ldflags로 덮어쓸 수 있습니다. (예: task build)
var Version = "dev"

// resolveVersion은 표시할 버전을 결정합니다.
// 1순위: -ldflags로 주입된 Version (task build / task install)
// 2순위: go install module@version 으로 설치 시 모듈 버전 (debug build info)
// 그 외에는 "dev"를 그대로 사용합니다.
func resolveVersion() string {
	if Version != "dev" && Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return Version
}

var versionCmd = &wcli.Command{
	Use:   "version",
	Short: "ppm의 버전 번호 출력",
	Long:  `ppm의 버전 번호를 출력합니다.`,
	Run: func(ctx *wcli.Context) error {
		fmt.Printf("ppm version %s %s/%s\n", resolveVersion(), runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
