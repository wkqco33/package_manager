package ui

import "fmt"

// ANSI 스타일 상수 (현대적인 CLI 디자인을 위한 조합)
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	Underline  = "\033[4m"

	// 기본 색상
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	Gray       = "\033[90m"

	// 밝은 색상 (더 눈에 띔)
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
)

// Info: 정보 안내 (Bright Blue)
func Info(msg string) string {
	return fmt.Sprintf("%s%s %s%s", BrightBlue, "ℹ", msg, Reset)
}

// Success: 작업 성공 (Bold Green + Check)
func Success(msg string) string {
	return fmt.Sprintf("%s%s%s %s%s", Bold, BrightGreen, "✔", msg, Reset)
}

// Warning: 주의 사항 (Bold Yellow + Alert)
func Warning(msg string) string {
	return fmt.Sprintf("%s%s%s %s%s", Bold, BrightYellow, "⚠", msg, Reset)
}

// Error: 오류 발생 (Bold Red + Cross)
func Error(msg string) string {
	return fmt.Sprintf("%s%s%s %s%s", Bold, BrightRed, "✖", msg, Reset)
}

// Highlight: 중요 키워드 강조 (Bright Cyan)
func Highlight(msg string) string {
	return fmt.Sprintf("%s%s%s", BrightCyan, msg, Reset)
}

// Label: 항목 이름 (Bold White)
func Label(msg string) string {
	return fmt.Sprintf("%s%s%s", Bold, msg, Reset)
}

// Muted: 부가 설명이나 덜 중요한 정보 (Gray/Dim)
func Muted(msg string) string {
	return fmt.Sprintf("%s%s%s", Gray, msg, Reset)
}

// Path: 파일 경로 (Underline Cyan)
func Path(msg string) string {
	return fmt.Sprintf("%s%s%s%s", Underline, Cyan, msg, Reset)
}

// Accent: 특별한 강조 (Bright Magenta)
func Accent(msg string) string {
	return fmt.Sprintf("%s%s%s", BrightMagenta, msg, Reset)
}
