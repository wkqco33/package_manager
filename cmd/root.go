package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/wkqco33/wcli"

	"ppm/internal/apperr"
	"ppm/internal/logger"
)

var rootCmd = &wcli.Command{
	Use:           "ppm",
	Short:         "ppm은 프라이빗 패키지 매니저입니다",
	Long:          `GitHub/GitLab을 위한 빠르고 데이터 중심적인 프라이빗 패키지 매니저입니다.`,
	SilenceErrors: true,
	PersistentPreRun: func(ctx *wcli.Context) error {
		logger.DebugMode = debugMode
		return nil
	},
}

var debugMode bool

// Execute는 프로세스의 커맨드라인 인자로 루트 명령을 실행합니다.
func Execute() error {
	return ExecuteArgs(os.Args[1:])
}

// ExecuteArgs는 전달받은 인자로 루트 명령을 실행합니다.
// 프로세스 전역 os.Args에 의존하지 않아 CLI 실행을 독립적으로 테스트할 수 있습니다.
func ExecuteArgs(args []string) error {
	if err := rootCmd.Execute(args); err != nil {
		return fmt.Errorf("execute ppm command: %w", err)
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", "d", false, "Enable debug logging")
	// 루트 전역 플래그는 여기서 설정합니다.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "설정 파일 경로 (기본값: $HOME/.config/ppm/config.yaml)")
}

// PrintError는 가능하면 AppError 정보를 활용해 오류를 출력합니다.
func PrintError(err error) {
	if err == nil {
		return
	}
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		logger.Error("[%s] %s", appErr.Code.String(), appErr.Message)
		if appErr.Err != nil {
			logger.Debug("Underlying error", "error", appErr.Err)
		}
	} else {
		logger.Error("%v", err)
	}
}
