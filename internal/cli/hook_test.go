package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/supervisor"
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
		// The prompt uses role keywords like "监督者" (supervisor), "审查者" (reviewer), or "Supervisor"
		if !strings.Contains(prompt, "监督者") && !strings.Contains(prompt, "审查者") && !strings.Contains(prompt, "Supervisor") {
			t.Error("getDefaultSupervisorPrompt() missing role keyword ('监督者', '审查者', or 'Supervisor')")
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
		if !strings.Contains(prompt, "停止任务") && !strings.Contains(prompt, "审查框架") && !strings.Contains(prompt, "审查思维") {
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
		if !strings.Contains(prompt, "监督者") && !strings.Contains(prompt, "审查者") && !strings.Contains(prompt, "Supervisor") {
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
		if !strings.Contains(prompt, "监督者") && !strings.Contains(prompt, "审查者") && !strings.Contains(prompt, "Supervisor") {
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

func TestSupervisorResultToHookOutput_PreToolUse_Allow(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: true,
		Feedback:  "问题合理，可以向用户提问",
	}

	output := SupervisorResultToHookOutput(result, "PreToolUse")

	if output == nil {
		t.Fatal("SupervisorResultToHookOutput() returned nil")
	}

	// Should have HookSpecificOutput for PreToolUse
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil for PreToolUse event")
	}

	// Check decision is "allow"
	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("PermissionDecision = %q, want 'allow'", output.HookSpecificOutput.PermissionDecision)
	}

	// Check hook event name
	if output.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want 'PreToolUse'", output.HookSpecificOutput.HookEventName)
	}

	// Check reason
	if output.HookSpecificOutput.PermissionDecisionReason != result.Feedback {
		t.Errorf("PermissionDecisionReason = %q, want %q", output.HookSpecificOutput.PermissionDecisionReason, result.Feedback)
	}

	// Stop event fields should be empty
	if output.Decision != nil {
		t.Error("Decision should be nil for PreToolUse event")
	}
}

func TestSupervisorResultToHookOutput_PreToolUse_Deny(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: false,
		Feedback:  "应该在代码中添加更多注释后再提问",
	}

	output := SupervisorResultToHookOutput(result, "PreToolUse")

	if output == nil {
		t.Fatal("SupervisorResultToHookOutput() returned nil")
	}

	// Should have HookSpecificOutput for PreToolUse
	if output.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput is nil for PreToolUse event")
	}

	// Check decision is "deny"
	if output.HookSpecificOutput.PermissionDecision != "deny" {
		t.Errorf("PermissionDecision = %q, want 'deny'", output.HookSpecificOutput.PermissionDecision)
	}

	// Check hook event name
	if output.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want 'PreToolUse'", output.HookSpecificOutput.HookEventName)
	}

	// Check reason
	if output.HookSpecificOutput.PermissionDecisionReason != result.Feedback {
		t.Errorf("PermissionDecisionReason = %q, want %q", output.HookSpecificOutput.PermissionDecisionReason, result.Feedback)
	}

	// Stop event fields should be empty
	if output.Decision != nil {
		t.Error("Decision should be nil for PreToolUse event")
	}
}

func TestSupervisorResultToHookOutput_Stop_Allow(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: true,
		Feedback:  "工作已完成",
	}

	output := SupervisorResultToHookOutput(result, "Stop")

	if output == nil {
		t.Fatal("SupervisorResultToHookOutput() returned nil")
	}

	// Should NOT have HookSpecificOutput for Stop
	if output.HookSpecificOutput != nil {
		t.Error("HookSpecificOutput should be nil for Stop event")
	}

	// Decision should be nil (allow stop)
	if output.Decision != nil {
		t.Errorf("Decision = %q, want nil (allow stop)", *output.Decision)
	}

	// Check reason
	if output.Reason != result.Feedback {
		t.Errorf("Reason = %q, want %q", output.Reason, result.Feedback)
	}
}

func TestSupervisorResultToHookOutput_Stop_Block(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: false,
		Feedback:  "需要继续完善测试用例",
	}

	output := SupervisorResultToHookOutput(result, "Stop")

	if output == nil {
		t.Fatal("SupervisorResultToHookOutput() returned nil")
	}

	// Should NOT have HookSpecificOutput for Stop
	if output.HookSpecificOutput != nil {
		t.Error("HookSpecificOutput should be nil for Stop event")
	}

	// Decision should be "block"
	if output.Decision == nil {
		t.Fatal("Decision is nil for block case")
	}
	if *output.Decision != "block" {
		t.Errorf("Decision = %q, want 'block'", *output.Decision)
	}

	// Check reason
	if output.Reason != result.Feedback {
		t.Errorf("Reason = %q, want %q", output.Reason, result.Feedback)
	}
}

func TestSupervisorResultToHookOutput_PreToolUse_EmptyFeedback(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: true,
		Feedback:  "",
	}

	output := SupervisorResultToHookOutput(result, "PreToolUse")

	if output == nil {
		t.Fatal("SupervisorResultToHookOutput() returned nil")
	}

	// Empty feedback should still be passed through
	if output.HookSpecificOutput.PermissionDecisionReason != "" {
		t.Errorf("PermissionDecisionReason = %q, want empty string", output.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestSupervisorResultToHookOutput_Stop_EmptyFeedback(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: true,
		Feedback:  "",
	}

	output := SupervisorResultToHookOutput(result, "Stop")

	if output == nil {
		t.Fatal("SupervisorResultToHookOutput() returned nil")
	}

	// Empty feedback should still be passed through
	if output.Reason != "" {
		t.Errorf("Reason = %q, want empty string", output.Reason)
	}
}

func TestHookInput_PreToolUse_Unmarshal(t *testing.T) {
	// Test that HookInput can unmarshal PreToolUse event JSON
	jsonInput := `{
		"session_id": "test-session-123",
		"hook_event_name": "PreToolUse",
		"tool_name": "AskUserQuestion",
		"tool_input": {
			"questions": [
				{
					"question": "请选择实现方案",
					"header": "方案选择"
				}
			]
		},
		"tool_use_id": "toolu_01ABC123"
	}`

	var input HookInput
	if err := json.Unmarshal([]byte(jsonInput), &input); err != nil {
		t.Fatalf("Failed to unmarshal PreToolUse input: %v", err)
	}

	// Verify common fields
	if input.SessionID != "test-session-123" {
		t.Errorf("SessionID = %q, want 'test-session-123'", input.SessionID)
	}
	if input.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want 'PreToolUse'", input.HookEventName)
	}

	// Verify PreToolUse fields
	if input.ToolName != "AskUserQuestion" {
		t.Errorf("ToolName = %q, want 'AskUserQuestion'", input.ToolName)
	}
	if input.ToolUseID != "toolu_01ABC123" {
		t.Errorf("ToolUseID = %q, want 'toolu_01ABC123'", input.ToolUseID)
	}

	// Verify tool_input is preserved
	if len(input.ToolInput) == 0 {
		t.Error("ToolInput is empty")
	}
}

func TestHookInput_Stop_Unmarshal(t *testing.T) {
	// Test backward compatibility: old Stop event format still works
	jsonInput := `{
		"session_id": "test-session-456",
		"stop_hook_active": false
	}`

	var input HookInput
	if err := json.Unmarshal([]byte(jsonInput), &input); err != nil {
		t.Fatalf("Failed to unmarshal Stop input: %v", err)
	}

	// Verify common fields
	if input.SessionID != "test-session-456" {
		t.Errorf("SessionID = %q, want 'test-session-456'", input.SessionID)
	}

	// StopHookActive should be parsed
	if input.StopHookActive != false {
		t.Errorf("StopHookActive = %v, want false", input.StopHookActive)
	}

	// PreToolUse fields should be empty
	if input.ToolName != "" {
		t.Errorf("ToolName = %q, want empty string", input.ToolName)
	}
	if input.HookEventName != "" {
		t.Errorf("HookEventName = %q, want empty string", input.HookEventName)
	}
}

func TestHookOutput_PreToolUse_Marshal(t *testing.T) {
	output := &HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "allow",
			PermissionDecisionReason: "问题合理",
		},
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal PreToolUse output: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify the output contains PreToolUse fields
	if !strings.Contains(jsonStr, `"hookSpecificOutput"`) {
		t.Error("Output should contain 'hookSpecificOutput'")
	}
	if !strings.Contains(jsonStr, `"permissionDecision"`) {
		t.Error("Output should contain 'permissionDecision'")
	}
	if !strings.Contains(jsonStr, `"allow"`) {
		t.Error("Output should contain 'allow' decision")
	}
	if !strings.Contains(jsonStr, `"PreToolUse"`) {
		t.Error("Output should contain 'PreToolUse' event name")
	}
	// Should NOT contain Stop event fields
	if strings.Contains(jsonStr, `"decision"`) {
		t.Error("Output should NOT contain 'decision' for PreToolUse event")
	}
}

func TestHookOutput_Stop_Block_Marshal(t *testing.T) {
	block := "block"
	output := &HookOutput{
		Decision: &block,
		Reason:   "需要继续工作",
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal Stop block output: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify the output contains Stop fields
	if !strings.Contains(jsonStr, `"decision"`) {
		t.Error("Output should contain 'decision'")
	}
	if !strings.Contains(jsonStr, `"block"`) {
		t.Error("Output should contain 'block' decision")
	}
	if !strings.Contains(jsonStr, `"reason"`) {
		t.Error("Output should contain 'reason'")
	}
	// Should NOT contain PreToolUse fields
	if strings.Contains(jsonStr, `"hookSpecificOutput"`) {
		t.Error("Output should NOT contain 'hookSpecificOutput' for Stop event")
	}
}

func TestHookOutput_Stop_Allow_Marshal(t *testing.T) {
	output := &HookOutput{
		Reason: "工作已完成",
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal Stop allow output: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify the output contains reason
	if !strings.Contains(jsonStr, `"reason"`) {
		t.Error("Output should contain 'reason'")
	}
	if !strings.Contains(jsonStr, `"工作已完成"`) {
		t.Error("Output should contain the reason text")
	}
	// Should NOT contain decision (allow = omit decision)
	if strings.Contains(jsonStr, `"decision"`) {
		t.Error("Output should NOT contain 'decision' when allowing stop")
	}
	// Should NOT contain PreToolUse fields
	if strings.Contains(jsonStr, `"hookSpecificOutput"`) {
		t.Error("Output should NOT contain 'hookSpecificOutput' for Stop event")
	}
}

// ============================================================================
// Edge Case Tests
// ============================================================================

func TestSupervisorResultToHookOutput_UnknownEventType(t *testing.T) {
	// Unknown event types should default to Stop event format
	tests := []struct {
		name      string
		eventType string
	}{
		{"empty string", ""},
		{"unknown event", "UnknownEvent"},
		{"post tool use", "PostToolUse"},
		{"mixed case pretooluse", "pretooluse"},
		{"PRETOOLUSE uppercase", "PRETOOLUSE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &SupervisorResult{
				AllowStop: false,
				Feedback:  "需要继续工作",
			}

			output := SupervisorResultToHookOutput(result, tt.eventType)

			if output == nil {
				t.Fatal("SupervisorResultToHookOutput() returned nil")
			}

			// Unknown event types should default to Stop format
			if output.HookSpecificOutput != nil {
				t.Errorf("HookSpecificOutput should be nil for unknown event type %q, got non-nil", tt.eventType)
			}

			// Should have Stop event decision format
			if output.Decision == nil {
				t.Error("Decision should be present for unknown event type (defaults to Stop)")
			}
			if output.Decision != nil && *output.Decision != "block" {
				t.Errorf("Decision = %q, want 'block' for unknown event type", *output.Decision)
			}

			// Should have reason
			if output.Reason != result.Feedback {
				t.Errorf("Reason = %q, want %q", output.Reason, result.Feedback)
			}
		})
	}
}

func TestSupervisorResultToHookOutput_PreToolUse_AllowStopTrue(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: true,
		Feedback:  "问题合理，可以继续",
	}

	output := SupervisorResultToHookOutput(result, "PreToolUse")

	if output == nil {
		t.Fatal("Expected non-nil output")
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("Expected HookSpecificOutput for PreToolUse event")
	}
	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("PermissionDecision = %q, want 'allow' when AllowStop=true", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestSupervisorResultToHookOutput_PreToolUse_AllowStopFalse(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: false,
		Feedback:  "需要更多上下文",
	}

	output := SupervisorResultToHookOutput(result, "PreToolUse")

	if output == nil {
		t.Fatal("Expected non-nil output")
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("Expected HookSpecificOutput for PreToolUse event")
	}
	if output.HookSpecificOutput.PermissionDecision != "deny" {
		t.Errorf("PermissionDecision = %q, want 'deny' when AllowStop=false", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestSupervisorResultToHookOutput_Stop_AllowStopTrue(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: true,
		Feedback:  "工作完成",
	}

	output := SupervisorResultToHookOutput(result, "Stop")

	if output == nil {
		t.Fatal("Expected non-nil output")
	}
	// Allow stop: no decision field, only reason
	if output.Decision != nil {
		t.Errorf("Decision should be nil for allow stop, got %q", *output.Decision)
	}
	if output.Reason != result.Feedback {
		t.Errorf("Reason = %q, want %q", output.Reason, result.Feedback)
	}
}

func TestSupervisorResultToHookOutput_Stop_AllowStopFalse(t *testing.T) {
	result := &SupervisorResult{
		AllowStop: false,
		Feedback:  "继续工作",
	}

	output := SupervisorResultToHookOutput(result, "Stop")

	if output == nil {
		t.Fatal("Expected non-nil output")
	}
	// Block stop: decision=block, with reason
	if output.Decision == nil {
		t.Fatal("Decision should be non-nil for block")
	}
	if *output.Decision != "block" {
		t.Errorf("Decision = %q, want 'block'", *output.Decision)
	}
	if output.Reason != result.Feedback {
		t.Errorf("Reason = %q, want %q", output.Reason, result.Feedback)
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

// TestRunSupervisorHook_RecursiveCallProtection tests the CCC_SUPERVISOR_HOOK=1
// protection mechanism that prevents infinite recursion when the supervisor
// itself calls tools that might trigger hooks.
func TestRunSupervisorHook_RecursiveCallProtection(t *testing.T) {
	// Save original GetDirFunc to restore after test
	originalGetDirFunc := config.GetDirFunc
	defer func() { config.GetDirFunc = originalGetDirFunc }()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	config.GetDirFunc = func() string { return tempDir }

	// Create a minimal state file with supervisor enabled
	state := &supervisor.State{
		Enabled: true,
	}
	if err := supervisor.SaveState("test-recursive-protection", state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Set required environment variables
	oldSupervisorID := os.Getenv("CCC_SUPERVISOR_ID")
	oldSupervisorHook := os.Getenv("CCC_SUPERVISOR_HOOK")
	defer func() {
		os.Setenv("CCC_SUPERVISOR_ID", oldSupervisorID)
		os.Setenv("CCC_SUPERVISOR_HOOK", oldSupervisorHook)
	}()
	os.Setenv("CCC_SUPERVISOR_ID", "test-recursive-protection")
	os.Setenv("CCC_SUPERVISOR_HOOK", "1") // Simulate recursive call

	// Create stdin with PreToolUse hook input
	hookInputJSON := `{
		"session_id": "test-recursive-protection",
		"hook_event_name": "PreToolUse",
		"tool_name": "AskUserQuestion",
		"tool_input": {},
		"tool_use_id": "toolu_test_123"
	}`

	// Simulate stdin by creating a pipe
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.Write([]byte(hookInputJSON))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	// Run the hook command - should return early with allow decision
	opts := &SupervisorHookCommand{}
	err := RunSupervisorHook(opts)

	if err != nil {
		t.Errorf("RunSupervisorHook() error = %v, want nil (early return due to CCC_SUPERVISOR_HOOK=1)", err)
	}
}

// TestRunSupervisorHook_SupervisorModeDisabled tests that when supervisor mode
// is disabled, the hook allows the operation without calling the SDK.
func TestRunSupervisorHook_SupervisorModeDisabled(t *testing.T) {
	// Save original GetDirFunc to restore after test
	originalGetDirFunc := config.GetDirFunc
	defer func() { config.GetDirFunc = originalGetDirFunc }()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	config.GetDirFunc = func() string { return tempDir }

	// Create a state file with supervisor DISABLED
	state := &supervisor.State{
		Enabled: false,
	}
	if err := supervisor.SaveState("test-supervisor-disabled", state); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Set required environment variables
	oldSupervisorID := os.Getenv("CCC_SUPERVISOR_ID")
	defer func() { os.Setenv("CCC_SUPERVISOR_ID", oldSupervisorID) }()
	os.Setenv("CCC_SUPERVISOR_ID", "test-supervisor-disabled")

	// Create stdin with PreToolUse hook input
	hookInputJSON := `{
		"session_id": "test-supervisor-disabled",
		"hook_event_name": "PreToolUse",
		"tool_name": "AskUserQuestion",
		"tool_input": {},
		"tool_use_id": "toolu_test_456"
	}`

	// Simulate stdin by creating a pipe
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		w.Write([]byte(hookInputJSON))
		w.Close()
	}()
	defer func() { os.Stdin = oldStdin }()

	// Run the hook command - should return early with allow decision
	opts := &SupervisorHookCommand{}
	err := RunSupervisorHook(opts)

	if err != nil {
		t.Errorf("RunSupervisorHook() error = %v, want nil (supervisor disabled)", err)
	}
}

func TestHookInput_PreToolUse_FullInput(t *testing.T) {
	// Test complete PreToolUse input with all fields
	jsonInput := `{
		"session_id": "test-session-full",
		"transcript_path": "/path/to/transcript.json",
		"cwd": "/workspace/project",
		"permission_mode": "edit",
		"hook_event_name": "PreToolUse",
		"tool_name": "AskUserQuestion",
		"tool_input": {
			"questions": [
				{
					"question": "确认实施计划？",
					"header": "计划确认",
					"multiSelect": false,
					"options": [
						{"label": "确认", "description": "开始实施"},
						{"label": "取消", "description": "重新规划"}
					]
				}
			]
		},
		"tool_use_id": "toolu_full_456"
	}`

	var input HookInput
	if err := json.Unmarshal([]byte(jsonInput), &input); err != nil {
		t.Fatalf("Failed to unmarshal PreToolUse input: %v", err)
	}

	// Verify all fields
	if input.SessionID != "test-session-full" {
		t.Errorf("SessionID = %q, want 'test-session-full'", input.SessionID)
	}
	if input.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want 'PreToolUse'", input.HookEventName)
	}
	if input.ToolName != "AskUserQuestion" {
		t.Errorf("ToolName = %q, want 'AskUserQuestion'", input.ToolName)
	}
	if input.TranscriptPath != "/path/to/transcript.json" {
		t.Errorf("TranscriptPath = %q, want '/path/to/transcript.json'", input.TranscriptPath)
	}
	if input.CWD != "/workspace/project" {
		t.Errorf("CWD = %q, want '/workspace/project'", input.CWD)
	}
	if input.PermissionMode != "edit" {
		t.Errorf("PermissionMode = %q, want 'edit'", input.PermissionMode)
	}
	if input.ToolUseID != "toolu_full_456" {
		t.Errorf("ToolUseID = %q, want 'toolu_full_456'", input.ToolUseID)
	}
	if len(input.ToolInput) == 0 {
		t.Error("ToolInput should not be empty")
	}
}

func TestHookOutput_PreToolUse_CompleteOutput(t *testing.T) {
	// Test complete PreToolUse output
	output := &HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "allow",
			PermissionDecisionReason: "问题合理，可以向用户提问",
		},
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal PreToolUse output: %v", err)
	}

	// Unmarshal back to verify structure
	var decoded HookOutput
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.HookSpecificOutput == nil {
		t.Fatal("HookSpecificOutput should not be nil")
	}
	if decoded.HookSpecificOutput.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want 'PreToolUse'", decoded.HookSpecificOutput.HookEventName)
	}
	if decoded.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("PermissionDecision = %q, want 'allow'", decoded.HookSpecificOutput.PermissionDecision)
	}
	if decoded.HookSpecificOutput.PermissionDecisionReason != "问题合理，可以向用户提问" {
		t.Errorf("PermissionDecisionReason = %q, want '问题合理，可以向用户提问'", decoded.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestSupervisorResultToHookOutput_EventTypeCaseSensitivity(t *testing.T) {
	// Test that eventType matching is case-sensitive
	// Only exact "PreToolUse" should use PreToolUse format
	tests := []struct {
		name             string
		eventType        string
		expectPreToolUse bool
	}{
		{"exact match", "PreToolUse", true},
		{"lowercase", "pretooluse", false},
		{"uppercase", "PRETOOLUSE", false},
		{"mixed case", "Pretooluse", false},
		{"with spaces", "PreToolUse ", false},
	}

	result := &SupervisorResult{
		AllowStop: true,
		Feedback:  "测试反馈",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := SupervisorResultToHookOutput(result, tt.eventType)

			hasHookSpecific := output.HookSpecificOutput != nil
			if hasHookSpecific != tt.expectPreToolUse {
				t.Errorf("HookSpecificOutput presence = %v, want %v for eventType %q", hasHookSpecific, tt.expectPreToolUse, tt.eventType)
			}
		})
	}
}
