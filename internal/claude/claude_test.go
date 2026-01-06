// Package claude tests for the Claude SDK.
package claude

import (
	"errors"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		// It's OK if claude is not installed in test environment
		t.Skip("claude not found in PATH, skipping test")
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.ClaudePath == "" {
		t.Error("expected ClaudePath to be set")
	}
}

func TestNewClientWithPath(t *testing.T) {
	client := NewClientWithPath("/usr/bin/claude")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.ClaudePath != "/usr/bin/claude" {
		t.Errorf("expected ClaudePath to be /usr/bin/claude, got %s", client.ClaudePath)
	}
}

func TestBuildArgs(t *testing.T) {
	client := NewClientWithPath("/usr/bin/claude")

	tests := []struct {
		name     string
		config   *SessionConfig
		expected []string
	}{
		{
			name: "basic prompt",
			config: &SessionConfig{
				Prompt: "hello",
			},
			expected: []string{"-p", "hello"},
		},
		{
			name: "with resume",
			config: &SessionConfig{
				Prompt:   "continue",
				ResumeID: "session123",
			},
			expected: []string{"-p", "--resume", "session123", "continue"},
		},
		{
			name: "with fork session",
			config: &SessionConfig{
				Prompt:      "test",
				ForkSession: true,
			},
			expected: []string{"-p", "--fork-session", "test"},
		},
		{
			name: "with stream-json output",
			config: &SessionConfig{
				Prompt:       "test",
				OutputFormat: OutputFormatStreamJSON,
			},
			expected: []string{"-p", "--output-format", "stream-json", "test"},
		},
		{
			name: "with verbose",
			config: &SessionConfig{
				Prompt:  "test",
				Verbose: true,
			},
			expected: []string{"-p", "--verbose", "test"},
		},
		{
			name: "with JSON schema",
			config: &SessionConfig{
				Prompt:       "test",
				JSONSchema:   `{"type":"object"}`,
				OutputFormat: OutputFormatStreamJSON,
			},
			expected: []string{"-p", "--output-format", "stream-json", "--json-schema", `{"type":"object"}`, "test"},
		},
		{
			name: "with extra args",
			config: &SessionConfig{
				Prompt:    "test",
				ExtraArgs: []string{"--debug", "--timing"},
			},
			expected: []string{"-p", "--debug", "--timing", "test"},
		},
		{
			name: "complex config",
			config: &SessionConfig{
				Prompt:       "review code",
				ResumeID:     "abc123",
				ForkSession:  true,
				OutputFormat: OutputFormatStreamJSON,
				Verbose:      true,
				JSONSchema:   `{"type":"object","properties":{"done":{"type":"boolean"}}}`,
				ExtraArgs:    []string{"--debug"},
			},
			expected: []string{
				"-p",
				"--fork-session",
				"--resume", "abc123",
				"--output-format", "stream-json",
				"--verbose",
				"--json-schema", `{"type":"object","properties":{"done":{"type":"boolean"}}}`,
				"--debug",
				"review code",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := client.buildArgs(tt.config)
			if len(args) != len(tt.expected) {
				t.Errorf("expected %d args, got %d", len(tt.expected), len(args))
				t.Errorf("expected: %v", tt.expected)
				t.Errorf("got: %v", args)
				return
			}
			for i := range args {
				if args[i] != tt.expected[i] {
					t.Errorf("arg %d: expected %q, got %q", i, tt.expected[i], args[i])
				}
			}
		})
	}
}

func TestError(t *testing.T) {
	err := NewError(ErrCodeNotFound, "test message", nil)
	if err.Error() != "NOT_FOUND: test message" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	cause := &Error{Code: ErrCodeParseError, Message: "inner error"}
	err = NewError(ErrCodeExecutionError, "outer error", cause)
	expected := "EXECUTION_ERROR: outer error: PARSE_ERROR: inner error"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestSession(t *testing.T) {
	client := NewClientWithPath("/usr/bin/claude")
	config := &SessionConfig{
		Prompt: "test",
	}

	session := client.NewSession(config)
	if session == nil {
		t.Fatal("expected non-nil session")
	}

	if session.Output() != "" {
		t.Error("expected empty output initially")
	}

	// Test Done channel
	select {
	case <-session.Done():
		t.Error("Done channel should not be closed immediately")
	default:
		// OK
	}

	// Test Cancel (should not panic)
	session.Cancel()
}

func TestExecuteWithTimeout(t *testing.T) {
	client := NewClientWithPath("/usr/bin/claude")
	config := &SessionConfig{
		Prompt: "test",
	}

	// Very short timeout - should cancel quickly
	output, err := client.ExecuteWithTimeout(config, 1*time.Nanosecond)
	if err == nil {
		t.Error("expected timeout error")
	}
	if err != nil {
		var cccErr *Error
		if errors.As(err, &cccErr) {
			if cccErr.Code != ErrCodeCancelled {
				t.Logf("got error: %v (code: %s)", err, cccErr.Code)
			}
		}
	}
	_ = output // May be empty
}

// BenchmarkBuildArgs benchmarks the buildArgs function.
func BenchmarkBuildArgs(b *testing.B) {
	client := NewClientWithPath("/usr/bin/claude")
	config := &SessionConfig{
		Prompt:       "test prompt",
		ResumeID:     "session123",
		ForkSession:  true,
		OutputFormat: OutputFormatStreamJSON,
		Verbose:      true,
		JSONSchema:   `{"type":"object"}`,
		ExtraArgs:    []string{"--debug"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.buildArgs(config)
	}
}
