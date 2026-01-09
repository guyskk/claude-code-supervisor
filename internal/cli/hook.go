// Package cli implements the supervisor-hook subcommand.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/logger"
	"github.com/guyskk/ccc/internal/llmparser"
	"github.com/guyskk/ccc/internal/supervisor"
	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

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

// SupervisorResult represents the structured output from Supervisor.
type SupervisorResult struct {
	Completed bool   `json:"completed"`
	Feedback  string `json:"feedback"`
}

// supervisorResultSchema is the JSON schema for supervisor structured output.
var supervisorResultSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"completed": map[string]interface{}{
			"type":        "boolean",
			"description": "Whether the task has been completed to the best possible state with nothing left to do",
		},
		"feedback": map[string]interface{}{
			"type":        "string",
			"description": "Specific feedback and guidance for continuing work when completed is false",
		},
	},
	"required": []string{"completed", "feedback"},
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

	if result.Completed {
		fileLogger.Info("task completed, allowing stop")
		fmt.Fprintf(os.Stderr, "[RESULT] Task completed, allowing stop\n")
		return nil
	}

	// Task not completed, block with feedback
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

	fmt.Fprintf(os.Stderr, "[RESULT] Task not completed\n")
	fmt.Fprintf(os.Stderr, "Feedback: %s\n", feedback)
	fmt.Fprintf(os.Stderr, "Agent will continue working based on feedback\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("=", 60))

	fileLogger.Info("blocking with feedback", logger.StringField("feedback", feedback))

	return nil
}

// runSupervisorWithSDK runs the supervisor using the Claude Agent SDK with structured output.
// The supervisor prompt is sent as a USER message, allowing SDK to load system prompts from settings.
func runSupervisorWithSDK(ctx context.Context, sessionID, prompt string, timeout time.Duration, log logger.Logger) (*SupervisorResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build options for SDK
	// - ForkSession: Create a fork to review the current session state
	// - Resume: Load the session context (includes system/user/project prompts from settings)
	// - SettingSources: Load system prompts from user, project, and local settings
	// - OutputFormat: Use structured output with JSON schema for supervisor result
	opts := types.NewClaudeAgentOptions().
		WithForkSession(true).                                                                            // Fork the current session
		WithResume(sessionID).                                                                            // Resume from specific session
		WithSettingSources(types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal). // Load all setting sources
		WithOutputFormat(supervisorResultSchema)                                                          // Enable structured output with JSON schema

	// Set environment variable to avoid infinite loop
	opts.Env = map[string]string{
		"CCC_SUPERVISOR_HOOK": "1",
	}

	log.Debug("SDK options",
		logger.StringField("fork_session", "true"),
		logger.StringField("resume", sessionID),
		logger.StringField("structured_output", "enabled"),
	)

	// Send supervisor prompt as USER message
	// The prompt contains the review instructions, system prompts are loaded from settings
	log.Debug("sending supervisor review request as user message")
	messages, err := claude.Query(ctx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK query: %w", err)
	}

	// Process messages and collect result
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
				logger.StringField("has_structured_output", fmt.Sprintf("%v", m.StructuredOutput != nil)),
			)
		case *types.SystemMessage:
			log.Debug("system message",
				logger.StringField("subtype", m.Subtype),
			)
		case *types.UserMessage:
			log.Debug("user message (echo)")
		}
	}

	// Extract structured output from ResultMessage
	if resultMessage == nil {
		log.Warn("no result message received from SDK")
		return nil, fmt.Errorf("no result message received from SDK")
	}

	// Try structured output first (preferred path)
	if resultMessage.StructuredOutput != nil {
		result, err := parseStructuredOutput(resultMessage.StructuredOutput)
		if err != nil {
			log.Error("failed to parse structured output", logger.StringField("error", err.Error()))
			return nil, fmt.Errorf("failed to parse structured output: %w", err)
		}
		log.Info("successfully parsed supervisor result from structured output")
		return result, nil
	}

	// Fallback: try parsing from Result field (for backward compatibility)
	log.Warn("no structured output in result message, falling back to result field")
	if resultMessage.Result != nil {
		result, err := parseResultFromMap(*resultMessage.Result)
		if err == nil {
			log.Info("successfully parsed result from ResultMessage.Result field")
			return result, nil
		}
		log.Warn("failed to parse ResultMessage.Result", logger.StringField("error", err.Error()))
	}

	return nil, fmt.Errorf("no structured output in result message and failed to parse Result field")
}

// parseStructuredOutput parses the structured output from the SDK into a SupervisorResult.
func parseStructuredOutput(output interface{}) (*SupervisorResult, error) {
	return parseResultFromMap(output)
}

// parseResultFromMap parses a map[string]interface{} into a SupervisorResult.
// This is used by both parseStructuredOutput and parseResultJSON.
func parseResultFromMap(data interface{}) (*SupervisorResult, error) {
	outputMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("parsed result is not a map, got %T", data)
	}

	result := &SupervisorResult{}

	// Extract completed field (boolean)
	if completed, ok := outputMap["completed"].(bool); ok {
		result.Completed = completed
	} else {
		return nil, fmt.Errorf("missing or invalid 'completed' field (expected bool)")
	}

	// Extract feedback field (string)
	if feedback, ok := outputMap["feedback"].(string); ok {
		result.Feedback = feedback
	} else {
		return nil, fmt.Errorf("missing or invalid 'feedback' field (expected string)")
	}

	return result, nil
}

// parseResultJSON parses the JSON text into a SupervisorResult.
// This is a fallback function when structured output is not available.
// It uses the llmparser package for fault-tolerant JSON parsing.
func parseResultJSON(jsonText string) (*SupervisorResult, error) {
	// Use llmparser for fault-tolerant JSON parsing with schema validation
	parsed, err := llmparser.Parse(jsonText, supervisorResultSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return parseResultFromMap(parsed)
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
func getDefaultSupervisorPrompt() string {
	return `# 任务：严格审查当前执行的工作并给出反馈意见

你是一个无比严格的 Supervisor，负责审查当前执行的工作，判断任务是否真正完成，达到了能做到的最好的、最完备状态，没有任何还能做的事情了。

## 核心原则

**你的职责是检查是否完成了实际工作，检查是否把能做/该做的事情都做好了，无可挑剔，而不是把问题抛给用户，交付不够好的结果。**
**你的任务是审查工作，而不是执行未完成的任务，如果发现任务存在任何问题，你应该给出反馈意见，不需要亲自执行。**

## 审查要点

1. **是否完成了实际工作？** - 如果只是在问问题而没有做事，返回 completed: false
2. **是否做了应该自己做的事？** - 运行测试、检查构建、创建PR、Review PR等应该自己做，不应该问用户
3. **代码质量** - 检查是否有 bug、边界情况、未完成的 TODO
4. **任务完整性** - 用户的所有需求是否都已满足
5. **无可挑剔** - 是否达到了能做到的最好的、最完备状态，没有任何还能做的事情了

## 判断标准

### completed: true
- Agent 完成了实际工作
- 测试已运行且通过
- 用户需求已满足
- 把能做/该做的事情都做好了，无可挑剔

### completed: false
- Agent 在等待用户确认
- Agent 问了应该自己解决的问题（如"是否运行测试？"）
- 测试未运行或未通过
- 任务未完成
- 存在任何还能做的事情，可以把任务完成的更好

## Feedback 要求

当 completed: false 时，feedback 必须是具体的反馈建议，用于指导继续完成工作。

## 输出格式

你的回复必须符合指定的 JSON Schema 格式，包含以下字段：
- completed (boolean): 任务是否已完成
- feedback (string): 当任务未完成时的具体反馈建议

请仔细回顾用户需求和方案规划，充分阅读所有的改动以及相关文档/代码等，严格检查评估当前任务的情况。
提交结果后立即停止，不需要再做任何其他工作。`
}
