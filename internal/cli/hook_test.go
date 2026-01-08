package cli

import (
	"strings"
	"testing"
)

func TestParseSupervisorResult(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		wantComp     bool
		wantFeedback string
		wantErr      bool
	}{
		{
			name: "valid json code block with markers",
			response: string([]byte{
				83, 111, 109, 101, 32, 116, 101, 120, 116, 32, 98, 101, 102, 111, 114, 101, 10, 96, 96, 96, 106, 115, 111, 110, 10,
				123, 10, 32, 32, 34, 99, 111, 109, 112, 108, 101, 116, 101, 100, 34, 58, 32, 116, 114, 117, 101, 44, 10, 32, 32,
				34, 102, 101, 101, 100, 98, 97, 99, 107, 34, 58, 32, 34, 84, 97, 115, 107, 32, 99, 111, 109, 112, 108, 101,
				116, 101, 100, 32, 115, 117, 99, 99, 101, 115, 115, 102, 117, 108, 108, 121, 34, 10, 125, 10, 96, 96, 96, 10,
				83, 111, 109, 101, 32, 116, 101, 120, 116, 32, 97, 102, 116, 101, 114,
			}),
			wantComp:     true,
			wantFeedback: "Task completed successfully",
			wantErr:      false,
		},
		{
			name:         "plain json without code block",
			response:     `{"completed": true, "feedback": "done"}`,
			wantComp:     true,
			wantFeedback: "done",
			wantErr:      false,
		},
		{
			name:         "json embedded in text",
			response:     `The result is {"completed": false, "feedback": "incomplete"} and that's it`,
			wantComp:     false,
			wantFeedback: "incomplete",
			wantErr:      false,
		},
		{
			name:     "no json found",
			response: `This is just plain text with no JSON at all`,
			wantErr:  true,
		},
		{
			name:     "empty response",
			response: "",
			wantErr:  true,
		},
		{
			name:     "malformed json",
			response: `{"completed": true,}`,
			wantErr:  true,
		},
		{
			name:         "json with extra whitespace",
			response:     "\n\n  {\n    \"completed\": false,\n    \"feedback\": \"Keep going\"\n  }\n\n  ",
			wantComp:     false,
			wantFeedback: "Keep going",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSupervisorResult(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSupervisorResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Completed != tt.wantComp {
					t.Errorf("parseSupervisorResult() completed = %v, want %v", result.Completed, tt.wantComp)
				}
				if result.Feedback != tt.wantFeedback {
					t.Errorf("parseSupervisorResult() feedback = %v, want %v", result.Feedback, tt.wantFeedback)
				}
			}
		})
	}
}

func TestGetDefaultSupervisorPrompt(t *testing.T) {
	prompt := getDefaultSupervisorPrompt()
	if prompt == "" {
		t.Error("getDefaultSupervisorPrompt() returned empty string")
	}
	// Check that key parts are present
	if !strings.Contains(prompt, "Supervisor") {
		t.Error("getDefaultSupervisorPrompt() missing 'Supervisor'")
	}
	if !strings.Contains(prompt, "completed") {
		t.Error("getDefaultSupervisorPrompt() missing 'completed'")
	}
	if !strings.Contains(prompt, "feedback") {
		t.Error("getDefaultSupervisorPrompt() missing 'feedback'")
	}
	if !strings.Contains(prompt, "StructuredOutput") {
		t.Error("getDefaultSupervisorPrompt() missing 'StructuredOutput'")
	}
}
