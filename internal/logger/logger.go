package logger

import (
	"fmt"
	"log/slog"
	"os"

	"ppm/internal/ui"
)

var (
	// DebugMode controls whether debug logs are printed
	DebugMode bool

	// slogLogger is the underlying structured logger using log/slog
	slogLogger *slog.Logger
)

func init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	// Use standard text handler for clear console output
	slogLogger = slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// Info prints a standard user-facing message
func Info(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(ui.Info(msg))
}

// Success prints a success message
func Success(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(ui.Success("✅ " + msg))
}

// Error prints an error message
func Error(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, ui.Error("❌ Error: "+msg))
}

// Warn prints a warning message
func Warn(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, ui.Warning("⚠️ Warn: "+msg))
}

// Debug logs a debug message using slog if DebugMode is true
func Debug(msg string, args ...any) {
	if DebugMode {
		slogLogger.Debug(msg, args...)
	}
}
