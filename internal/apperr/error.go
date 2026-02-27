package apperr

import (
	"fmt"
)

// ErrorCode represents a specific type of application error
type ErrorCode int

const (
	// Enum for error codes
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

// AppError is a data-centric error structure for easy handling and logging
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

// New creates a new AppError without an internal wrapped error
func New(code ErrorCode, format string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap creates a new AppError wrapping an existing error
func Wrap(code ErrorCode, err error, format string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}
