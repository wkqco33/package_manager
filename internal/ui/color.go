package ui

import "fmt"

// ANSI 색상 코드 (데이터 중심 설계, 상수 사용)
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
	Bold   = "\033[1m"
)

// Info 메시지 포맷팅 (파란색)
func Info(msg string) string {
	return fmt.Sprintf("%s%s%s", Blue, msg, Reset)
}

// Success 메시지 포맷팅 (초록색)
func Success(msg string) string {
	return fmt.Sprintf("%s%s%s", Green, msg, Reset)
}

// Warning 메시지 포맷팅 (노란색)
func Warning(msg string) string {
	return fmt.Sprintf("%s%s%s", Yellow, msg, Reset)
}

// Error 메시지 포맷팅 (빨간색 굵게)
func Error(msg string) string {
	return fmt.Sprintf("%s%s%s%s", Bold, Red, msg, Reset)
}

// Highlight 특정 부분 강조 (청록색)
func Highlight(msg string) string {
	return fmt.Sprintf("%s%s%s", Cyan, msg, Reset)
}
