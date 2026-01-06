// Package claude_agent_sdk provides tests for the agent SDK.
package claude_agent_sdk

import (
	"testing"
	"time"
)

func TestNewAgent(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "empty config uses defaults",
			config:  &Config{},
			wantErr: false,
		},
		{
			name: "custom claude path",
			config: &Config{
				ClaudePath: "claude",
			},
			wantErr: false,
		},
		{
			name: "invalid claude path",
			config: &Config{
				ClaudePath: "/nonexistent/claude",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && agent == nil {
				t.Error("NewAgent() returned nil agent")
			}
			if !tt.wantErr && agent != nil {
				if agent.config.Timeout != 10*time.Minute {
					t.Errorf("default timeout = %v, want %v", agent.config.Timeout, 10*time.Minute)
				}
			}
		})
	}
}

func TestAgent_buildArgs(t *testing.T) {
	agent, _ := NewAgent(nil)

	tests := []struct {
		name string
		opts RunOptions
		want []string
	}{
		{
			name: "minimal options",
			opts: RunOptions{
				Prompt: "test prompt",
			},
			want: []string{"-p", "--verbose", "test prompt"},
		},
		{
			name: "with fork session",
			opts: RunOptions{
				SessionID:   "abc123",
				ForkSession: true,
				Prompt:      "test prompt",
			},
			want: []string{"-p", "--fork-session", "--resume", "abc123", "--verbose", "test prompt"},
		},
		{
			name: "with JSON schema",
			opts: RunOptions{
				Prompt:     "test prompt",
				JSONSchema: `{"type":"object"}`,
			},
			want: []string{"-p", "--verbose", "--json-schema", `{"type":"object"}`, "test prompt"},
		},
		{
			name: "stream-json format",
			opts: RunOptions{
				Prompt:       "test prompt",
				OutputFormat: "stream-json",
			},
			want: []string{"-p", "--verbose", "--output-format", "stream-json", "test prompt"},
		},
		{
			name: "text format explicitly",
			opts: RunOptions{
				Prompt:       "test prompt",
				OutputFormat: "text",
			},
			want: []string{"-p", "--verbose", "--output-format", "text", "test prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.buildArgs(tt.opts)
			if len(got) != len(tt.want) {
				t.Errorf("buildArgs() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("buildArgs()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseStreamJSONLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *StreamMessage
		wantErr bool
	}{
		{
			name:    "empty line",
			line:    "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "non-json line",
			line:    "plain text output",
			want:    nil,
			wantErr: false,
		},
		{
			name: "valid text message",
			line: `{"type":"text","content":"hello"}`,
			want: &StreamMessage{
				Type:    "text",
				Content: "hello",
			},
			wantErr: false,
		},
		{
			name: "valid result message",
			line: `{"type":"result","structured_output":{"completed":true,"feedback":"good"}}`,
			want: &StreamMessage{
				Type: "result",
				StructuredOutput: map[string]interface{}{
					"completed": true,
					"feedback":  "good",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			line:    `{invalid json}`,
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStreamJSONLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStreamJSONLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil && got != nil {
				t.Errorf("ParseStreamJSONLine() = %v, want nil", got)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("ParseStreamJSONLine() = nil, want %v", tt.want)
				return
			}
			if tt.want != nil && got != nil {
				if got.Type != tt.want.Type {
					t.Errorf("ParseStreamJSONLine().Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Content != tt.want.Content {
					t.Errorf("ParseStreamJSONLine().Content = %v, want %v", got.Content, tt.want.Content)
				}
			}
		})
	}
}

func TestParseSupervisorResult(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]interface{}
		want    *SupervisorResult
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "completed true",
			input: map[string]interface{}{
				"completed": true,
				"feedback":  "good job",
			},
			want: &SupervisorResult{
				Completed: true,
				Feedback:  "good job",
			},
			wantErr: false,
		},
		{
			name: "completed false with feedback",
			input: map[string]interface{}{
				"completed": false,
				"feedback":  "needs more work",
			},
			want: &SupervisorResult{
				Completed: false,
				Feedback:  "needs more work",
			},
			wantErr: false,
		},
		{
			name: "missing completed field",
			input: map[string]interface{}{
				"feedback": "test",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid completed type",
			input: map[string]interface{}{
				"completed": "true",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSupervisorResult(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSupervisorResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("ParseSupervisorResult() returned nil")
			}
			if !tt.wantErr && got != nil {
				if got.Completed != tt.want.Completed {
					t.Errorf("ParseSupervisorResult().Completed = %v, want %v", got.Completed, tt.want.Completed)
				}
				if got.Feedback != tt.want.Feedback {
					t.Errorf("ParseSupervisorResult().Feedback = %v, want %v", got.Feedback, tt.want.Feedback)
				}
			}
		})
	}
}

// TestProcess_Start tests Process.Start (unit test without actual execution)
func TestProcess_Start_NotStarted(t *testing.T) {
	// This is a basic test - full integration tests would require mocking
	// For now, we test that calling Start twice returns an error
	// TODO: Add mock-based tests
	t.Skip("requires mock implementation")
}

// TestRunContext tests Agent.Run with context cancellation
func TestRunContext_Timeout(t *testing.T) {
	// Integration test that requires claude to be installed
	// TODO: Add mock-based tests for unit testing
	t.Skip("requires claude to be installed")
}
