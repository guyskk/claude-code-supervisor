// Package logger provides tests for the logger.
package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Level
		wantErr bool
	}{
		{"debug lower", "debug", LevelDebug, false},
		{"debug upper", "DEBUG", LevelDebug, false},
		{"info lower", "info", LevelInfo, false},
		{"info upper", "INFO", LevelInfo, false},
		{"warn lower", "warn", LevelWarn, false},
		{"warn upper", "WARN", LevelWarn, false},
		{"warning lower", "warning", LevelWarn, false},
		{"error lower", "error", LevelError, false},
		{"error upper", "ERROR", LevelError, false},
		{"unknown", "unknown", LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		name string
		l    Level
		want string
	}{
		{"debug", LevelDebug, "DEBUG"},
		{"info", LevelInfo, "INFO"},
		{"warn", LevelWarn, "WARN"},
		{"error", LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	logger := NewLogger(&buf, LevelDebug)

	// Test logging at different levels
	logger.Debug("debug message", StringField("key1", "value1"))
	logger.Info("info message", StringField("key2", "value2"))
	logger.Warn("warn message", StringField("key3", "value3"))
	logger.Error("error message", StringField("key4", "value4"))

	output := buf.String()

	// Verify all levels are present
	if !strings.Contains(output, "debug message") {
		t.Error("Debug message not found in output")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info message not found in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message not found in output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message not found in output")
	}

	// Verify fields are present
	if !strings.Contains(output, "key1=value1") {
		t.Error("Field key1 not found in output")
	}
	if !strings.Contains(output, "key2=value2") {
		t.Error("Field key2 not found in output")
	}

	// Verify levels are present
	if !strings.Contains(output, "[DEBUG]") {
		t.Error("DEBUG level not found in output")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("INFO level not found in output")
	}
	if !strings.Contains(output, "[WARN]") {
		t.Error("WARN level not found in output")
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Error("ERROR level not found in output")
	}

	// Verify timestamp format (should be ISO 8601)
	// ISO 8601 format: 2006-01-02T15:04:05.000Z
	if !strings.Contains(output, "T") && !strings.Contains(output, "Z") {
		t.Error("Timestamp format does not appear to be ISO 8601")
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	// Create a logger with INFO level
	var buf bytes.Buffer
	logger := NewLogger(&buf, LevelInfo)

	// Log at different levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")

	output := buf.String()

	// Debug should be filtered out
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}

	// Info and Warn should be present
	if !strings.Contains(output, "info message") {
		t.Error("Info message should be in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should be in output")
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, LevelInfo)

	// Create a logger with additional fields
	loggerWithFields := logger.With(
		StringField("component", "test"),
		StringField("version", "1.0"),
	)

	// Log with the new logger
	loggerWithFields.Info("test message")

	output := buf.String()

	// Verify fields are present
	if !strings.Contains(output, "component=test") {
		t.Error("Field component not found in output")
	}
	if !strings.Contains(output, "version=1.0") {
		t.Error("Field version not found in output")
	}
}

func TestLoggerWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(&buf, LevelInfo)

	// Create a logger with error
	err := &testError{msg: "test error"}
	loggerWithError := logger.WithError(err)

	loggerWithError.Error("an error occurred")

	output := buf.String()

	// Verify error field is present
	if !strings.Contains(output, "error=test error") {
		t.Error("Error field not found in output")
	}
}

func TestNopLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewNopLogger()

	// Log to nop logger - should not write anything
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	// Try to write to buffer with nop logger
	logger.Debug("test", StringField("key", "value"))

	if buf.Len() > 0 {
		t.Error("NopLogger should not write anything")
	}
}

func TestDefaultLogger(t *testing.T) {
	// Set a custom default logger
	var buf bytes.Buffer
	customLogger := NewLogger(&buf, LevelInfo)
	SetDefault(customLogger)

	// Use the default logger functions
	Info("test info", StringField("key", "value"))

	// Verify output was written to our buffer
	output := buf.String()
	if !strings.Contains(output, "test info") {
		t.Error("Default logger output not found")
	}

	// Reset to avoid affecting other tests
	SetDefault(NewDefaultLogger())
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name string
		fn   func() Field
		key  string
	}{
		{"StringField", func() Field { return StringField("key", "value") }, "key"},
		{"IntField", func() Field { return IntField("num", 42) }, "num"},
		{"ErrField", func() Field { return ErrField(&testError{msg: "test"}) }, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.fn()
			if field.Key != tt.key {
				t.Errorf("Field key = %v, want %v", field.Key, tt.key)
			}
		})
	}

	// Test ErrField with nil error
	field := ErrField(nil)
	if field.Value != "<nil>" {
		t.Errorf("ErrField(nil) = %v, want <nil>", field.Value)
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
