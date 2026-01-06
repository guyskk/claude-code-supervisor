// Package supervisor provides unit tests for stream-json parsing.
package supervisor

import (
	"encoding/json"
	"testing"
)

// TestParseStreamJSONLine tests the ParseStreamJSONLine function.
func TestParseStreamJSONLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantNil bool
		wantErr bool
		check   func(*testing.T, *StreamMessage)
	}{
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:    "whitespace only",
			line:    "   \n\t  ",
			wantNil: true,
		},
		{
			name: "text message",
			line: `{"type":"text","content":"Hello, world!"}`,
			check: func(t *testing.T, msg *StreamMessage) {
				if msg.Type != "text" {
					t.Errorf("Type = %s, want text", msg.Type)
				}
				if msg.Content != "Hello, world!" {
					t.Errorf("Content = %s, want 'Hello, world!'", msg.Content)
				}
			},
		},
		{
			name: "message with session_id",
			line: `{"type":"text","session_id":"abc123","content":"test"}`,
			check: func(t *testing.T, msg *StreamMessage) {
				if msg.SessionID != "abc123" {
					t.Errorf("SessionID = %s, want abc123", msg.SessionID)
				}
			},
		},
		{
			name: "result message with text result",
			line: `{"type":"result","result":"some result text"}`,
			check: func(t *testing.T, msg *StreamMessage) {
				if msg.Type != "result" {
					t.Errorf("Type = %s, want result", msg.Type)
				}
				if msg.Result != "some result text" {
					t.Errorf("Result = %s, want 'some result text'", msg.Result)
				}
			},
		},
		{
			name: "result message with structured output",
			line: `{"type":"result","structured_output":{"completed":true,"feedback":"Great job!"}}`,
			check: func(t *testing.T, msg *StreamMessage) {
				if msg.StructuredOutput == nil {
					t.Fatal("StructuredOutput = nil, want non-nil")
				}
				if !msg.StructuredOutput.Completed {
					t.Errorf("Completed = false, want true")
				}
				if msg.StructuredOutput.Feedback != "Great job!" {
					t.Errorf("Feedback = %s, want 'Great job!'", msg.StructuredOutput.Feedback)
				}
			},
		},
		{
			name: "whitespace trimmed from line",
			line: `  {"type":"text","content":"test"}  `,
			check: func(t *testing.T, msg *StreamMessage) {
				if msg.Type != "text" {
					t.Errorf("Type = %s, want text (whitespace should be trimmed)", msg.Type)
				}
			},
		},
		{
			name:    "invalid JSON",
			line:    `{invalid json}`,
			wantNil: true,
			wantErr: true,
		},
		{
			name:    "non-object JSON",
			line:    `["array","values"]`,
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStreamJSONLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStreamJSONLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("ParseStreamJSONLine() got = %v, wantNil %v", got, tt.wantNil)
				return
			}
			if tt.check != nil && got != nil {
				tt.check(t, got)
			}
		})
	}
}

// TestSupervisorResult tests the SupervisorResult struct.
func TestSupervisorResult(t *testing.T) {
	t.Run("completed task", func(t *testing.T) {
		resultJSON := `{"completed":true,"feedback":"Task completed successfully"}`
		var result SupervisorResult
		if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if !result.Completed {
			t.Error("Completed = false, want true")
		}
		if result.Feedback != "Task completed successfully" {
			t.Errorf("Feedback = %s, want 'Task completed successfully'", result.Feedback)
		}
	})

	t.Run("incomplete task with feedback", func(t *testing.T) {
		resultJSON := `{"completed":false,"feedback":"Need to add more tests"}`
		var result SupervisorResult
		if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if result.Completed {
			t.Error("Completed = true, want false")
		}
		if result.Feedback != "Need to add more tests" {
			t.Errorf("Feedback = %s, want 'Need to add more tests'", result.Feedback)
		}
	})

	t.Run("incomplete task with empty feedback", func(t *testing.T) {
		resultJSON := `{"completed":false,"feedback":""}`
		var result SupervisorResult
		if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if result.Completed {
			t.Error("Completed = true, want false")
		}
		if result.Feedback != "" {
			t.Errorf("Feedback = %s, want empty string", result.Feedback)
		}
	})
}

// TestParseSupervisorResult tests the ParseSupervisorResult function.
func TestParseSupervisorResult(t *testing.T) {
	tests := []struct {
		name          string
		resultJSON    string
		wantCompleted bool
		wantFeedback  string
		wantErr       bool
	}{
		{
			name:          "completed task",
			resultJSON:    `{"completed":true,"feedback":"Done"}`,
			wantCompleted: true,
			wantFeedback:  "Done",
		},
		{
			name:          "incomplete task",
			resultJSON:    `{"completed":false,"feedback":"Keep working"}`,
			wantCompleted: false,
			wantFeedback:  "Keep working",
		},
		{
			name:       "invalid JSON",
			resultJSON: `{invalid}`,
			wantErr:    true,
		},
		{
			// Note: JSON unmarshaling doesn't return error for missing fields
			// Missing "completed" field defaults to false (zero value)
			name:          "missing completed field defaults to false",
			resultJSON:    `{"feedback":"test"}`,
			wantCompleted: false,
			wantFeedback:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSupervisorResult(tt.resultJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSupervisorResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Completed != tt.wantCompleted {
					t.Errorf("ParseSupervisorResult() Completed = %v, want %v", got.Completed, tt.wantCompleted)
				}
				if got.Feedback != tt.wantFeedback {
					t.Errorf("ParseSupervisorResult() Feedback = %s, want %s", got.Feedback, tt.wantFeedback)
				}
			}
		})
	}
}

// BenchmarkParseStreamJSONLine benchmarks the ParseStreamJSONLine function.
func BenchmarkParseStreamJSONLine(b *testing.B) {
	line := `{"type":"text","content":"This is a test message with some content"}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseStreamJSONLine(line)
	}
}
