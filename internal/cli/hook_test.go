package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guyskk/ccc/internal/config"
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
	// Save original GetDirFunc to restore after test
	originalGetDirFunc := config.GetDirFunc
	defer func() { config.GetDirFunc = originalGetDirFunc }()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	config.GetDirFunc = func() string { return tempDir }

	t.Run("default prompt when no custom file", func(t *testing.T) {
		prompt, source := getDefaultSupervisorPrompt()
		if prompt == "" {
			t.Error("getDefaultSupervisorPrompt() returned empty string")
		}
		if source != "supervisor_prompt_default" {
			t.Errorf("getDefaultSupervisorPrompt() source = %q, want %q", source, "supervisor_prompt_default")
		}
		// Check that key parts are present (prompt is in Chinese)
		// The prompt uses "审查者" (reviewer) instead of "Supervisor"
		if !strings.Contains(prompt, "审查者") && !strings.Contains(prompt, "Supervisor") {
			t.Error("getDefaultSupervisorPrompt() missing '审查者' or 'Supervisor'")
		}
		if !strings.Contains(prompt, "allow_stop") {
			t.Error("getDefaultSupervisorPrompt() missing 'allow_stop'")
		}
		if !strings.Contains(prompt, "feedback") {
			t.Error("getDefaultSupervisorPrompt() missing 'feedback'")
		}
		// Check that the prompt mentions JSON output format
		if !strings.Contains(prompt, "JSON") && !strings.Contains(prompt, "json") {
			t.Error("getDefaultSupervisorPrompt() should mention JSON format")
		}
		// Check for key sections in the Chinese prompt
		if !strings.Contains(prompt, "暂停当前") && !strings.Contains(prompt, "审查框架") {
			t.Error("getDefaultSupervisorPrompt() should contain key sections")
		}
	})

	t.Run("custom prompt from SUPERVISOR.md", func(t *testing.T) {
		customPrompt := "Custom supervisor prompt for testing"
		customPath := filepath.Join(tempDir, "SUPERVISOR.md")
		if err := os.WriteFile(customPath, []byte(customPrompt), 0644); err != nil {
			t.Fatalf("failed to write custom prompt file: %v", err)
		}

		prompt, source := getDefaultSupervisorPrompt()
		if prompt != customPrompt {
			t.Errorf("getDefaultSupervisorPrompt() = %q, want %q", prompt, customPrompt)
		}
		if source != customPath {
			t.Errorf("getDefaultSupervisorPrompt() source = %q, want %q", source, customPath)
		}
	})

	t.Run("empty custom file falls back to default", func(t *testing.T) {
		customPath := filepath.Join(tempDir, "SUPERVISOR.md")
		if err := os.WriteFile(customPath, []byte("   \n\t  "), 0644); err != nil {
			t.Fatalf("failed to write empty custom prompt file: %v", err)
		}

		prompt, source := getDefaultSupervisorPrompt()
		if prompt == "" {
			t.Error("getDefaultSupervisorPrompt() returned empty string for empty custom file")
		}
		if source != "supervisor_prompt_default" {
			t.Errorf("getDefaultSupervisorPrompt() source = %q, want %q", source, "supervisor_prompt_default")
		}
		// Should use default prompt (contains Chinese keywords)
		if !strings.Contains(prompt, "审查者") && !strings.Contains(prompt, "Supervisor") {
			t.Error("getDefaultSupervisorPrompt() should use default prompt when custom file is empty")
		}
	})

	t.Run("custom file with only whitespace falls back to default", func(t *testing.T) {
		customPath := filepath.Join(tempDir, "SUPERVISOR.md")
		if err := os.WriteFile(customPath, []byte("\n\n"), 0644); err != nil {
			t.Fatalf("failed to write whitespace-only custom prompt file: %v", err)
		}

		prompt, source := getDefaultSupervisorPrompt()
		if prompt == "" {
			t.Error("getDefaultSupervisorPrompt() returned empty string for whitespace-only custom file")
		}
		if source != "supervisor_prompt_default" {
			t.Errorf("getDefaultSupervisorPrompt() source = %q, want %q", source, "supervisor_prompt_default")
		}
		// Should use default prompt
		if !strings.Contains(prompt, "审查者") && !strings.Contains(prompt, "Supervisor") {
			t.Error("getDefaultSupervisorPrompt() should use default prompt when custom file is whitespace-only")
		}
	})
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
