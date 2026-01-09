package cli

import (
	"strings"
	"testing"
)

func TestParseResultJSON(t *testing.T) {
	tests := []struct {
		name         string
		jsonText     string
		wantAllow    bool
		wantFeedback string
	}{
		{
			name:         "valid json - allow stop true",
			jsonText:     `{"allow_stop": true, "feedback": "work is complete"}`,
			wantAllow:    true,
			wantFeedback: "work is complete",
		},
		{
			name:         "valid json - allow stop false",
			jsonText:     `{"allow_stop": false, "feedback": "needs more work"}`,
			wantAllow:    false,
			wantFeedback: "needs more work",
		},
		{
			// llmparser can repair JSON with trailing commas
			name:         "malformed json with trailing comma (repaired by llmparser)",
			jsonText:     `{"allow_stop": true, "feedback": "test",}`,
			wantAllow:    true,
			wantFeedback: "test",
		},
		{
			// llmparser can extract JSON from markdown code blocks
			name:         "json in markdown code block",
			jsonText:     "Some text\n```json\n{\"allow_stop\": false, \"feedback\": \"keep going\"}\n```\nMore text",
			wantAllow:    false,
			wantFeedback: "keep going",
		},
		{
			// Fallback: missing required field - use original text as feedback
			name:         "missing required feedback field - fallback",
			jsonText:     `{"allow_stop": true}`,
			wantAllow:    false,
			wantFeedback: `{"allow_stop": true}`,
		},
		{
			// Fallback: missing required allow_stop field - use original text as feedback
			name:         "missing required allow_stop field - fallback",
			jsonText:     `{"feedback": "some feedback"}`,
			wantAllow:    false,
			wantFeedback: `{"feedback": "some feedback"}`,
		},
		{
			// Fallback: empty string - use default feedback
			name:         "empty string - fallback with default",
			jsonText:     "",
			wantAllow:    false,
			wantFeedback: "请继续完成任务",
		},
		{
			// Fallback: not json - use original text as feedback
			name:         "not json - fallback",
			jsonText:     "just plain text",
			wantAllow:    false,
			wantFeedback: "just plain text",
		},
		{
			// Fallback: invalid JSON-like content
			name:         "invalid json - fallback",
			jsonText:     "{broken json",
			wantAllow:    false,
			wantFeedback: "{broken json",
		},
		{
			// Fallback: whitespace only - use default feedback
			name:         "whitespace only - fallback with default",
			jsonText:     "   \n\t  ",
			wantAllow:    false,
			wantFeedback: "请继续完成任务",
		},
		{
			// Fallback: Chinese text feedback
			name:         "chinese feedback - fallback",
			jsonText:     "任务还没有完成，请继续",
			wantAllow:    false,
			wantFeedback: "任务还没有完成，请继续",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseResultJSON(tt.jsonText)
			if result == nil {
				t.Fatal("parseResultJSON() returned nil result")
			}
			if result.AllowStop != tt.wantAllow {
				t.Errorf("parseResultJSON() allow_stop = %v, want %v", result.AllowStop, tt.wantAllow)
			}
			if result.Feedback != tt.wantFeedback {
				t.Errorf("parseResultJSON() feedback = %q, want %q", result.Feedback, tt.wantFeedback)
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
	if !strings.Contains(prompt, "allow_stop") {
		t.Error("getDefaultSupervisorPrompt() missing 'allow_stop'")
	}
	if !strings.Contains(prompt, "feedback") {
		t.Error("getDefaultSupervisorPrompt() missing 'feedback'")
	}
	// Check that the prompt mentions JSON code block format
	if !strings.Contains(prompt, "JSON") && !strings.Contains(prompt, "json") {
		t.Error("getDefaultSupervisorPrompt() should mention JSON format")
	}
}

func TestSupervisorResultSchema(t *testing.T) {
	// Verify the schema has the correct structure
	if supervisorResultSchema == nil {
		t.Fatal("supervisorResultSchema is nil")
	}

	schemaMap := supervisorResultSchema

	if schemaMap["type"] != "object" {
		t.Errorf("schema type = %v, want 'object'", schemaMap["type"])
	}

	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("schema properties is not a map")
	}

	// Check required fields
	required, ok := schemaMap["required"].([]string)
	if !ok {
		t.Fatal("schema required is not a string slice")
	}
	if len(required) != 2 {
		t.Errorf("required fields count = %d, want 2", len(required))
	}

	// Check required field names
	if required[0] != "allow_stop" && required[1] != "allow_stop" {
		t.Error("schema required fields should include 'allow_stop'")
	}
	if required[0] != "feedback" && required[1] != "feedback" {
		t.Error("schema required fields should include 'feedback'")
	}

	// Check properties exist
	if _, ok := properties["allow_stop"]; !ok {
		t.Error("schema missing 'allow_stop' property")
	}
	if _, ok := properties["feedback"]; !ok {
		t.Error("schema missing 'feedback' property")
	}
}
