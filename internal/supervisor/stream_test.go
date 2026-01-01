// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"testing"
)

func TestParseStreamJSONLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *StreamMessage
		wantErr bool
	}{
		{
			name: "valid message with sessionId",
			line: `{"type":"text","sessionId":"abc-123","content":"hello"}`,
			want: &StreamMessage{
				Type:      "text",
				SessionID: "abc-123",
				Content:   "hello",
			},
			wantErr: false,
		},
		{
			name:    "empty line",
			line:    "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "whitespace only",
			line:    "   ",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "invalid json",
			line:    `{invalid json}`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid message without sessionId",
			line: `{"type":"error","content":"error message"}`,
			want: &StreamMessage{
				Type:    "error",
				Content: "error message",
			},
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
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseStreamJSONLine() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("ParseStreamJSONLine() = nil, want %v", tt.want)
				return
			}
			if got.Type != tt.want.Type {
				t.Errorf("ParseStreamJSONLine().Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.SessionID != tt.want.SessionID {
				t.Errorf("ParseStreamJSONLine().SessionID = %v, want %v", got.SessionID, tt.want.SessionID)
			}
			if got.Content != tt.want.Content {
				t.Errorf("ParseStreamJSONLine().Content = %v, want %v", got.Content, tt.want.Content)
			}
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name string
		msg  *StreamMessage
		want string
	}{
		{
			name: "valid session ID",
			msg:  &StreamMessage{SessionID: "abc-123"},
			want: "abc-123",
		},
		{
			name: "empty session ID",
			msg:  &StreamMessage{SessionID: ""},
			want: "",
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractSessionID(tt.msg); got != tt.want {
				t.Errorf("ExtractSessionID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectAgentWaiting(t *testing.T) {
	tests := []struct {
		name string
		msg  *StreamMessage
		want bool
	}{
		{
			name: "result type - waiting",
			msg:  &StreamMessage{Type: "result"},
			want: true,
		},
		{
			name: "text type - not waiting",
			msg:  &StreamMessage{Type: "text"},
			want: false,
		},
		{
			name: "nil message",
			msg:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectAgentWaiting(tt.msg); got != tt.want {
				t.Errorf("DetectAgentWaiting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTaskCompleted(t *testing.T) {
	tests := []struct {
		name   string
		output string
		marker string
		want   bool
	}{
		{
			name:   "contains marker",
			output: "Some text [TASK_COMPLETED] more text",
			marker: "[TASK_COMPLETED]",
			want:   true,
		},
		{
			name:   "does not contain marker",
			output: "Some text without marker",
			marker: "[TASK_COMPLETED]",
			want:   false,
		},
		{
			name:   "empty output",
			output: "",
			marker: "[TASK_COMPLETED]",
			want:   false,
		},
		{
			name:   "marker only",
			output: "[TASK_COMPLETED]",
			marker: "[TASK_COMPLETED]",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTaskCompleted(tt.output, tt.marker); got != tt.want {
				t.Errorf("IsTaskCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}
