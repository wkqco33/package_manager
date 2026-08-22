package cmd

import "github.com/wkqco33/wcli"

// completionCmd는 wcli가 ppm의 실제 명령 트리와 플래그를 바탕으로
// Bash/Zsh/Fish 자동완성 스크립트를 생성하도록 위임합니다.
var completionCmd = wcli.NewCompletionCommand(rootCmd)

func init() { rootCmd.AddCommand(completionCmd) }
