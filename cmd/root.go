package cmd

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/wkqco33/wcli"

	"github.com/wkqco33/package_manager/internal/apperr"
	"github.com/wkqco33/package_manager/internal/logger"
	"github.com/wkqco33/package_manager/internal/ui"
)

var rootCmd = &wcli.Command{
	Use:           "ppm",
	Short:         "ppm은 프라이빗 패키지 매니저입니다",
	Long:          `GitHub/GitLab을 위한 빠르고 데이터 중심적인 프라이빗 패키지 매니저입니다.`,
	SilenceErrors: true,
	PersistentPreRun: func(ctx *wcli.Context) error {
		logger.DebugMode = debugMode
		logger.Quiet = quietMode
		configureOutput()
		return nil
	},
}

var (
	debugMode   bool
	quietMode   bool
	noColorMode bool
	noInputMode bool
	versionMode bool
)

// Execute는 프로세스의 커맨드라인 인자로 루트 명령을 실행합니다.
func Execute() error {
	return ExecuteArgs(os.Args[1:])
}

// ExecuteArgs는 전달받은 인자로 루트 명령을 실행합니다.
// 프로세스 전역 os.Args에 의존하지 않아 CLI 실행을 독립적으로 테스트할 수 있습니다.
func ExecuteArgs(args []string) error {
	rootCmd.ResetFlags()
	if requestedRootVersion(args) {
		configureOutput()
		fmt.Printf("ppm version %s %s/%s\n", resolveVersion(), runtime.GOOS, runtime.GOARCH)
		return nil
	}
	args = consumeLeadingRootFlags(args)
	configureOutput()
	if err := validateCommandPath(args); err != nil {
		return fmt.Errorf("execute ppm command: %w", err)
	}

	if err := rootCmd.Execute(args); err != nil {
		if errors.Is(err, wcli.ErrHelp) {
			return nil
		}
		return fmt.Errorf("execute ppm command: %w", err)
	}
	return nil
}

func consumeLeadingRootFlags(args []string) []string {
	index := 0
	for index < len(args) {
		switch args[index] {
		case "--debug", "-d":
			debugMode = true
		case "--quiet", "-q":
			quietMode = true
		case "--no-color":
			noColorMode = true
		case "--no-input":
			noInputMode = true
		default:
			if args[index] == "--" || !strings.HasPrefix(args[index], "-") {
				return args[index:]
			}
			return args
		}
		index++
	}
	return args[index:]
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", "d", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&quietMode, "quiet", "q", false, "상태 출력 억제 (오류·경고는 유지)")
	rootCmd.PersistentFlags().BoolVar(&noColorMode, "no-color", "", false, "ANSI 색상과 애니메이션 비활성화")
	rootCmd.PersistentFlags().BoolVar(&noInputMode, "no-input", "", false, "대화형 입력 금지")
	rootCmd.PersistentFlags().BoolVar(&versionMode, "version", "v", false, "ppm 버전 출력")
	// 루트 전역 플래그는 여기서 설정합니다.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "설정 파일 경로 (기본값: $HOME/.config/ppm/config.yaml)")
}

func configureOutput() {
	logger.Quiet = quietMode
	ui.ColorEnabled = !noColorMode && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && isTerminal(os.Stdout)
	ui.AnimationEnabled = ui.ColorEnabled && isTerminal(os.Stderr)
}

func isTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func requestedRootVersion(args []string) bool {
	for _, arg := range args {
		if arg == "--version" || arg == "-v" || arg == "--version=true" {
			return true
		}
		if arg == "--" {
			break
		}
	}
	return false
}

// wcli v0.2.0은 알 수 없는 최상위/그룹 명령을 도움말로 처리해 exit 0을 반환합니다.
// 실행 전에 명령 경로를 검증해 오타가 성공으로 오인되지 않도록 합니다.
func validateCommandPath(args []string) error {
	tokens := commandTokens(args)
	if len(tokens) == 0 {
		return nil
	}
	topLevel := map[string]bool{
		"apps": true, "auth": true, "cache": true, "changelog": true, "clean": true, "completion": true,
		"config": true, "doctor": true, "info": true, "init": true, "install": true,
		"list": true, "lock": true, "manifest": true, "outdated": true, "package": true,
		"search": true, "self-update": true, "uninstall": true, "update": true,
		"verify": true, "version": true,
	}
	if !topLevel[tokens[0]] {
		return fmt.Errorf("unknown command %q", tokens[0])
	}
	subcommands := map[string]map[string]bool{
		"auth":     {"login": true, "status": true, "logout": true},
		"cache":    {"list": true, "clean": true},
		"config":   {"show": true, "set": true},
		"manifest": {"validate": true},
		"package":  {"init": true, "validate": true},
	}
	if allowed, ok := subcommands[tokens[0]]; ok && len(tokens) > 1 && !allowed[tokens[1]] {
		return fmt.Errorf("unknown command %q for %s", tokens[1], tokens[0])
	}
	return nil
}

func commandTokens(args []string) []string {
	var tokens []string
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		tokens = append(tokens, arg)
	}
	return tokens
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
