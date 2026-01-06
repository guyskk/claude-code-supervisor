// Package errors tests for the unified error handling.
package errors

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrCodeConfig, "test message")
	if err.Code != ErrCodeConfig {
		t.Errorf("expected code %s, got %s", ErrCodeConfig, err.Code)
	}
	if err.Message != "test message" {
		t.Errorf("expected message 'test message', got '%s'", err.Message)
	}
	if err.Cause != nil {
		t.Error("expected nil cause")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(ErrCodeIO, "wrapper message", cause)

	if err.Code != ErrCodeIO {
		t.Errorf("expected code %s, got %s", ErrCodeIO, err.Code)
	}
	if err.Message != "wrapper message" {
		t.Errorf("expected message 'wrapper message', got '%s'", err.Message)
	}
	if err.Cause != cause {
		t.Error("expected cause to be set")
	}

	expectedError := "IO_ERROR: wrapper message: underlying error"
	if err.Error() != expectedError {
		t.Errorf("expected '%s', got '%s'", expectedError, err.Error())
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("cause")
	err := Wrap(ErrCodeValidation, "validation failed", cause)

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestErrorWithStandardErrorsIs(t *testing.T) {
	cause := errors.New("test")
	err := Wrap(ErrCodeIO, "message", cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should work with wrapped error")
	}
}

func TestErrorWithStandardErrorsAs(t *testing.T) {
	cause := &os.PathError{Path: "/test"}
	err := Wrap(ErrCodeIO, "message", cause)

	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Error("errors.As should work with wrapped error")
	}
}

func TestIs(t *testing.T) {
	err := New(ErrCodeConfig, "config error")

	if !Is(err, ErrCodeConfig) {
		t.Error("Is should return true for matching code")
	}
	if Is(err, ErrCodeIO) {
		t.Error("Is should return false for non-matching code")
	}
	if Is(nil, ErrCodeConfig) {
		t.Error("Is should return false for nil error")
	}
}

func TestGetCode(t *testing.T) {
	err := New(ErrCodeValidation, "validation error")
	if GetCode(err) != ErrCodeValidation {
		t.Error("GetCode should return the error code")
	}

	stdErr := errors.New("standard error")
	if GetCode(stdErr) != "" {
		t.Error("GetCode should return empty string for non-Error type")
	}
}

func TestIsNotExist(t *testing.T) {
	// Test with os.ErrNotExist
	err := WrapIOError("file not found", os.ErrNotExist)
	if !IsNotExist(err) {
		t.Error("IsNotExist should return true for os.ErrNotExist")
	}

	// Test with NotFound code
	err2 := NotFoundError("resource not found")
	if !IsNotExist(err2) {
		t.Error("IsNotExist should return true for NotFound code")
	}

	// Test with other error
	err3 := New(ErrCodeConfig, "config error")
	if IsNotExist(err3) {
		t.Error("IsNotExist should return false for other errors")
	}
}

func TestIsPermission(t *testing.T) {
	// Test with os.ErrPermission
	err := WrapIOError("access denied", os.ErrPermission)
	if !IsPermission(err) {
		t.Error("IsPermission should return true for os.ErrPermission")
	}

	// Test with Permission code
	err2 := PermissionError("access denied")
	if !IsPermission(err2) {
		t.Error("IsPermission should return true for Permission code")
	}

	// Test with other error
	err3 := New(ErrCodeConfig, "config error")
	if IsPermission(err3) {
		t.Error("IsPermission should return false for other errors")
	}
}

func TestIsTimeout(t *testing.T) {
	err := TimeoutError("operation timed out")
	if !IsTimeout(err) {
		t.Error("IsTimeout should return true for Timeout code")
	}

	err2 := New(ErrCodeConfig, "config error")
	if IsTimeout(err2) {
		t.Error("IsTimeout should return false for other errors")
	}
}

func TestIsConnectionRefused(t *testing.T) {
	err := WrapExecError("connection failed", syscall.ECONNREFUSED)
	if !IsConnectionRefused(err) {
		t.Error("IsConnectionRefused should return true for ECONNREFUSED")
	}

	err2 := New(ErrCodeConfig, "config error")
	if IsConnectionRefused(err2) {
		t.Error("IsConnectionRefused should return false for other errors")
	}
}

func TestContext(t *testing.T) {
	err := New(ErrCodeConfig, "config error").
		WithContext("file", "config.json").
		WithContext("line", 42)

	if val, ok := err.GetContext("file"); !ok || val != "config.json" {
		t.Error("WithContext should add context values")
	}
	if val, ok := err.GetContext("line"); !ok || val != 42 {
		t.Error("WithContext should add context values")
	}
	if _, ok := err.GetContext("missing"); ok {
		t.Error("GetContext should return false for missing keys")
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() *Error
		wantCode ErrorCode
		wantMsg  string
	}{
		{"ConfigError", func() *Error { return ConfigError("test") }, ErrCodeConfig, "test"},
		{"ConfigErrorf", func() *Error { return ConfigErrorf("test %d", 42) }, ErrCodeConfig, "test 42"},
		{"WrapConfigError", func() *Error { return WrapConfigError("test", errors.New("cause")) }, ErrCodeConfig, "test"},
		{"IOError", func() *Error { return IOError("test") }, ErrCodeIO, "test"},
		{"WrapIOError", func() *Error { return WrapIOError("test", errors.New("cause")) }, ErrCodeIO, "test"},
		{"ValidationError", func() *Error { return ValidationError("test") }, ErrCodeValidation, "test"},
		{"ValidationErrorf", func() *Error { return ValidationErrorf("test %d", 42) }, ErrCodeValidation, "test 42"},
		{"NotFoundError", func() *Error { return NotFoundError("test") }, ErrCodeNotFound, "test"},
		{"PermissionError", func() *Error { return PermissionError("test") }, ErrCodePermission, "test"},
		{"ExecError", func() *Error { return ExecError("test") }, ErrCodeExec, "test"},
		{"WrapExecError", func() *Error { return WrapExecError("test", errors.New("cause")) }, ErrCodeExec, "test"},
		{"TimeoutError", func() *Error { return TimeoutError("test") }, ErrCodeTimeout, "test"},
		{"CancelledError", func() *Error { return CancelledError("test") }, ErrCodeCancelled, "test"},
		{"InternalError", func() *Error { return InternalError("test") }, ErrCodeInternal, "test"},
		{"WrapInternalError", func() *Error { return WrapInternalError("test", errors.New("cause")) }, ErrCodeInternal, "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err.Code != tt.wantCode {
				t.Errorf("expected code %s, got %s", tt.wantCode, err.Code)
			}
			if err.Message != tt.wantMsg {
				t.Errorf("expected message %q, got %q", tt.wantMsg, err.Message)
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	cause := errors.New("cause")
	err := Wrapf(ErrCodeIO, "formatted: %v", cause)

	if err.Code != ErrCodeIO {
		t.Errorf("expected code %s, got %s", ErrCodeIO, err.Code)
	}
	if err.Cause != cause {
		t.Error("expected cause to be set")
	}
}
