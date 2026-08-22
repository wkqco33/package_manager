package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wkqco33/wcli"
)

type manifest struct {
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Homepage     string            `json:"homepage"`
	BinName      string            `json:"bin_name"`
	Dependencies map[string]string `json:"dependencies"`
}

var manifestCmd = &wcli.Command{Use: "manifest", Short: "패키지 manifest 관리"}
var manifestValidateCmd = &wcli.Command{
	Use: "validate [path]", Short: "ppm.json 유효성 검증",
	Run: func(ctx *wcli.Context) error {
		path := "ppm.json"
		if len(ctx.Args) > 0 {
			path = ctx.Args[0]
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("manifest를 읽을 수 없습니다: %w", err)
		}
		var m manifest
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("잘못된 JSON: %w", err)
		}
		if strings.TrimSpace(m.BinName) == "" {
			return fmt.Errorf("bin_name은 필수입니다")
		}
		if strings.ContainsAny(m.BinName, `/\\`) || m.BinName == "." || m.BinName == ".." {
			return fmt.Errorf("bin_name에 경로를 사용할 수 없습니다")
		}
		for name, constraint := range m.Dependencies {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(constraint) == "" {
				return fmt.Errorf("의존성 이름과 버전 제약은 비어 있을 수 없습니다")
			}
		}
		fmt.Printf("manifest가 유효합니다: %s\n", path)
		return nil
	},
}

func init() { manifestCmd.AddCommand(manifestValidateCmd); rootCmd.AddCommand(manifestCmd) }
