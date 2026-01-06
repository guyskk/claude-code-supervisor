// Package errors provides unified error handling for the ccc application.
// It defines error codes, error types, and helper functions for error wrapping
// and checking following Go best practices.
package errors

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// ErrorCode represents a specific error category.
type ErrorCode string

const (
	// ErrCodeConfig indicates a configuration-related error.
	ErrCodeConfig ErrorCode = "CONFIG_ERROR"
	// ErrCodeIO indicates an I/O error.
	ErrCodeIO ErrorCode = "IO_ERROR"
	// ErrCodeValidation indicates a validation error.
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"
	// ErrCodeNotFound indicates a resource was not found.
	ErrCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrCodePermission indicates a permission error.
	ErrCodePermission ErrorCode = "PERMISSION_ERROR"
	// ErrCodeExec indicates an execution error.
	ErrCodeExec ErrorCode = "EXEC_ERROR"
	// ErrCodeTimeout indicates a timeout error.
	ErrCodeTimeout ErrorCode = "TIMEOUT"
	// ErrCodeCancelled indicates a cancelled operation.
	ErrCodeCancelled ErrorCode = "CANCELLED"
	// ErrCodeInternal indicates an internal error.
	ErrCodeInternal ErrorCode = "INTERNAL_ERROR"
)

// Error is the standard error type for the ccc application.
// It wraps the underlying error with additional context.
type Error struct {
	// Code is the error category code.
	Code ErrorCode
	// Message is a human-readable description of what went wrong.
	Message string
	// Cause is the underlying error that caused this error.
	Cause error
	// Context contains additional key-value pairs for debugging.
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *Error) Error() string {
	base := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.Cause != nil {
		base += fmt.Sprintf(": %v", e.Cause)
	}
	return base
}

// Unwrap returns the underlying cause error for use with errors.Is/As.
func (e *Error) Unwrap() error {
	return e.Cause
}

// New creates a new Error with the given code and message.
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap creates a new Error wrapping an underlying cause.
func Wrap(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrapf creates a new Error wrapping an underlying cause with formatted message.
func Wrapf(code ErrorCode, format string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, cause),
		Cause:   cause,
	}
}

// WithContext adds key-value context to an error.
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// GetContext retrieves a context value by key.
func (e *Error) GetContext(key string) (interface{}, bool) {
	if e.Context == nil {
		return nil, false
	}
	val, ok := e.Context[key]
	return val, ok
}

// Is checks if an error matches a specific error code.
func Is(err error, code ErrorCode) bool {
	var e *Error
	if err == nil {
		return false
	}
	if errors.As(err, &e) {
		return e.Code == code
	}
	return false
}

// GetCode extracts the error code from an error.
// Returns empty string if the error is not an *Error.
func GetCode(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// IsNotExist returns true if the error indicates something does not exist.
// This checks for both *Error with NotFound code and os.ErrNotExist.
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	// Check if it's our wrapped error
	var e *Error
	if errors.As(err, &e) {
		if e.Code == ErrCodeNotFound {
			return true
		}
		if e.Cause != nil && os.IsNotExist(e.Cause) {
			return true
		}
	}
	// Check directly for os.ErrNotExist
	return os.IsNotExist(err)
}

// IsPermission returns true if the error indicates a permission problem.
// This checks for both *Error with Permission code and os.ErrPermission.
func IsPermission(err error) bool {
	if err == nil {
		return false
	}
	// Check if it's our wrapped error
	var e *Error
	if errors.As(err, &e) {
		if e.Code == ErrCodePermission {
			return true
		}
		if e.Cause != nil && os.IsPermission(e.Cause) {
			return true
		}
	}
	// Check directly for os.ErrPermission
	return os.IsPermission(err)
}

// IsTimeout returns true if the error indicates a timeout.
// This checks for both *Error with Timeout code and context.DeadlineExceeded.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return Is(err, ErrCodeTimeout)
}

// IsConnectionRefused returns true if the error is a connection refused error.
// This checks for syscall.ECONNREFUSED.
func IsConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.ECONNREFUSED)
}

// Helper functions for creating common errors

// ConfigError creates a configuration error.
func ConfigError(message string) *Error {
	return New(ErrCodeConfig, message)
}

// ConfigErrorf creates a configuration error with formatted message.
func ConfigErrorf(format string, args ...interface{}) *Error {
	return &Error{
		Code:    ErrCodeConfig,
		Message: fmt.Sprintf(format, args...),
	}
}

// WrapConfigError wraps an error as a configuration error.
func WrapConfigError(message string, cause error) *Error {
	return Wrap(ErrCodeConfig, message, cause)
}

// IOError creates an I/O error.
func IOError(message string) *Error {
	return New(ErrCodeIO, message)
}

// WrapIOError wraps an error as an I/O error.
func WrapIOError(message string, cause error) *Error {
	return Wrap(ErrCodeIO, message, cause)
}

// ValidationError creates a validation error.
func ValidationError(message string) *Error {
	return New(ErrCodeValidation, message)
}

// ValidationErrorf creates a validation error with formatted message.
func ValidationErrorf(format string, args ...interface{}) *Error {
	return &Error{
		Code:    ErrCodeValidation,
		Message: fmt.Sprintf(format, args...),
	}
}

// NotFoundError creates a not found error.
func NotFoundError(message string) *Error {
	return New(ErrCodeNotFound, message)
}

// PermissionError creates a permission error.
func PermissionError(message string) *Error {
	return New(ErrCodePermission, message)
}

// ExecError creates an execution error.
func ExecError(message string) *Error {
	return New(ErrCodeExec, message)
}

// WrapExecError wraps an error as an execution error.
func WrapExecError(message string, cause error) *Error {
	return Wrap(ErrCodeExec, message, cause)
}

// TimeoutError creates a timeout error.
func TimeoutError(message string) *Error {
	return New(ErrCodeTimeout, message)
}

// CancelledError creates a cancelled error.
func CancelledError(message string) *Error {
	return New(ErrCodeCancelled, message)
}

// InternalError creates an internal error.
func InternalError(message string) *Error {
	return New(ErrCodeInternal, message)
}

// WrapInternalError wraps an error as an internal error.
func WrapInternalError(message string, cause error) *Error {
	return Wrap(ErrCodeInternal, message, cause)
}
