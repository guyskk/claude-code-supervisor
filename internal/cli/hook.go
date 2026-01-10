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
	"github.com/guyskk/ccc/internal/supervisor"
	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

//go:embed supervisor_prompt_default.md
var defaultPromptContent []byte

// StopHookInput represents the input from Stop hook.
type StopHookInput struct {
	SessionID      string `json:"session_id"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// SupervisorResult represents the parsed output from Supervisor.
type SupervisorResult struct {
	AllowStop bool   `json:"allow_stop"` // Whether to allow the Agent to stop
	Feedback  string `json:"feedback"`   // Feedback when AllowStop is false
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

// RunSupervisorHook executes the supervisor-hook subcommand.
func RunSupervisorHook(args []string) error {
	// Step 1: Get environment variables first
	isSupervisorMode := os.Getenv("CCC_SUPERVISOR") == "1"
	isSupervisorHook := os.Getenv("CCC_SUPERVISOR_HOOK") == "1"
	supervisorID := os.Getenv("CCC_SUPERVISOR_ID")

	// Step 2: Create logger as early as possible
	log := supervisor.NewSupervisorLogger(supervisorID)

	// Step 3: Log environment variables for debugging
	log.Debug("supervisor hook environment",
		"is_supervisor_mode", isSupervisorMode,
		"is_supervisor_hook", isSupervisorHook,
		"supervisor_id", supervisorID,
		"args", strings.Join(args, " "),
	)

	// Step 4: Check if this is a Supervisor's hook call (to avoid infinite loop):
	// - When NOT in supervisor mode (!CCC_SUPERVISOR=1), output empty JSON to allow stop
	// - When CCC_SUPERVISOR_HOOK=1 (called from supervisor itself), output empty JSON to allow stop
	if !isSupervisorMode || isSupervisorHook {
		return supervisor.OutputDecision(log, true, "not in supervisor mode or called from supervisor hook")
	}

	// Step 5: Validate supervisorID is present
	if supervisorID == "" {
		return fmt.Errorf("CCC_SUPERVISOR_ID is required from env var")
	}

	// Step 6: Load supervisor configuration
	supervisorCfg, err := config.LoadSupervisorConfig()
	if err != nil {
		return fmt.Errorf("failed to load supervisor config: %w", err)
	}

	// Step 7: Parse stdin input
	var input StopHookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("failed to parse stdin JSON: %w", err)
	}

	// Step 8: Validate and log input
	sessionID := input.SessionID
	if sessionID == "" {
		return fmt.Errorf("session_id is required from stdin")
	}

	inputJSON, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		log.Warn("failed to marshal hook input for logging", "error", err.Error())
	} else {
		log.Debug("hook input", "input", string(inputJSON))
	}

	// Step 9: Check iteration count limit using configured max_iterations
	maxIterations := supervisorCfg.MaxIterations
	shouldContinue, count, err := supervisor.ShouldContinue(sessionID, maxIterations)
	if err != nil {
		log.Warn("failed to check state", "error", err.Error())
	}
	if !shouldContinue {
		log.Info("max iterations reached, allowing stop",
			"count", count,
			"max", maxIterations,
		)
		return supervisor.OutputDecision(log, true, fmt.Sprintf("max iterations (%d/%d) reached", count, maxIterations))
	}

	// Step 10: Increment count
	newCount, err := supervisor.IncrementCount(sessionID)
	if err != nil {
		log.Warn("failed to increment count", "error", err.Error())
	} else {
		log.Info("iteration count",
			"count", newCount,
			"max", maxIterations,
		)
	}

	// Step 11: Get default supervisor prompt (hardcoded)
	supervisorPrompt := getDefaultSupervisorPrompt()
	log.Debug("supervisor prompt loaded",
		"prompt_length", len(supervisorPrompt),
	)

	// Step 12: Inform user about supervisor review
	logFilePath, err := supervisor.GetLogFilePath(supervisorID)
	if err != nil {
		log.Warn("failed to get log file path", "error", err.Error())
	}
	log.Info("starting supervisor review", "log_file", logFilePath)

	// Step 13: Run supervisor using Claude Agent SDK
	result, err := runSupervisorWithSDK(context.Background(), sessionID, supervisorPrompt, supervisorCfg.Timeout(), log)
	if err != nil {
		log.Error("supervisor SDK failed", "error", err.Error())
		return fmt.Errorf("supervisor SDK failed: %w", err)
	}

	log.Info("supervisor review completed")

	// Step 14: Output result based on AllowStop decision
	if result == nil {
		log.Info("no supervisor result found, allowing stop")
		return supervisor.OutputDecision(log, true, "no supervisor result found")
	}

	// Log the result (only once, in addition to raw message log)
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Warn("failed to marshal result for logging", "error", err.Error())
	} else {
		log.Info("supervisor result", "result", string(resultJSON))
	}

	if result.AllowStop {
		log.Info("work satisfactory, allowing stop")
		// Ensure feedback is not empty when allowing stop
		feedback := strings.TrimSpace(result.Feedback)
		if feedback == "" {
			feedback = "Work completed satisfactorily"
		}
		return supervisor.OutputDecision(log, true, feedback)
	}

	// Block with feedback
	log.Info("work not satisfactory, agent will continue", "feedback", result.Feedback)
	return supervisor.OutputDecision(log, false, result.Feedback)
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
		msgJSON, _ := json.Marshal(msg)
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
	log.Debug("parsing JSON from result field", "result_text", resultText)

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

// getDefaultSupervisorPrompt returns the default supervisor prompt.
// The prompt content is embedded from supervisor_prompt_default.md at compile time.
func getDefaultSupervisorPrompt() string {
	return string(defaultPromptContent)
}
