// Package cli implements the supervisor-hook subcommand.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guyskk/ccc/internal/claude_agent_sdk"
	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/logger"
	"github.com/guyskk/ccc/internal/supervisor"
)

const (
	// supervisorJSONSchema is the JSON schema for structured output from Supervisor.
	supervisorJSONSchema = `{"type":"object","properties":{"completed":{"type":"boolean"},"feedback":{"type":"string"}},"required":["completed","feedback"]}`
)

// StopHookInput represents the input from Stop hook.
type StopHookInput struct {
	SessionID      string `json:"session_id"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// HookOutput represents the output to stdout.
// When Decision is nil/empty, the decision field is omitted to allow stop.
type HookOutput struct {
	Decision string `json:"decision,omitempty"` // "block" or omitted (allows stop)
	Reason   string `json:"reason,omitempty"`
}

// RunSupervisorHook executes the supervisor-hook subcommand.
func RunSupervisorHook(args []string) error {
	// Check if this is a Supervisor's hook call (to avoid infinite loop)
	// When CCC_SUPERVISOR_HOOK=1 is set, output empty JSON to allow stop
	if os.Getenv("CCC_SUPERVISOR_HOOK") == "1" {
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

	// Create logger
	logLevel := logger.LevelInfo
	switch supervisorCfg.LogLevel {
	case "debug":
		logLevel = logger.LevelDebug
	case "info":
		logLevel = logger.LevelInfo
	case "warn":
		logLevel = logger.LevelWarn
	case "error":
		logLevel = logger.LevelError
	}

	// Get supervisor ID: from environment variable
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

	// Create file logger
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open supervisor log file: %w", err)
	}
	defer logFile.Close()

	fileLogger := logger.NewLogger(logFile, logLevel).With(
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
	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	fileLogger.Debug("hook input", logger.StringField("input", string(inputJSON)))

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

	// Get supervisor prompt
	supervisorPrompt, err := getSupervisorPrompt(supervisorCfg)
	if err != nil {
		fileLogger.Warn("failed to read supervisor prompt", logger.StringField("error", err.Error()))
		supervisorPrompt = getDefaultSupervisorPrompt()
	}

	fileLogger.Debug("supervisor prompt loaded",
		logger.IntField("prompt_length", len(supervisorPrompt)),
	)

	// Create claude agent
	agent, err := claude_agent_sdk.NewAgent(&claude_agent_sdk.Config{
		Timeout: supervisorCfg.Timeout(),
		Logger:  fileLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to create claude agent: %w", err)
	}

	// Inform user
	fmt.Fprintf(os.Stderr, "\n[SUPERVISOR] Reviewing work...\n")
	fmt.Fprintf(os.Stderr, "See log file for details: %s\n\n", logFilePath)

	fileLogger.Info("starting supervisor review")

	// Run supervisor via claude agent
	opts := claude_agent_sdk.RunOptions{
		SessionID:    sessionID,
		ForkSession:  true,
		Prompt:       supervisorPrompt,
		OutputFormat: "stream-json",
		JSONSchema:   supervisorJSONSchema,
		Env:          []string{"CCC_SUPERVISOR_HOOK=1"},
	}

	result, err := agent.Run(nil, opts)
	if err != nil {
		fileLogger.Error("claude agent run failed", logger.StringField("error", err.Error()))
		return fmt.Errorf("claude agent run failed: %w", err)
	}

	fileLogger.Info("supervisor review completed",
		logger.StringField("duration", result.Duration.String()),
		logger.IntField("success", func() int {
			if result.Success {
				return 1
			} else {
				return 0
			}
		}()),
	)

	// Extract structured output
	var supervisorResult *claude_agent_sdk.SupervisorResult
	if result.StructuredOutput != nil {
		supervisorResult, err = claude_agent_sdk.ParseSupervisorResult(result.StructuredOutput)
		if err != nil {
			fileLogger.Warn("failed to parse supervisor result",
				logger.StringField("error", err.Error()),
			)
		}
	}

	// Process result
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 60))

	if supervisorResult == nil {
		fileLogger.Warn("no supervisor result found, allowing stop")
		fmt.Fprintf(os.Stderr, "[RESULT] No supervisor result found, allowing stop\n")
		return nil
	}

	// Log the result
	resultJSON, _ := json.MarshalIndent(supervisorResult, "", "  ")
	fileLogger.Info("supervisor result", logger.StringField("result", string(resultJSON)))

	if supervisorResult.Completed {
		fileLogger.Info("task completed, allowing stop")
		fmt.Fprintf(os.Stderr, "[RESULT] Task completed, allowing stop\n")
		return nil
	}

	// Task not completed, block with feedback
	if supervisorResult.Feedback == "" {
		supervisorResult.Feedback = "Please continue completing the task"
	}

	output := HookOutput{
		Decision: "block",
		Reason:   supervisorResult.Feedback,
	}
	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hook output: %w", err)
	}
	fmt.Println(string(outputJSON))

	fmt.Fprintf(os.Stderr, "[RESULT] Task not completed\n")
	fmt.Fprintf(os.Stderr, "Feedback: %s\n", supervisorResult.Feedback)
	fmt.Fprintf(os.Stderr, "Agent will continue working based on feedback\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("=", 60))

	fileLogger.Info("blocking with feedback", logger.StringField("feedback", supervisorResult.Feedback))

	return nil
}

// getSupervisorPrompt reads the supervisor prompt from the configured path.
func getSupervisorPrompt(cfg *config.SupervisorConfig) (string, error) {
	promptPath, err := cfg.GetResolvedPromptPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(promptPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
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

调用StructuredOutput工具提供JSON结果:
{"completed": boolean, "feedback": "string"}

请仔细回顾用户需求和方案规划，充分阅读所有的改动以及相关文档/代码等，严格检查评估当前任务的情况。
调用StructuredOutput工具成功提交反馈后立即停止，不需要再做任何其他工作。
`
}
