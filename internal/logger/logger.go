package logger

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/wkqco33/package_manager/internal/ui"
)

var (
	// DebugMode는 디버그 로그 출력 여부를 제어합니다.
	DebugMode bool

	// slogLogger는 log/slog 기반 내부 구조화 로거입니다.
	slogLogger *slog.Logger
)

func init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	// 콘솔 가독성을 위해 표준 텍스트 핸들러 사용
	slogLogger = slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// Info는 일반 사용자 메시지를 출력합니다.
func Info(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(ui.Info(msg))
}

// Success는 성공 메시지를 출력합니다.
func Success(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(ui.Success(msg))
}

// Error는 오류 메시지를 출력합니다.
func Error(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, ui.Error(msg))
}

// Warn은 경고 메시지를 출력합니다.
func Warn(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, ui.Warning(msg))
}

// Debug는 DebugMode가 true일 때 slog로 디버그 로그를 남깁니다.
func Debug(msg string, args ...any) {
	if DebugMode {
		slogLogger.Debug(msg, args...)
	}
}
