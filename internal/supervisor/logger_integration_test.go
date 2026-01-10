// Package supervisor provides supervisor-specific logging functionality.
package supervisor

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestSupervisorLogger_Output verifies that logs are written to both stderr and file.
func TestSupervisorLogger_Output(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create a mock GetStateDir function
	origGetStateDir := getStateDirFunc
	getStateDirFunc = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getStateDirFunc = origGetStateDir }()

	supervisorID := "test-session"
	log := NewSupervisorLogger(supervisorID)

	// Test logging
	log.Info("test message", "key1", "value1", "key2", 42)
	log.Debug("debug message", "debug_key", "debug_value")
	log.Warn("warning message", "warn_key", "warn_value")
	log.Error("error message", "error_key", "error_value")

	// Close the logger to flush and close the file
	if handler, ok := log.Handler().(*SupervisorLogger); ok {
		if err := handler.Close(); err != nil {
			t.Fatalf("failed to close logger: %v", err)
		}
	}

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "supervisor-"+supervisorID+".log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify all messages are in the log
	if !strings.Contains(logContent, "test message") {
		t.Error("log file missing 'test message'")
	}
	if !strings.Contains(logContent, "key1=value1") {
		t.Error("log file missing 'key1=value1'")
	}
	if !strings.Contains(logContent, "key2=42") {
		t.Error("log file missing 'key2=42'")
	}
	if !strings.Contains(logContent, "debug message") {
		t.Error("log file missing 'debug message'")
	}
	if !strings.Contains(logContent, "warning message") {
		t.Error("log file missing 'warning message'")
	}
	if !strings.Contains(logContent, "error message") {
		t.Error("log file missing 'error message'")
	}

	// Note: supervisor_id is no longer automatically added to logs.
	// Callers can explicitly include it if needed (e.g., in exec.go).

	// Verify log file contains proper number of lines (each log should have one line)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 log lines, got %d", len(lines))
	}
}

// TestSupervisorLogger_NoSupervisorID verifies that logs only go to stderr when supervisorID is empty.
func TestSupervisorLogger_NoSupervisorID(t *testing.T) {
	// Note: We can't easily redirect stderr in a test without affecting the whole process,
	// so we just verify that no file is created

	tempDir := t.TempDir()
	origGetStateDir := getStateDirFunc
	getStateDirFunc = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getStateDirFunc = origGetStateDir }()

	// Create logger without supervisorID
	log := NewSupervisorLogger("")

	// Test logging - should not create any file
	log.Info("test message")

	// Verify no log file was created
	logFilePath := filepath.Join(tempDir, "supervisor-.log")
	if _, err := os.Stat(logFilePath); !os.IsNotExist(err) {
		t.Error("log file should not exist when supervisorID is empty")
	}
}

// TestSupervisorLogger_Close verifies that Close() properly closes the file.
func TestSupervisorLogger_Close(t *testing.T) {
	tempDir := t.TempDir()

	origGetStateDir := getStateDirFunc
	getStateDirFunc = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getStateDirFunc = origGetStateDir }()

	supervisorID := "test-close"
	log := NewSupervisorLogger(supervisorID)

	handler, ok := log.Handler().(*SupervisorLogger)
	if !ok {
		t.Fatal("expected SupervisorLogger handler")
	}

	// Log some messages
	log.Info("before close")

	// Close the logger
	if err := handler.Close(); err != nil {
		t.Fatalf("failed to close logger: %v", err)
	}

	// Verify file was closed by checking logFilePath
	logFilePath := filepath.Join(tempDir, "supervisor-"+supervisorID+".log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "before close") {
		t.Error("log file missing 'before close' message")
	}

	// Close should be idempotent
	if err := handler.Close(); err != nil {
		t.Errorf("close should be idempotent: %v", err)
	}
}

// TestSupervisorLogger_WithAttrs verifies that WithAttrs works correctly.
func TestSupervisorLogger_WithAttrs(t *testing.T) {
	tempDir := t.TempDir()

	origGetStateDir := getStateDirFunc
	getStateDirFunc = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getStateDirFunc = origGetStateDir }()

	supervisorID := "test-attrs"
	log := NewSupervisorLogger(supervisorID)

	// Add additional attributes
	logWithAttrs := log.With("extra_key", "extra_value")

	logWithAttrs.Info("test with attrs")

	// Close the logger
	if handler, ok := log.Handler().(*SupervisorLogger); ok {
		handler.Close()
	}

	// Verify log file
	logFilePath := filepath.Join(tempDir, "supervisor-"+supervisorID+".log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "extra_key=extra_value") {
		t.Error("log file missing extra attribute from With()")
	}
}

// TestOutputDecision_JSONFormat verifies that OutputDecision produces correct JSON.
func TestOutputDecision_JSONFormat(t *testing.T) {
	tests := []struct {
		name       string
		allowStop  bool
		feedback   string
		wantJSON   string
		wantReason string
	}{
		{
			name:      "allow stop - empty feedback",
			allowStop: true,
			feedback:  "",
			wantJSON:  `{"reason":""}` + "\n",
		},
		{
			name:      "allow stop - with feedback",
			allowStop: true,
			feedback:  "some feedback",
			wantJSON:  `{"reason":"some feedback"}` + "\n",
		},
		{
			name:       "block stop - with feedback",
			allowStop:  false,
			feedback:   "needs more work",
			wantJSON:   `{"decision":"block","reason":"needs more work"}` + "\n",
			wantReason: "needs more work",
		},
		{
			name:       "block stop - empty feedback uses default",
			allowStop:  false,
			feedback:   "",
			wantJSON:   `{"decision":"block","reason":"Please continue completing the task"}` + "\n",
			wantReason: "Please continue completing the task",
		},
		{
			name:       "block stop - whitespace feedback uses default",
			allowStop:  false,
			feedback:   "   ",
			wantJSON:   `{"decision":"block","reason":"Please continue completing the task"}` + "\n",
			wantReason: "Please continue completing the task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			var stdoutBuf bytes.Buffer

			// Redirect stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				os.Stdout = oldStdout
				r.Close()
				w.Close()
			}()

			// Create logger that writes to stderr only (we don't care about logs in this test)
			log := slog.New(slog.NewTextHandler(os.Stderr, nil))

			// Call OutputDecision
			err := OutputDecision(log, tt.allowStop, tt.feedback)
			if err != nil {
				t.Fatalf("OutputDecision failed: %v", err)
			}

			// Close the write end
			w.Close()

			// Read from the pipe
			os.Stdout = oldStdout
			stdoutBuf.ReadFrom(r)

			// Verify JSON output
			gotJSON := stdoutBuf.String()
			if gotJSON != tt.wantJSON {
				t.Errorf("JSON output mismatch\nGot:      %q\nExpected: %q", gotJSON, tt.wantJSON)
			}

			// Parse JSON to verify it's valid
			var parsed map[string]interface{}
			if err := json.Unmarshal(stdoutBuf.Bytes(), &parsed); err != nil {
				t.Errorf("invalid JSON output: %v", err)
			}

			// Verify decision field
			if tt.allowStop {
				// When allowStop=true, decision field should be omitted (not present)
				if _, exists := parsed["decision"]; exists {
					t.Errorf("decision should be omitted when allowStop=true, got %v", parsed["decision"])
				}
			} else {
				if decision, exists := parsed["decision"]; !exists || decision != "block" {
					t.Errorf("decision should be 'block', got %v", decision)
				}
			}

			// Verify reason field
			wantReason := tt.wantReason
			if wantReason == "" {
				wantReason = tt.feedback
			}
			if reason, exists := parsed["reason"]; !exists || reason != wantReason {
				t.Errorf("reason should be %q, got %v", wantReason, reason)
			}
		})
	}
}

// TestSupervisorLogger_LogLevels verifies that different log levels produce correct output.
func TestSupervisorLogger_LogLevels(t *testing.T) {
	tempDir := t.TempDir()

	origGetStateDir := getStateDirFunc
	getStateDirFunc = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getStateDirFunc = origGetStateDir }()

	supervisorID := "test-levels"
	log := NewSupervisorLogger(supervisorID)

	// Test all log levels with structured attributes
	log.Debug("debug message", "debug_key", "debug_value", "count", 1)
	log.Info("info message", "info_key", "info_value", "count", 2)
	log.Warn("warning message", "warn_key", "warn_value", "count", 3)
	log.Error("error message", "error_key", "error_value", "count", 4)

	// Close the logger
	if handler, ok := log.Handler().(*SupervisorLogger); ok {
		handler.Close()
	}

	// Verify log file content
	logFilePath := filepath.Join(tempDir, "supervisor-"+supervisorID+".log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify each log level message is present
	tests := []struct {
		level      string
		message    string
		key        string
		value      string
		countValue string
	}{
		{"DEBUG", "debug message", "debug_key", "debug_value", "count=1"},
		{"INFO", "info message", "info_key", "info_value", "count=2"},
		{"WARN", "warning message", "warn_key", "warn_value", "count=3"},
		{"ERROR", "error message", "error_key", "error_value", "count=4"},
	}

	for _, tt := range tests {
		// Verify log level is present
		if !strings.Contains(logContent, tt.level) {
			t.Errorf("log file missing level %s", tt.level)
		}
		// Verify message is present
		if !strings.Contains(logContent, tt.message) {
			t.Errorf("log file missing message %q", tt.message)
		}
		// Verify key-value pairs are present
		if !strings.Contains(logContent, tt.key+"="+tt.value) {
			t.Errorf("log file missing %s=%s", tt.key, tt.value)
		}
		// Verify count attribute
		if !strings.Contains(logContent, tt.countValue) {
			t.Errorf("log file missing %s", tt.countValue)
		}
	}
}

// TestSupervisorLogger_StructuredLogging verifies structured logging format.
func TestSupervisorLogger_StructuredLogging(t *testing.T) {
	tempDir := t.TempDir()

	origGetStateDir := getStateDirFunc
	getStateDirFunc = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getStateDirFunc = origGetStateDir }()

	supervisorID := "test-structured"
	log := NewSupervisorLogger(supervisorID)

	// Test various attribute types
	log.Info("test attributes",
		"string_attr", "string_value",
		"int_attr", 42,
		"bool_attr", true,
		"float_attr", 3.14,
	)

	// Close the logger
	if handler, ok := log.Handler().(*SupervisorLogger); ok {
		handler.Close()
	}

	// Verify log file content
	logFilePath := filepath.Join(tempDir, "supervisor-"+supervisorID+".log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify all attributes are present in the log
	expectedAttrs := []string{
		"string_attr=string_value",
		"int_attr=42",
		"bool_attr=true",
		"float_attr=3.14",
	}

	for _, attr := range expectedAttrs {
		if !strings.Contains(logContent, attr) {
			t.Errorf("log file missing attribute %s", attr)
		}
	}

	// Verify message is present
	if !strings.Contains(logContent, "test attributes") {
		t.Error("log file missing message")
	}
}

// TestSupervisorLogger_OutputDecisionLogging verifies that OutputDecision logs correctly.
func TestSupervisorLogger_OutputDecisionLogging(t *testing.T) {
	tests := []struct {
		name      string
		allowStop bool
		feedback  string
		wantLog   string
	}{
		{
			name:      "allow stop",
			allowStop: true,
			feedback:  "",
			wantLog:   "supervisor output: allow stop",
		},
		{
			name:      "block with feedback",
			allowStop: false,
			feedback:  "needs improvement",
			wantLog:   "supervisor output: not allow stop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			origGetStateDir := getStateDirFunc
			getStateDirFunc = func() (string, error) {
				return tempDir, nil
			}
			defer func() { getStateDirFunc = origGetStateDir }()

			supervisorID := "test-output-decision"
			log := NewSupervisorLogger(supervisorID)
			logFilePath := filepath.Join(tempDir, "supervisor-"+supervisorID+".log")

			// Call OutputDecision
			if err := OutputDecision(log, tt.allowStop, tt.feedback); err != nil {
				t.Fatalf("OutputDecision failed: %v", err)
			}

			// Close the logger
			if handler, ok := log.Handler().(*SupervisorLogger); ok {
				handler.Close()
			}

			// Verify log file content
			content, err := os.ReadFile(logFilePath)
			if err != nil {
				t.Fatalf("failed to read log file: %v", err)
			}

			logContent := string(content)

			// Verify the log message is present
			if !strings.Contains(logContent, tt.wantLog) {
				t.Errorf("log file missing expected message %q, got: %s", tt.wantLog, logContent)
			}

			// For block case, verify feedback is logged
			if !tt.allowStop && tt.feedback != "" {
				// New friendly format: feedback=value (without quotes)
				if !strings.Contains(logContent, `feedback=`+tt.feedback) {
					t.Errorf("log file missing feedback, got: %s", logContent)
				}
			}
		})
	}
}

// TestSupervisorLogger_ConcurrentLogging tests concurrent log writes.
func TestSupervisorLogger_ConcurrentLogging(t *testing.T) {
	tempDir := t.TempDir()

	origGetStateDir := getStateDirFunc
	getStateDirFunc = func() (string, error) {
		return tempDir, nil
	}
	defer func() { getStateDirFunc = origGetStateDir }()

	supervisorID := "test-concurrent"
	log := NewSupervisorLogger(supervisorID)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				log.Info("concurrent message", "goroutine", id, "iteration", j)
			}
		}(i)
	}

	wg.Wait()

	// Close the logger
	if handler, ok := log.Handler().(*SupervisorLogger); ok {
		handler.Close()
	}

	// Verify log file was created
	logFilePath := filepath.Join(tempDir, "supervisor-"+supervisorID+".log")
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	logContent := string(content)
	lines := strings.Split(strings.TrimSpace(logContent), "\n")

	// We expect 10 goroutines * 50 iterations + 1 "all done" = 501 lines
	// But we also have the initial "Supervisor started" message
	// So total should be 502 lines
	// However, due to timing, the "all done" might come from different goroutines
	// Let's just check we have a reasonable number of lines
	if len(lines) < 500 {
		t.Errorf("expected at least 500 log lines, got %d", len(lines))
	}

	// Verify no lines are corrupted (all should have timestamp prefix)
	for i, line := range lines {
		if len(line) < 30 {
			t.Errorf("line %d is too short: %q", i, line)
		}
	}
}
