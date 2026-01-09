// Package errors provides tests for error handling.
package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		name string
		t    ErrorType
		want string
	}{
		{"config", ErrTypeConfig, "CONFIG"},
		{"network", ErrTypeNetwork, "NETWORK"},
		{"process", ErrTypeProcess, "PROCESS"},
		{"validation", ErrTypeValidation, "VALIDATION"},
		{"timeout", ErrTypeTimeout, "TIMEOUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.String(); got != tt.want {
				t.Errorf("ErrorType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		want string
	}{
		{
			name: "basic error",
			err:  NewError(ErrTypeConfig, CCCConfigNotFound, "配置文件不存在", nil),
			want: "[CCC_CONFIG_NOT_FOUND] 配置文件不存在",
		},
		{
			name: "error with context",
			err:  NewError(ErrTypeConfig, CCCConfigNotFound, "配置文件不存在", nil).WithContext("path", "/path/to/config"),
			want: "[CCC_CONFIG_NOT_FOUND] path=/path/to/config 配置文件不存在",
		},
		{
			name: "error with cause",
			err:  NewError(ErrTypeConfig, CCCConfigReadFailed, "读取配置失败", errors.New("file not found")),
			want: "[CCC_CONFIG_READ_FAILED] 读取配置失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !strings.Contains(got, tt.want) {
				t.Errorf("AppError.Error() = %v, want to contain %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := NewError(ErrTypeConfig, CCCConfigParseFailed, "解析失败", cause)

	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestAppError_With(t *testing.T) {
	baseErr := NewError(ErrTypeConfig, CCCConfigNotFound, "配置文件不存在", nil)
	err := baseErr.WithContext("path", "/test/path").WithContext("user", "testuser")

	// Check context values
	if err.Context["path"] != "/test/path" {
		t.Errorf("Context[path] = %v, want /test/path", err.Context["path"])
	}
	if err.Context["user"] != "testuser" {
		t.Errorf("Context[user] = %v, want testuser", err.Context["user"])
	}

	// Original error should be unchanged
	if baseErr.Context["path"] != nil {
		t.Error("Original error should not have context")
	}
}

func TestAppError_WithMap(t *testing.T) {
	baseErr := NewError(ErrTypeConfig, CCCConfigNotFound, "配置文件不存在", nil)
	err := baseErr.WithContextMap(map[string]interface{}{
		"path": "/test/path",
		"user": "testuser",
	})

	// Check context values
	if err.Context["path"] != "/test/path" {
		t.Errorf("Context[path] = %v, want /test/path", err.Context["path"])
	}
	if err.Context["user"] != "testuser" {
		t.Errorf("Context[user] = %v, want testuser", err.Context["user"])
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrTypeConfig, CCCConfigInvalid, "配置无效", nil)

	if err.Type != ErrTypeConfig {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeConfig)
	}
	if err.Code != CCCConfigInvalid {
		t.Errorf("Code = %v, want %v", err.Code, CCCConfigInvalid)
	}
	if err.Message != "配置无效" {
		t.Errorf("Message = %v, want %v", err.Message, "配置无效")
	}
	if err.Cause != nil {
		t.Errorf("Cause = %v, want nil", err.Cause)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	err := Wrap(cause, CCCCLaudeNotFound, "claude 未找到")

	if err.Code != CCCCLaudeNotFound {
		t.Errorf("Code = %v, want %v", err.Code, CCCCLaudeNotFound)
	}
	if err.Message != "claude 未找到" {
		t.Errorf("Message = %v, want %v", err.Message, "claude 未找到")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}

	// Test nil error
	if Wrap(nil, "CODE", "message") != nil {
		t.Error("Wrap(nil) should return nil")
	}
}

func TestWrap_AppError(t *testing.T) {
	original := NewError(ErrTypeConfig, CCCConfigNotFound, "配置不存在", nil)
	wrapped := Wrap(original, "NEW_CODE", "新的消息")

	// Wrapped error should preserve original as cause
	if wrapped.Cause != original {
		t.Errorf("Cause = %v, want %v", wrapped.Cause, original)
	}

	// Verify error chain
	if !errors.Is(wrapped, original) {
		t.Error("wrapped should be is original")
	}
}

func TestIsAppError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"AppError", NewError(ErrTypeConfig, CCCConfigNotFound, "配置不存在", nil), true},
		{"standard error", errors.New("standard"), false},
		{"nil", nil, false},
		{"wrapped AppError", Wrap(NewError(ErrTypeConfig, CCCConfigNotFound, "配置不存在", nil), "CODE", "msg"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAppError(tt.err); got != tt.want {
				t.Errorf("IsAppError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsAppError(t *testing.T) {
	appErr := NewError(ErrTypeConfig, CCCConfigNotFound, "配置不存在", nil)

	// Should return the AppError
	if got, ok := AsAppError(appErr); !ok || got != appErr {
		t.Error("AsAppError(AppError) should return the same error")
	}

	// Standard error should not be convertible
	stdErr := errors.New("standard")
	if _, ok := AsAppError(stdErr); ok {
		t.Error("AsAppError(standard error) should return false")
	}

	// Wrapped error should be unwrapped
	wrapped := Wrap(appErr, "NEW_CODE", "消息")
	if got, ok := AsAppError(wrapped); !ok {
		t.Error("AsAppError(wrapped AppError) should return true")
	} else if got.Code != "NEW_CODE" {
		t.Errorf("Code = %v, want NEW_CODE", got.Code)
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"AppError", NewError(ErrTypeConfig, CCCConfigNotFound, "配置不存在", nil), CCCConfigNotFound},
		{"standard error", errors.New("standard"), ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCode(tt.err); got != tt.want {
				t.Errorf("GetCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorType
	}{
		{"AppError", NewError(ErrTypeConfig, CCCConfigNotFound, "配置不存在", nil), ErrTypeConfig},
		{"AppError network", NewError(ErrTypeNetwork, "CODE", "msg", nil), ErrTypeNetwork},
		{"standard error", errors.New("standard"), ErrTypeProcess},
		{"nil", nil, ErrTypeProcess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetType(tt.err); got != tt.want {
				t.Errorf("GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppError_Is(t *testing.T) {
	err1 := NewError(ErrTypeConfig, CCCConfigNotFound, "配置不存在", nil)
	err2 := NewError(ErrTypeConfig, CCCConfigNotFound, "另一个消息", nil)
	err3 := NewError(ErrTypeConfig, CCCConfigInvalid, "配置无效", nil)

	// Same code should be equal
	if !err1.Is(err2) {
		t.Error("Errors with same code should be equal")
	}

	// Different code should not be equal
	if err1.Is(err3) {
		t.Error("Errors with different codes should not be equal")
	}

	// Standard error should not be equal
	if err1.Is(errors.New("test")) {
		t.Error("AppError should not equal standard error")
	}
}

func ExampleAppError() {
	// Create a new error
	err := NewError(ErrTypeConfig, CCCConfigNotFound, "配置文件不存在", nil).
		WithContext("path", "/path/to/ccc.json")

	fmt.Println(err)
	// Output: [CCC_CONFIG_NOT_FOUND] path=/path/to/ccc.json 配置文件不存在
}

func ExampleWrap() {
	cause := errors.New("permission denied")
	err := Wrap(cause, "CCC_CONFIG_READ_FAILED", "读取配置失败").
		WithContext("path", "/path/to/file")

	fmt.Println(err)
	fmt.Println(errors.Unwrap(err))
	// Unordered output:
	// [CCC_CONFIG_READ_FAILED] path=/path/to/file 读取配置失败
	// permission denied
}
