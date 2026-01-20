// Package cli implements the supervisor-hook subcommand.
package cli

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/llmparser"
	"github.com/guyskk/ccc/internal/prettyjson"
	"github.com/guyskk/ccc/internal/supervisor"
	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

//go:embed supervisor_prompt_default.md
var defaultPromptContent []byte

// HookInput represents the input from any Claude Code hook event.
// It supports all hook event types including Stop and PreToolUse.
type HookInput struct {
	// Common fields (all event types)
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	CWD            string `json:"cwd,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	HookEventName  string `json:"hook_event_name,omitempty"` // "Stop", "PreToolUse", etc.

	// Stop event fields
	StopHookActive bool `json:"stop_hook_active,omitempty"`

	// PreToolUse event fields
	ToolName  string          `json:"tool_name,omitempty"`   // e.g., "AskUserQuestion"
	ToolInput json.RawMessage `json:"tool_input,omitempty"`  // Tool-specific input
	ToolUseID string          `json:"tool_use_id,omitempty"` // Tool call ID
}

// StopHookInput is an alias for HookInput to maintain backward compatibility.
type StopHookInput = HookInput

// SupervisorResult represents the parsed output from Supervisor.
type SupervisorResult struct {
	AllowStop bool   `json:"allow_stop"` // Whether to allow the Agent to stop
	Feedback  string `json:"feedback"`   // Feedback when AllowStop is false
}

// HookOutput represents the output to any Claude Code hook event.
// The format varies based on the event type (Stop vs PreToolUse).
type HookOutput struct {
	// Stop event fields
	Decision *string `json:"decision,omitempty"` // "block" or omitted (allows stop)
	Reason   string  `json:"reason,omitempty"`   // Feedback message

	// PreToolUse event fields
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// HookSpecificOutput represents the PreToolUse hook specific output.
type HookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`            // "PreToolUse"
	PermissionDecision       string `json:"permissionDecision"`       // "allow", "deny", "ask"
	PermissionDecisionReason string `json:"permissionDecisionReason"` // Decision reason
}

// supervisorResultSchema is the JSON schema for parsing supervisor output.
var supervisorResultSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"allow_stop": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether to allow the Agent to stop working (true = work is satisfactory, false = needs more work)",
		},
		"feedback": map[string]interface{}{
			"type":        "string",
			"description": "Specific feedback and guidance for continuing work when allow_stop is false",
		},
	},
	"required": []string{"allow_stop", "feedback"},
}

// SupervisorResultToHookOutput converts a SupervisorResult to HookOutput based on event type.
// For PreToolUse events, it returns permissionDecision (allow/deny).
// For Stop events (default), it returns decision (block/allow).
func SupervisorResultToHookOutput(result *SupervisorResult, eventType string) *HookOutput {
	if eventType == "PreToolUse" {
		decision := "allow"
		if !result.AllowStop {
			decision = "deny"
		}
		return &HookOutput{
			HookSpecificOutput: &HookSpecificOutput{
				HookEventName:            "PreToolUse",
				PermissionDecision:       decision,
				PermissionDecisionReason: result.Feedback,
			},
		}
	}

	// Stop event (default)
	if !result.AllowStop {
		block := "block"
		return &HookOutput{
			Decision: &block,
			Reason:   result.Feedback,
		}
	}

	// Allow stop
	return &HookOutput{
		Reason: result.Feedback,
	}
}

func logCurrentEnv(log *slog.Logger) {
	// Log environment variables for debugging
	lines := []string{}
	// Add all environment variables starting with CLAUDE_, ANTHROPIC_, CCC_
	prefixes := []string{"CLAUDE_", "ANTHROPIC_", "CCC_"}
	for _, env := range os.Environ() {
		for _, prefix := range prefixes {
			if strings.HasPrefix(env, prefix) {
				lines = append(lines, env)
				break
			}
		}
	}
	envStr := strings.Join(lines, "\n")
	log.Debug(fmt.Sprintf("supervisor hook environment:\n%s", envStr))
}

// RunSupervisorHook executes the supervisor-hook subcommand.
func RunSupervisorHook(opts *SupervisorHookCommand) error {
	// Validate supervisorID is present
	supervisorID := os.Getenv("CCC_SUPERVISOR_ID")
	if supervisorID == "" {
		return fmt.Errorf("CCC_SUPERVISOR_ID is required from env var")
	}

	// Create logger as early as possible
	log := supervisor.NewSupervisorLogger(supervisorID)
	logCurrentEnv(log)

	isSupervisorHook := os.Getenv("CCC_SUPERVISOR_HOOK") == "1"
	if isSupervisorHook {
		return supervisor.OutputDecision(log, true, "called from supervisor hook")
	}

	// Load state to check if supervisor mode is enabled
	state, err := supervisor.LoadState(supervisorID)
	if err != nil {
		return fmt.Errorf("failed to load supervisor state: %w", err)
	}

	// Check if supervisor mode is enabled
	if !state.Enabled {
		log.Debug("supervisor mode disabled, allowing stop", "enabled", state.Enabled)
		return supervisor.OutputDecision(log, true, "supervisor mode disabled")
	}

	// Load supervisor configuration
	supervisorCfg, err := config.LoadSupervisorConfig()
	if err != nil {
		return fmt.Errorf("failed to load supervisor config: %w", err)
	}

	// Get sessionID from command line argument or stdin
	var sessionID string
	if opts != nil && opts.SessionId != "" {
		// Use sessionID from command line argument
		sessionID = opts.SessionId
		log.Debug("using session_id from command line argument", "session_id", sessionID)
	}
	var input HookInput
	var eventType string // Track the event type (Stop, PreToolUse, etc.)
	if sessionID == "" {
		// Parse stdin input
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(&input); err != nil {
			return fmt.Errorf("failed to parse stdin JSON: %w", err)
		}
		sessionID = input.SessionID
		// Identify event type from hook_event_name field
		eventType = input.HookEventName
		if eventType == "" {
			eventType = "Stop" // Default to Stop event if not specified
		}
		// Log input
		inputJSON, err := json.MarshalIndent(input, "", "  ")
		if err != nil {
			log.Warn("failed to marshal hook input", "error", err.Error())
		} else {
			log.Debug("hook input", "event_type", eventType, "input", string(inputJSON))
		}
	} else {
		// When sessionID comes from command line, default to Stop event
		eventType = "Stop"
	}

	// Validate sessionID is present
	if sessionID == "" {
		return fmt.Errorf("session_id is required (either from --session-id argument or stdin)")
	}

	// Check iteration count limit using configured max_iterations
	maxIterations := supervisorCfg.MaxIterations
	shouldContinue, count, err := supervisor.ShouldContinue(sessionID, maxIterations)
	if err != nil {
		log.Warn("failed to check supervisor state", "error", err.Error())
	}
	if !shouldContinue {
		log.Info("max iterations reached, allowing stop",
			"count", count,
			"max", maxIterations,
		)
		return supervisor.OutputDecision(log, true, fmt.Sprintf("max iterations (%d/%d) reached", count, maxIterations))
	}

	// Increment count
	newCount, err := supervisor.IncrementCount(sessionID)
	if err != nil {
		log.Warn("failed to increment count", "error", err.Error())
	} else {
		log.Info("iteration count",
			"count", newCount,
			"max", maxIterations,
		)
	}

	// Get default supervisor prompt
	supervisorPrompt, promptSource := getDefaultSupervisorPrompt()
	log.Debug("supervisor prompt loaded",
		"source", promptSource,
		"length", len(supervisorPrompt),
	)

	// Inform user about supervisor review
	log.Info("starting supervisor review", "session_id", sessionID)

	// Run supervisor using Claude Agent SDK
	result, err := runSupervisorWithSDK(context.Background(), sessionID, supervisorPrompt, supervisorCfg.Timeout(), log)
	if err != nil {
		log.Error("supervisor SDK failed", "error", err.Error())
		return fmt.Errorf("supervisor SDK failed: %w", err)
	}

	log.Info("supervisor review completed")

	// Output result based on AllowStop decision
	if result == nil {
		log.Info("no supervisor result found, allowing operation")
		// For PreToolUse events, allow the tool call
		// For Stop events, allow stopping
		output := SupervisorResultToHookOutput(&SupervisorResult{AllowStop: true, Feedback: "no supervisor result found"}, eventType)
		return outputHookOutput(output, log)
	}

	// Convert SupervisorResult to HookOutput based on event type
	output := SupervisorResultToHookOutput(result, eventType)

	// Log the decision based on event type
	if eventType == "PreToolUse" {
		if result.AllowStop {
			log.Info("supervisor output: allow tool call", "feedback", result.Feedback)
		} else {
			log.Info("supervisor output: deny tool call", "feedback", result.Feedback)
		}
	} else {
		if result.AllowStop {
			log.Info("supervisor output: allow stop", "feedback", result.Feedback)
		} else {
			log.Info("supervisor output: not allow stop", "feedback", result.Feedback)
		}
	}

	return outputHookOutput(output, log)
}

// outputHookOutput outputs the HookOutput as JSON to stdout.
func outputHookOutput(output *HookOutput, log *slog.Logger) error {
	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hook output: %w", err)
	}
	fmt.Println(string(outputJSON))
	return nil
}

// runSupervisorWithSDK runs the supervisor using the Claude Agent SDK.
// The supervisor prompt is sent as a USER message, and we parse the Result field for JSON output.
func runSupervisorWithSDK(ctx context.Context, sessionID, prompt string, timeout time.Duration, log *slog.Logger) (*SupervisorResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build options for SDK
	// - ForkSession: Create a fork to review the current session state
	// - Resume: Load the session context (includes system/user/project prompts from settings)
	// - SettingSources: Load system prompts from user, project, and local settings
	// NOTE: We do NOT use WithOutputFormat - we parse the Result field directly
	// [Bug] StructuredOutput tool doesn't stop agent execution - agent continues after calling it
	// https://github.com/anthropics/claude-code/issues/17125
	opts := types.NewClaudeAgentOptions().
		WithForkSession(true).                                                                            // Fork the current session
		WithResume(sessionID).                                                                            // Resume from specific session
		WithSettingSources(types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal) // Load all setting sources

	// Set environment variable to avoid infinite loop
	opts.Env["CCC_SUPERVISOR_HOOK"] = "1"

	log.Debug("SDK options",
		"fork_session", "true",
		"resume", sessionID,
	)

	// Create interactive client
	log.Debug("creating SDK client")
	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK client: %w", err)
	}
	defer client.Close(ctx)

	// Connect to Claude
	log.Debug("connecting SDK client")
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect SDK client: %w", err)
	}

	// Send supervisor prompt as USER message
	log.Debug("sending supervisor review request as user message")
	if err := client.Query(ctx, prompt); err != nil {
		return nil, fmt.Errorf("failed to send query: %w", err)
	}

	// Process messages and get ResultMessage
	log.Debug("receiving messages from SDK")

	var resultMessage *types.ResultMessage

	for msg := range client.ReceiveResponse(ctx) {
		// Log raw message JSON for debugging (this is the ONE place where all messages are logged)
		msgJSON, err := prettyjson.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message: %w", err)
		}
		log.Debug("raw message", "json", string(msgJSON))

		switch m := msg.(type) {
		case *types.ResultMessage:
			resultMessage = m
		}
	}

	// Extract and parse result from ResultMessage
	if resultMessage == nil {
		log.Error("no result message received from SDK")
		return nil, fmt.Errorf("no result message received from SDK")
	}

	if resultMessage.Result == nil {
		log.Error("result message has no Result field")
		return nil, fmt.Errorf("result message has no Result field")
	}

	// Parse JSON from Result field using llmparser
	resultText := *resultMessage.Result
	result := parseResultJSON(resultText)

	return result, nil
}

// parseResultJSON parses the JSON text into a SupervisorResult.
// It uses the llmparser package for fault-tolerant JSON parsing.
// When parsing fails, it returns a fallback result with allow_stop=false
// and the original text as feedback, allowing the agent to continue working.
func parseResultJSON(jsonText string) *SupervisorResult {
	// Use llmparser for fault-tolerant JSON parsing with schema validation
	parsed, err := llmparser.Parse(jsonText, supervisorResultSchema)
	if err != nil {
		// Fallback: parsing failed, use original text as feedback
		// This allows the agent to continue working instead of failing
		fallbackText := strings.TrimSpace(jsonText)
		if fallbackText == "" {
			fallbackText = "请继续完成任务"
		}
		return &SupervisorResult{
			AllowStop: false,
			Feedback:  fallbackText,
		}
	}

	// Convert parsed interface{} to SupervisorResult
	outputMap, ok := parsed.(map[string]interface{})
	if !ok {
		// Fallback: wrong type
		return &SupervisorResult{
			AllowStop: false,
			Feedback:  strings.TrimSpace(jsonText),
		}
	}

	result := &SupervisorResult{}

	// Extract allow_stop field (boolean)
	if allowStop, ok := outputMap["allow_stop"].(bool); ok {
		result.AllowStop = allowStop
	} else {
		// Fallback: missing or invalid allow_stop field
		return &SupervisorResult{
			AllowStop: false,
			Feedback:  strings.TrimSpace(jsonText),
		}
	}

	// Extract feedback field (string)
	if feedback, ok := outputMap["feedback"].(string); ok {
		result.Feedback = feedback
	} else {
		// Fallback: missing or invalid feedback field
		result.Feedback = strings.TrimSpace(jsonText)
	}

	return result
}

// getDefaultSupervisorPrompt returns the supervisor prompt and its source.
// It first tries to read from ~/.claude/SUPERVISOR.md (or CCC_CONFIG_DIR/SUPERVISOR.md).
// If the custom file exists and has content, it is used; otherwise, the default embedded prompt is returned.
// The source return value indicates where the prompt came from:
// - "supervisor_prompt_default" for the embedded default prompt
// - Full file path for a custom SUPERVISOR.md file
func getDefaultSupervisorPrompt() (string, string) {
	customPromptPath := config.GetDir() + "/SUPERVISOR.md"
	data, err := os.ReadFile(customPromptPath)
	if err == nil {
		prompt := strings.TrimSpace(string(data))
		if prompt != "" {
			return prompt, customPromptPath
		}
	}
	return string(defaultPromptContent), "supervisor_prompt_default"
}
