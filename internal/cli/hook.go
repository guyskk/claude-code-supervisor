// Package cli implements the supervisor-hook subcommand.
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
		// Output empty object to allow stop (no decision field = allow stop)
		output := HookOutput{}
		outputJSON, err := json.Marshal(output)
		if err != nil {
			return fmt.Errorf("failed to marshal hook output: %w", err)
		}
		fmt.Println(string(outputJSON))
		return nil
	}

	// Get supervisor ID: from environment variable
	supervisorID := os.Getenv("CCC_SUPERVISOR_ID")
	if supervisorID == "" {
		return fmt.Errorf("CCC_SUPERVISOR_ID is required from env var")
	}

	// Session-specific log file: supervisor-{id}.log
	stateDir, err := supervisor.GetStateDir()
	if err != nil {
		return fmt.Errorf("failed to get state directory: %w", err)
	}
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	logFilePath := filepath.Join(stateDir, fmt.Sprintf("supervisor-%s.log", supervisorID))
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open supervisor log file: %w", err)
	}
	defer logFile.Close()
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
	fmt.Fprintf(logFile, "\n%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(logFile, "[SUPERVISOR HOOK] Starting %s\n", supervisorID)
	logFile.Sync()

	var input StopHookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("failed to parse stdin JSON: %w", err)
	}
	sessionID := input.SessionID
	stopHookActive := input.StopHookActive
	if sessionID == "" {
		return fmt.Errorf("session_id is required from stdin")
	}
	fmt.Fprintf(logFile, "%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(logFile, "[%s] Session ID: %s\n", timestamp, sessionID)
	fmt.Fprintf(logFile, "[%s] Stop Hook Active: %v\n", timestamp, stopHookActive)
	fmt.Fprintf(logFile, "[%s] Args: %v\n", timestamp, args)
	logFile.Sync()

	// Log input as formatted JSON
	inputJSON, _ := json.MarshalIndent(input, "", "  ")
	fmt.Fprintf(logFile, "[%s] Input:\n%s\n\n", timestamp, string(inputJSON))
	logFile.Sync()

	// Check iteration count limit
	shouldContinue, count, err := supervisor.ShouldContinue(sessionID, supervisor.DefaultMaxIterations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking state: %v\n", err)
		// Continue anyway
	}
	if !shouldContinue {
		// Max iterations reached, allow stop
		fmt.Fprintf(os.Stderr, "\n[STOP] Max iterations (%d) reached, allowing stop\n", count)
		timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Max iterations (%d) reached, allowing stop\n", timestamp, count)
		logFile.Sync()
		return nil
	}

	// Increment count
	newCount, err := supervisor.IncrementCount(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to increment count: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Iteration count: %d/%d\n", newCount, supervisor.DefaultMaxIterations)
		timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Iteration count: %d/%d\n", timestamp, newCount, supervisor.DefaultMaxIterations)
		logFile.Sync()
	}

	// Get supervisor prompt
	supervisorPrompt, err := getSupervisorPrompt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read supervisor prompt: %v\n", err)
		supervisorPrompt = getDefaultSupervisorPrompt()
	}

	// Log the supervisor prompt
	timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
	fmt.Fprintf(logFile, "\n%s\n", strings.Repeat("-", 70))
	fmt.Fprintf(logFile, "[SUPERVISOR PROMPT]\n")
	fmt.Fprintf(logFile, "%s\n", strings.Repeat("-", 70))
	fmt.Fprintf(logFile, "%s\n", supervisorPrompt)
	fmt.Fprintf(logFile, "%s\n", strings.Repeat("-", 70))
	logFile.Sync()

	// Build claude command using --fork-session (not --print)
	// Note: NOT using --system-prompt - supervisor prompt is part of user prompt
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

	// Log the command being executed
	fmt.Fprintf(os.Stderr, "\n[SUPERVISOR] Reviewing work...\n")
	fmt.Fprintf(os.Stderr, "See log file for details: %s\n\n", logFilePath)

	timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
	fmt.Fprintf(logFile, "\n%s\n", strings.Repeat("-", 70))
	fmt.Fprintf(logFile, "[SUPERVISOR] Executing review\n")
	fmt.Fprintf(logFile, "%s\n", strings.Repeat("-", 70))
	fmt.Fprintf(logFile, "[%s] Command: %s\n", timestamp, args2Str)
	fmt.Fprintf(logFile, "[%s] Args: %d\n", timestamp, len(args2))
	logFile.Sync()

	// Execute command with CCC_SUPERVISOR_HOOK=1 to prevent infinite loop
	cmd := exec.Command(args2[0], args2[1:]...)
	cmd.Env = append(os.Environ(), "CCC_SUPERVISOR_HOOK=1")

	// Capture stdout for parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
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

	// Read stdout line by line using bufio.Scanner for proper line handling
	stdoutScanner := bufio.NewScanner(stdout)
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()

		// Write raw line to session log file
		fmt.Fprintf(logFile, "[%s] > %s\n", time.Now().Format("2006-01-02T15:04:05.000Z"), line)

		// Try to parse the line as JSON
		msg, parseErr := supervisor.ParseStreamJSONLine(line)
		if parseErr == nil && msg != nil {
			// Output text content to stderr
			if msg.Type == "text" && msg.Content != "" {
				fmt.Fprintf(os.Stderr, "%s\n", msg.Content)
			}
			// Extract structured output from result message
			if msg.Type == "result" && msg.StructuredOutput != nil {
				result = msg.StructuredOutput
				err := cmd.Process.Kill()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error cancelling command: %v\n", err)
				}
			}
		}
	}

	if scanErr := stdoutScanner.Err(); scanErr != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdout: %v\n", scanErr)
		timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Error reading stdout: %v\n", timestamp, scanErr)
		logFile.Sync()
	}

	// Wait for stderr goroutine to finish
	<-stderrDone

	// Wait for command to finish
	cmdErr := cmd.Wait()
	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "Claude command finished with error: %v\n", cmdErr)
		timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Claude command finished with error: %v\n", timestamp, cmdErr)
		if stderrContent.Len() > 0 {
			fmt.Fprintf(logFile, "[%s] Stderr: %s\n", timestamp, stderrContent.String())
		}
		logFile.Sync()
	} else {
		timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Claude command completed successfully\n", timestamp)
		logFile.Sync()
	}

	// Process result
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 60))
	timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
	fmt.Fprintf(logFile, "\n%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(logFile, "[RESULT] Review result\n")
	fmt.Fprintf(logFile, "%s\n", strings.Repeat("=", 70))

	if result == nil {
		// No result found, allow stop
		fmt.Fprintf(os.Stderr, "[RESULT] No supervisor result found, allowing stop\n")
		fmt.Fprintf(logFile, "[%s] No result found, allowing stop\n", timestamp)
		logFile.Sync()
		return nil
	}

	// Log the result as formatted JSON
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Fprintf(logFile, "[%s] Result:\n%s\n\n", timestamp, string(resultJSON))
	logFile.Sync()

	if result.Completed {
		// Task completed, allow stop
		fmt.Fprintf(os.Stderr, "[RESULT] Task completed, allowing stop\n")
		fmt.Fprintf(logFile, "[%s] Task completed, allowing stop\n", timestamp)
		logFile.Sync()
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
		return fmt.Errorf("failed to marshal hook output: %w", err)
	}
	fmt.Println(string(outputJSON))

	fmt.Fprintf(os.Stderr, "[RESULT] Task not completed\n")
	fmt.Fprintf(os.Stderr, "Feedback: %s\n", result.Feedback)
	fmt.Fprintf(os.Stderr, "Agent will continue working based on feedback\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("=", 60))

	fmt.Fprintf(logFile, "[%s] Blocking with feedback: %s\n", timestamp, result.Feedback)
	fmt.Fprintf(logFile, "[%s] Output:\n%s\n", timestamp, string(outputJSON))
	logFile.Sync()

	return nil
}

// getSupervisorPrompt reads the supervisor prompt from ~/.claude/SUPERVISOR.md
func getSupervisorPrompt() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	supervisorPath := filepath.Join(homeDir, ".claude", "SUPERVISOR.md")

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
