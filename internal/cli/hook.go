// Package cli implements the supervisor-hook subcommand.
package cli

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/llmparser"
	"github.com/guyskk/ccc/internal/logger"
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

// HookOutput represents the output to stdout.
// When Decision is empty, the decision field is omitted from JSON to allow stop.
type HookOutput struct {
	Decision string `json:"decision,omitempty"` // "block" or omitted (allows stop)
	Reason   string `json:"reason,omitempty"`
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
	// Check if this is a Supervisor's hook call (to avoid infinite loop):
	// - When NOT in supervisor mode (!CCC_SUPERVISOR=1), output empty JSON to allow stop
	// - When CCC_SUPERVISOR_HOOK=1 (called from supervisor itself), output empty JSON to allow stop
	isSupervisorMode := os.Getenv("CCC_SUPERVISOR") == "1"
	isSupervisorHook := os.Getenv("CCC_SUPERVISOR_HOOK") == "1"
	if !isSupervisorMode || isSupervisorHook {
		output := HookOutput{}
		outputJSON, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("failed to marshal hook output: %w", err)
		}
		fmt.Println(string(outputJSON))
		return nil
	}

	// Load supervisor configuration
	supervisorCfg, err := config.LoadSupervisorConfig()
	if err != nil {
		return fmt.Errorf("failed to load supervisor config: %w", err)
	}

	// Get supervisor ID from environment variable
	supervisorID := os.Getenv("CCC_SUPERVISOR_ID")
	if supervisorID == "" {
		return fmt.Errorf("CCC_SUPERVISOR_ID is required from env var")
	}

	// Setup session-specific log file
	stateDir, err := supervisor.GetStateDir()
	if err != nil {
		return fmt.Errorf("failed to get state directory: %w", err)
	}
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	logFilePath := filepath.Join(stateDir, fmt.Sprintf("supervisor-%s.log", supervisorID))

	// Create file logger with debug level (supervisor needs detailed logging)
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open supervisor log file: %w", err)
	}
	defer logFile.Close()

	fileLogger := logger.NewLogger(logFile, logger.LevelDebug).With(
		logger.StringField("supervisor_id", supervisorID),
	)

	fileLogger.Info("supervisor hook started",
		logger.StringField("args", strings.Join(args, " ")),
	)

	// Parse stdin input
	var input StopHookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("failed to parse stdin JSON: %w", err)
	}
	sessionID := input.SessionID
	if sessionID == "" {
		return fmt.Errorf("session_id is required from stdin")
	}

	// Log input
	inputJSON, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		fileLogger.Warn("failed to marshal hook input for logging", logger.StringField("error", err.Error()))
	} else {
		fileLogger.Debug("hook input", logger.StringField("input", string(inputJSON)))
	}

	// Check iteration count limit using configured max_iterations
	maxIterations := supervisorCfg.MaxIterations
	shouldContinue, count, err := supervisor.ShouldContinue(sessionID, maxIterations)
	if err != nil {
		fileLogger.Warn("failed to check state", logger.StringField("error", err.Error()))
	}
	if !shouldContinue {
		fileLogger.Warn("max iterations reached",
			logger.IntField("count", count),
			logger.IntField("max", maxIterations),
		)
		fmt.Fprintf(os.Stderr, "\n[STOP] Max iterations (%d) reached, allowing stop\n", count)
		return nil
	}

	// Increment count
	newCount, err := supervisor.IncrementCount(sessionID)
	if err != nil {
		fileLogger.Warn("failed to increment count", logger.StringField("error", err.Error()))
	} else {
		fileLogger.Info("iteration count",
			logger.IntField("count", newCount),
			logger.IntField("max", maxIterations),
		)
		fmt.Fprintf(os.Stderr, "Iteration count: %d/%d\n", newCount, maxIterations)
	}

	// Get default supervisor prompt (hardcoded)
	supervisorPrompt := getDefaultSupervisorPrompt()

	fileLogger.Debug("supervisor prompt loaded",
		logger.IntField("prompt_length", len(supervisorPrompt)),
	)

	// Inform user
	fmt.Fprintf(os.Stderr, "\n[SUPERVISOR] Reviewing work...\n")
	fmt.Fprintf(os.Stderr, "See log file for details: %s\n\n", logFilePath)

	fileLogger.Info("starting supervisor review")

	// Run supervisor using Claude Agent SDK
	result, err := runSupervisorWithSDK(context.Background(), sessionID, supervisorPrompt, supervisorCfg.Timeout(), fileLogger)
	if err != nil {
		fileLogger.Error("supervisor SDK failed", logger.StringField("error", err.Error()))
		return fmt.Errorf("supervisor SDK failed: %w", err)
	}

	fileLogger.Info("supervisor review completed")

	// Process result
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 60))

	if result == nil {
		fileLogger.Warn("no supervisor result found, allowing stop")
		fmt.Fprintf(os.Stderr, "[RESULT] No supervisor result found, allowing stop\n")
		return nil
	}

	// Log the result
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fileLogger.Warn("failed to marshal result for logging", logger.StringField("error", err.Error()))
	} else {
		fileLogger.Info("supervisor result", logger.StringField("result", string(resultJSON)))
	}

	if result.AllowStop {
		fileLogger.Info("work satisfactory, allowing stop")
		fmt.Fprintf(os.Stderr, "[RESULT] Work satisfactory, allowing stop\n")
		return nil
	}

	// Work not satisfactory, block with feedback
	// Use default feedback if empty (after trimming whitespace)
	feedback := strings.TrimSpace(result.Feedback)
	if feedback == "" {
		feedback = "Please continue completing the task"
	}

	output := HookOutput{
		Decision: "block",
		Reason:   feedback,
	}
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal hook output: %w", err)
	}
	fmt.Println(string(outputJSON))

	fmt.Fprintf(os.Stderr, "[RESULT] Work not satisfactory\n")
	fmt.Fprintf(os.Stderr, "Feedback: %s\n", feedback)
	fmt.Fprintf(os.Stderr, "Agent will continue working based on feedback\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("=", 60))

	fileLogger.Info("blocking with feedback", logger.StringField("feedback", feedback))

	return nil
}

// runSupervisorWithSDK runs the supervisor using the Claude Agent SDK.
// The supervisor prompt is sent as a USER message, and we parse the Result field for JSON output.
func runSupervisorWithSDK(ctx context.Context, sessionID, prompt string, timeout time.Duration, log logger.Logger) (*SupervisorResult, error) {
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
		logger.StringField("fork_session", "true"),
		logger.StringField("resume", sessionID),
	)

	// Send supervisor prompt as USER message
	log.Debug("sending supervisor review request as user message")
	messages, err := claude.Query(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK query: %w", err)
	}

	// Process messages and get ResultMessage
	log.Debug("receiving messages from SDK")

	var resultMessage *types.ResultMessage

	for msg := range messages {
		// Log raw message JSON for debugging
		msgJSON, _ := json.Marshal(msg)
		log.Debug("raw message", logger.StringField("json", string(msgJSON)))

		switch m := msg.(type) {
		case *types.AssistantMessage:
			// Log assistant messages for debugging
			for _, block := range m.Content {
				if textBlock, ok := block.(*types.TextBlock); ok {
					log.Debug("assistant text block",
						logger.StringField("text", textBlock.Text),
					)
				}
			}
		case *types.ResultMessage:
			resultMessage = m
			log.Debug("result message",
				logger.StringField("subtype", m.Subtype),
				logger.StringField("result", safeString(m.Result)),
				logger.StringField("cost", fmt.Sprintf("%.4f", float64Value(m.TotalCostUSD))),
			)
		case *types.SystemMessage:
			log.Debug("system message",
				logger.StringField("subtype", m.Subtype),
			)
		case *types.UserMessage:
			log.Debug("user message (echo)")
		}
	}

	// Extract and parse result from ResultMessage
	if resultMessage == nil {
		log.Warn("no result message received from SDK")
		return nil, fmt.Errorf("no result message received from SDK")
	}

	if resultMessage.Result == nil {
		log.Warn("result message has no Result field")
		return nil, fmt.Errorf("result message has no Result field")
	}

	// Parse JSON from Result field using llmparser
	resultText := *resultMessage.Result
	log.Debug("parsing JSON from result field", logger.StringField("result_text", resultText))

	result, err := parseResultJSON(resultText)
	if err != nil {
		log.Error("failed to parse result JSON", logger.StringField("error", err.Error()))
		return nil, fmt.Errorf("failed to parse result JSON: %w", err)
	}

	log.Info("successfully parsed supervisor result")
	return result, nil
}

// parseResultJSON parses the JSON text into a SupervisorResult.
// It uses the llmparser package for fault-tolerant JSON parsing.
func parseResultJSON(jsonText string) (*SupervisorResult, error) {
	// Use llmparser for fault-tolerant JSON parsing with schema validation
	parsed, err := llmparser.Parse(jsonText, supervisorResultSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert parsed interface{} to SupervisorResult
	outputMap, ok := parsed.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parsed result is not a map, got %T", parsed)
	}

	result := &SupervisorResult{}

	// Extract allow_stop field (boolean)
	if allowStop, ok := outputMap["allow_stop"].(bool); ok {
		result.AllowStop = allowStop
	} else {
		return nil, fmt.Errorf("missing or invalid 'allow_stop' field (expected bool)")
	}

	// Extract feedback field (string)
	if feedback, ok := outputMap["feedback"].(string); ok {
		result.Feedback = feedback
	} else {
		return nil, fmt.Errorf("missing or invalid 'feedback' field (expected string)")
	}

	return result, nil
}

// float64Value safely dereferences a float64 pointer.
func float64Value(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// safeString safely dereferences a string pointer.
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// getDefaultSupervisorPrompt returns the default supervisor prompt.
// The prompt content is embedded from supervisor_prompt_default.md at compile time.
func getDefaultSupervisorPrompt() string {
	return string(defaultPromptContent)
}
