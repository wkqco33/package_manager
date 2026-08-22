package cmd

import (
	"fmt"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/manifest"
)

var manifestCmd = &wcli.Command{
	Use:   "manifest",
	Short: "패키지 manifest 관리",
}

var manifestValidateCmd = &wcli.Command{
	Use:   "validate [path]",
	Short: "ppm.json 유효성 검증",
	Run: func(ctx *wcli.Context) error {
		path := "ppm.json"
		if len(ctx.Args) > 0 {
			path = ctx.Args[0]
		}
		m, err := manifest.Load(path)
		if err != nil {
			return err
		}
		if err := m.Validate(); err != nil {
			return err
		}
		fmt.Printf("manifest가 유효합니다: %s\n", path)
		return nil
	},
}

func init() {
	manifestCmd.AddCommand(manifestValidateCmd)
	rootCmd.AddCommand(manifestCmd)
}
