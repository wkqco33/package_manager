package ui

import "fmt"

// ColorEnabled는 ANSI 스타일 출력 사용 여부를 제어합니다.
// 기본값은 호환성을 위해 활성화되어 있으며, CLI 초기화 시 터미널 상태에 맞춰 갱신됩니다.
var ColorEnabled = true

// ANSI 스타일 상수 (현대적인 CLI 디자인을 위한 조합)
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Underline = "\033[4m"

	// 기본 색상
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// 밝은 색상 (더 눈에 띔)
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
)

// 정보 메시지 스타일
func Info(msg string) string {
	if !ColorEnabled {
		return fmt.Sprintf("ℹ %s", msg)
	}
	return fmt.Sprintf("%s%s %s%s", BrightBlue, "ℹ", msg, Reset)
}

// 성공 메시지 스타일
func Success(msg string) string {
	if !ColorEnabled {
		return fmt.Sprintf("✔ %s", msg)
	}
	return fmt.Sprintf("%s%s%s %s%s", Bold, BrightGreen, "✔", msg, Reset)
}

// 경고 메시지 스타일
func Warning(msg string) string {
	if !ColorEnabled {
		return fmt.Sprintf("⚠ %s", msg)
	}
	return fmt.Sprintf("%s%s%s %s%s", Bold, BrightYellow, "⚠", msg, Reset)
}

// 오류 메시지 스타일
func Error(msg string) string {
	if !ColorEnabled {
		return fmt.Sprintf("✖ %s", msg)
	}
	return fmt.Sprintf("%s%s%s %s%s", Bold, BrightRed, "✖", msg, Reset)
}

// 강조 텍스트 스타일
func Highlight(msg string) string {
	if !ColorEnabled {
		return msg
	}
	return fmt.Sprintf("%s%s%s", BrightCyan, msg, Reset)
}

// 라벨 텍스트 스타일
func Label(msg string) string {
	if !ColorEnabled {
		return msg
	}
	return fmt.Sprintf("%s%s%s", Bold, msg, Reset)
}

// 보조 텍스트 스타일
func Muted(msg string) string {
	if !ColorEnabled {
		return msg
	}
	return fmt.Sprintf("%s%s%s", Gray, msg, Reset)
}

// 경로 텍스트 스타일
func Path(msg string) string {
	if !ColorEnabled {
		return msg
	}
	return fmt.Sprintf("%s%s%s%s", Underline, Cyan, msg, Reset)
}

// 액센트 텍스트 스타일
func Accent(msg string) string {
	if !ColorEnabled {
		return msg
	}
	return fmt.Sprintf("%s%s%s", BrightMagenta, msg, Reset)
}
