package apperr

import (
	"fmt"
)

// ErrorCode는 애플리케이션 오류 유형을 나타냅니다.
type ErrorCode int

const (
	// 오류 코드 열거형
	CodeUnknown ErrorCode = iota
	CodeConfig
	CodeRegistry
	CodeNetwork
	CodeArchive
	CodeFileSystem
	CodeInvalidInput
)

func (c ErrorCode) String() string {
	switch c {
	case CodeConfig:
		return "CONFIG_ERROR"
	case CodeRegistry:
		return "REGISTRY_ERROR"
	case CodeNetwork:
		return "NETWORK_ERROR"
	case CodeArchive:
		return "ARCHIVE_ERROR"
	case CodeFileSystem:
		return "FS_ERROR"
	case CodeInvalidInput:
		return "INVALID_INPUT"
	default:
		return "UNKNOWN_ERROR"
	}
}

// AppError는 처리와 로깅을 위한 구조화 오류입니다.
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code.String(), e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New는 내부 래핑 오류 없이 AppError를 생성합니다.
func New(code ErrorCode, format string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap은 기존 오류를 감싸 AppError를 생성합니다.
func Wrap(code ErrorCode, err error, format string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}
