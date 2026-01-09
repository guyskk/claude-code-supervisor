// Package errors provides unified error handling for ccc.
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType represents the category of an error.
type ErrorType int

const (
	// ErrTypeConfig indicates a configuration-related error.
	ErrTypeConfig ErrorType = iota
	// ErrTypeNetwork indicates a network-related error.
	ErrTypeNetwork
	// ErrTypeProcess indicates a process-related error.
	ErrTypeProcess
	// ErrTypeValidation indicates a validation error.
	ErrTypeValidation
	// ErrTypeTimeout indicates a timeout error.
	ErrTypeTimeout
)

// String returns the string representation of the ErrorType.
func (t ErrorType) String() string {
	switch t {
	case ErrTypeConfig:
		return "CONFIG"
	case ErrTypeNetwork:
		return "NETWORK"
	case ErrTypeProcess:
		return "PROCESS"
	case ErrTypeValidation:
		return "VALIDATION"
	case ErrTypeTimeout:
		return "TIMEOUT"
	default:
		return "UNKNOWN"
	}
}

// Predefined error codes.
const (
	// Config errors
	CCCConfigInvalid     = "CCC_CONFIG_INVALID"
	CCCConfigNotFound    = "CCC_CONFIG_NOT_FOUND"
	CCCConfigReadFailed  = "CCC_CONFIG_READ_FAILED"
	CCCConfigParseFailed = "CCC_CONFIG_PARSE_FAILED"

	// Provider errors
	CCCProviderNotFound = "CCC_PROVIDER_NOT_FOUND"
	CCCProviderInvalid  = "CCC_PROVIDER_INVALID"

	// Claude errors
	CCCCLaudeNotFound       = "CCC_CLAUDE_NOT_FOUND"
	CCCCLaudeStartFailed    = "CCC_CLAUDE_START_FAILED"
	CCCCLaudeExitAbnormally = "CCC_CLAUDE_EXIT_ABNORMALLY"

	// Supervisor errors
	CCCSupervisorTimeout       = "CCC_SUPERVISOR_TIMEOUT"
	CCCSupervisorMaxIterations = "CCC_SUPERVISOR_MAX_ITERATIONS"
	CCCSupervisorParseFailed   = "CCC_SUPERVISOR_PARSE_FAILED"
)

// AppError represents an application error with structured information.
type AppError struct {
	Type    ErrorType
	Code    string
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error returns the formatted error message.
func (e *AppError) Error() string {
	var parts []string

	// Error code in brackets
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}

	// Context key-value pairs (without extra brackets)
	if len(e.Context) > 0 {
		ctxParts := make([]string, 0, len(e.Context))
		for k, v := range e.Context {
			ctxParts = append(ctxParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, strings.Join(ctxParts, " "))
	}

	// Main message
	parts = append(parts, e.Message)

	return strings.Join(parts, " ")
}

// Unwrap returns the underlying cause error.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is checks if the target error is the same type as this error.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithContext returns a copy of the error with additional context.
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	newErr := &AppError{
		Type:    e.Type,
		Code:    e.Code,
		Message: e.Message,
		Cause:   e.Cause,
		Context: make(map[string]interface{}),
	}

	// Copy existing context
	for k, v := range e.Context {
		newErr.Context[k] = v
	}

	// Add new context
	newErr.Context[key] = value

	return newErr
}

// WithContextMap returns a copy of the error with additional context from a map.
func (e *AppError) WithContextMap(ctx map[string]interface{}) *AppError {
	newErr := &AppError{
		Type:    e.Type,
		Code:    e.Code,
		Message: e.Message,
		Cause:   e.Cause,
		Context: make(map[string]interface{}),
	}

	// Copy existing context
	for k, v := range e.Context {
		newErr.Context[k] = v
	}

	// Add new context
	for k, v := range ctx {
		newErr.Context[k] = v
	}

	return newErr
}

// NewError creates a new AppError.
func NewError(typ ErrorType, code, message string, cause error) *AppError {
	return &AppError{
		Type:    typ,
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, code, message string) *AppError {
	if err == nil {
		return nil
	}

	// If already an AppError, just update the message
	var appErr *AppError
	if errors.As(err, &appErr) {
		return &AppError{
			Type:    appErr.Type,
			Code:    code,
			Message: message,
			Cause:   appErr,
			Context: make(map[string]interface{}),
		}
	}

	// Create a new AppError
	return &AppError{
		Type:    ErrTypeProcess, // Default type for wrapped errors
		Code:    code,
		Message: message,
		Cause:   err,
		Context: make(map[string]interface{}),
	}
}

// IsAppError checks if an error is an AppError.
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// AsAppError converts an error to AppError if possible.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// GetCode extracts the error code from an error.
// Returns empty string if the error is not an AppError.
func GetCode(err error) string {
	if appErr, ok := AsAppError(err); ok {
		return appErr.Code
	}
	return ""
}

// GetType extracts the error type from an error.
// Returns ErrTypeProcess as default if the error is not an AppError.
func GetType(err error) ErrorType {
	if appErr, ok := AsAppError(err); ok {
		return appErr.Type
	}
	return ErrTypeProcess
}
