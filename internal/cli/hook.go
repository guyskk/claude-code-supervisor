// Package cli implements the supervisor-hook subcommand.
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/log"
	"github.com/guyskk/ccc/internal/supervisor"
)

// JSONSchema for structured output from Supervisor.
const supervisorJSONSchema = `{"type":"object","properties":{"completed":{"type":"boolean"},"feedback":{"type":"string"}},"required":["completed","feedback"]}`

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
		log.Debug("Supervisor hook detected (CCC_SUPERVISOR_HOOK=1), allowing stop")
		output := HookOutput{}
		outputJSON, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("failed to marshal hook output: %w", err)
		}
		fmt.Println(string(outputJSON))
		return nil
	}

	// Get supervisor ID from environment variable
	supervisorID := os.Getenv("CCC_SUPERVISOR_ID")
	if supervisorID == "" {
		return fmt.Errorf("CCC_SUPERVISOR_ID is required from env var")
	}

	// Create supervisor logger
	logger, err := log.NewSupervisorLogger(supervisorID)
	if err != nil {
		return fmt.Errorf("failed to create supervisor logger: %w", err)
	}
	defer logger.Close()

	// Parse stdin input
	var input StopHookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		logger.LogError("Failed to parse stdin JSON", err)
		return fmt.Errorf("failed to parse stdin JSON: %w", err)
	}

	sessionID := input.SessionID
	if sessionID == "" {
		logger.Error("Session ID is required from stdin")
		return fmt.Errorf("session_id is required from stdin")
	}

	// Log hook start
	logger.LogHookStart(sessionID, input.StopHookActive, args)

	// Log input JSON
	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	logger.LogInput(string(inputJSON))

	// Get max iterations from config
	cfg, cfgErr := config.Load()
	maxIterations := supervisor.DefaultMaxIterations
	if cfgErr == nil && cfg.Supervisor != nil && cfg.Supervisor.MaxIterations > 0 {
		maxIterations = cfg.Supervisor.MaxIterations
	}
	if cfgErr != nil {
		logger.Warn("Failed to load config, using default max iterations: %v", cfgErr)
	}

	// Check iteration count limit
	shouldContinue, count, err := supervisor.ShouldContinue(sessionID, maxIterations)
	if err != nil {
		logger.LogError("Error checking state", err)
		// Continue anyway
	}
	logger.LogIterationCheck(count, maxIterations, shouldContinue)

	if !shouldContinue {
		// Max iterations reached, allow stop
		logger.Warn("Max iterations (%d) reached, allowing stop", count)
		return nil
	}

	// Increment count
	newCount, err := supervisor.IncrementCount(sessionID)
	if err != nil {
		logger.LogError("Failed to increment count", err)
	} else {
		logger.Info("Iteration count: %d/%d", newCount, maxIterations)
	}

	// Get supervisor prompt
	supervisorPrompt, err := getSupervisorPrompt()
	if err != nil {
		logger.Warn("Failed to read supervisor prompt: %v, using default", err)
		supervisorPrompt = getDefaultSupervisorPrompt()
	}

	logger.Subsection("SUPERVISOR PROMPT")
	logger.Debug("%s", supervisorPrompt)

	// Build claude command
	args2 := []string{
		"claude",
		"-p",
		"--fork-session", // Create child session instead of --print
		"--resume", sessionID,
		"--verbose", // Required for stream-json output format
		"--output-format", "stream-json",
		"--json-schema", supervisorJSONSchema,
		supervisorPrompt, // User prompt as positional argument
	}
	args2Str := strings.Join(args2[:len(args2)-1], " ")

	logger.LogSupervisorCommand(args2Str, len(args2))
	fmt.Fprintf(os.Stderr, "\n[SUPERVISOR] Reviewing work...\n")
	fmt.Fprintf(os.Stderr, "See log file for details: %s\n\n", logger.FilePath())

	// Execute command with CCC_SUPERVISOR_HOOK=1 to prevent infinite loop
	cmd := exec.Command(args2[0], args2[1:]...)
	cmd.Env = append(os.Environ(), "CCC_SUPERVISOR_HOOK=1")

	// Capture stdout for parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.LogError("Failed to create stdout pipe", err)
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.LogError("Failed to create stderr pipe", err)
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		logger.LogError("Failed to start claude command", err)
		return fmt.Errorf("failed to start claude command: %w", err)
	}

	// Read stdout and stderr concurrently
	var result *supervisor.SupervisorResult
	var stderrContent strings.Builder

	// Start goroutine to read stderr
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		stderrBuf := make([]byte, 4096)
		for {
			n, err := stderrPipe.Read(stderrBuf)
			if n > 0 {
				content := string(stderrBuf[:n])
				stderrContent.WriteString(content)
				fmt.Fprintf(os.Stderr, "%s", content)
			}
			if err != nil {
				// Break on error (including EOF)
				break
			}
		}
	}()

	// Read stdout line by line
	stdoutScanner := bufio.NewScanner(stdout)
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()
		logger.LogStreamMessage(line)

		// Try to parse the line as JSON
		msg, parseErr := supervisor.ParseStreamJSONLine(line)
		if parseErr == nil && msg != nil {
			// Output text content to stderr
			if msg.Type == "text" && msg.Content != "" {
				logger.LogTextContent(msg.Content)
				fmt.Fprintf(os.Stderr, "%s\n", msg.Content)
			}
			// Extract structured output from result message
			if msg.Type == "result" && msg.StructuredOutput != nil {
				result = msg.StructuredOutput
				// Kill the process as we got the result
				err := cmd.Process.Kill()
				if err != nil {
					logger.Warn("Failed to kill command after getting result: %v", err)
				} else {
					logger.Debug("Command killed after receiving structured result")
				}
			}
		}
	}

	if scanErr := stdoutScanner.Err(); scanErr != nil {
		logger.LogError("Error reading stdout", scanErr)
	}

	// Wait for stderr goroutine to finish
	<-stderrDone

	// Wait for command to finish
	cmdErr := cmd.Wait()
	logger.LogCommandComplete(cmdErr)

	if cmdErr != nil && stderrContent.Len() > 0 {
		logger.KeyValue("Stderr", stderrContent.String())
	}

	// Process result
	if result == nil {
		logger.Warn("No supervisor result found, allowing stop")
		fmt.Fprintf(os.Stderr, "[RESULT] No supervisor result found, allowing stop\n")
		return nil
	}

	// Log the result
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	logger.LogSupervisorResult(string(resultJSON), result.Completed, result.Feedback)

	if result.Completed {
		// Task completed, allow stop
		fmt.Fprintf(os.Stderr, "[RESULT] Task completed, allowing stop\n")
		logger.LogHookDecision("allow stop", "Task completed")
		return nil
	}

	// Task not completed, block with feedback
	if result.Feedback == "" {
		result.Feedback = "Please continue completing the task"
	}

	output := HookOutput{
		Decision: "block",
		Reason:   result.Feedback,
	}
	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		logger.LogError("Failed to marshal hook output", err)
		return fmt.Errorf("failed to marshal hook output: %w", err)
	}
	fmt.Println(string(outputJSON))

	logger.LogHookDecision("block", result.Feedback)

	fmt.Fprintf(os.Stderr, "\n[RESULT] Task not completed\n")
	fmt.Fprintf(os.Stderr, "Feedback: %s\n", result.Feedback)
	fmt.Fprintf(os.Stderr, "Agent will continue working based on feedback\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("=", 60))

	return nil
}

// getSupervisorPrompt reads the supervisor prompt from ~/.claude/SUPERVISOR.md
func getSupervisorPrompt() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	supervisorPath := strings.Join([]string{homeDir, ".claude", "SUPERVISOR.md"}, string(os.PathSeparator))

	data, err := os.ReadFile(supervisorPath)
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
