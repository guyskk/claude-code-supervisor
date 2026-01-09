package cli

import (
	"strings"
	"testing"
)

func TestParseStructuredOutput(t *testing.T) {
	tests := []struct {
		name         string
		output       interface{}
		wantComp     bool
		wantFeedback string
		wantErr      bool
	}{
		{
			name: "valid structured output",
			output: map[string]interface{}{
				"completed": true,
				"feedback":  "Task completed successfully",
			},
			wantComp:     true,
			wantFeedback: "Task completed successfully",
			wantErr:      false,
		},
		{
			name: "completed false with feedback",
			output: map[string]interface{}{
				"completed": false,
				"feedback":  "Task not complete, please continue",
			},
			wantComp:     false,
			wantFeedback: "Task not complete, please continue",
			wantErr:      false,
		},
		{
			name: "missing completed field",
			output: map[string]interface{}{
				"feedback": "some feedback",
			},
			wantErr: true,
		},
		{
			name: "missing feedback field",
			output: map[string]interface{}{
				"completed": true,
			},
			wantErr: true,
		},
		{
			name: "invalid completed type",
			output: map[string]interface{}{
				"completed": "true",
				"feedback":  "test",
			},
			wantErr: true,
		},
		{
			name: "invalid feedback type",
			output: map[string]interface{}{
				"completed": true,
				"feedback":  123,
			},
			wantErr: true,
		},
		{
			name:     "not a map",
			output:   "just a string",
			wantErr:  true,
		},
		{
			name:     "nil output",
			output:   nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseStructuredOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStructuredOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Completed != tt.wantComp {
					t.Errorf("parseStructuredOutput() completed = %v, want %v", result.Completed, tt.wantComp)
				}
				if result.Feedback != tt.wantFeedback {
					t.Errorf("parseStructuredOutput() feedback = %v, want %v", result.Feedback, tt.wantFeedback)
				}
			}
		})
	}
}

func TestParseResultJSON(t *testing.T) {
	tests := []struct {
		name         string
		jsonText     string
		wantComp     bool
		wantFeedback string
		wantErr      bool
	}{
		{
			name:         "valid json",
			jsonText:     `{"completed": true, "feedback": "done"}`,
			wantComp:     true,
			wantFeedback: "done",
			wantErr:      false,
		},
		{
			name:         "completed false",
			jsonText:     `{"completed": false, "feedback": "incomplete"}`,
			wantComp:     false,
			wantFeedback: "incomplete",
			wantErr:      false,
		},
		{
			// llmparser can repair JSON with trailing commas
			name:         "malformed json with trailing comma (repaired by llmparser)",
			jsonText:     `{"completed": true, "feedback": "test",}`,
			wantComp:     true,
			wantFeedback: "test",
			wantErr:      false,
		},
		{
			// llmparser can extract JSON from markdown code blocks
			name:         "json in markdown code block",
			jsonText:     "Some text\n```json\n{\"completed\": false, \"feedback\": \"keep going\"}\n```\nMore text",
			wantComp:     false,
			wantFeedback: "keep going",
			wantErr:      false,
		},
		{
			// Missing required field - schema validation fails
			name:     "missing required feedback field",
			jsonText: `{"completed": true}`,
			wantErr:  true,
		},
		{
			name:     "empty string",
			jsonText: "",
			wantErr:  true,
		},
		{
			name:     "not json",
			jsonText: `just plain text`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseResultJSON(tt.jsonText)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseResultJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Completed != tt.wantComp {
					t.Errorf("parseResultJSON() completed = %v, want %v", result.Completed, tt.wantComp)
				}
				if result.Feedback != tt.wantFeedback {
					t.Errorf("parseResultJSON() feedback = %v, want %v", result.Feedback, tt.wantFeedback)
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
	// Check that the prompt mentions JSON Schema format (new implementation)
	if !strings.Contains(prompt, "JSON Schema") && !strings.Contains(prompt, "JSON格式") {
		t.Error("getDefaultSupervisorPrompt() should mention JSON Schema format")
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

	// Check properties exist
	if _, ok := properties["completed"]; !ok {
		t.Error("schema missing 'completed' property")
	}
	if _, ok := properties["feedback"]; !ok {
		t.Error("schema missing 'feedback' property")
	}
}
