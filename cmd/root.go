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

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")
	// Root 전역 플래그를 여기서 설정할 수 있습니다.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/ppm/config.yaml)")
}

// PrintError nicely formats and prints an error, leveraging AppError if possible
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
