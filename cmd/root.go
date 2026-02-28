package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ppm/internal/apperr"
	"ppm/internal/logger"
)

var rootCmd = &cobra.Command{
	Use:   "ppm",
	Short: "ppm은 프라이빗 패키지 매니저입니다",
	Long:  `GitHub/GitLab을 위한 빠르고 데이터 중심적인 프라이빗 패키지 매니저입니다.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.DebugMode = debugMode
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var debugMode bool

// Execute는 루트 명령 실행과 하위 명령 등록을 수행합니다.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")
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
