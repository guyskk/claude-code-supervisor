// Package cli implements the supervisor-hook subcommand.
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/guyskk/ccc/internal/supervisor"
)

// StopHookInput represents the input from Stop hook.
type StopHookInput struct {
	SessionID      string `json:"session_id"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// HookOutput represents the output to stdout.
type HookOutput struct {
	Decision string `json:"decision"` // "block" or empty
	Reason   string `json:"reason,omitempty"`
}

// RunSupervisorHook executes the supervisor-hook subcommand.
func RunSupervisorHook(args []string) error {
	// Check if this is a Supervisor's hook call (to avoid infinite loop)
	// When CCC_SUPERVISOR_HOOK=1 is set, return fixed JSON to allow stop
	if os.Getenv("CCC_SUPERVISOR_HOOK") == "1" {
		output := HookOutput{Decision: "", Reason: ""}
		outputJSON, _ := json.Marshal(output)
		fmt.Println(string(outputJSON))
		return nil
	}

	// Get state directory using supervisor.GetStateDir() which checks CCC_WORK_DIR
	stateDir, err := supervisor.GetStateDir()
	if err != nil {
		return fmt.Errorf("failed to get state directory: %w", err)
	}

	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Open log file for writing
	logPath := filepath.Join(stateDir, "hook-invocation.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] supervisor-hook invoked\n", timestamp)
		fmt.Fprintf(logFile, "[%s] args: %v\n", timestamp, args)
		fmt.Fprintf(logFile, "[%s] stateDir: %s\n", timestamp, stateDir)
		logFile.Sync()
	}

	// Read stdin JSON
	var input StopHookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] ERROR: failed to parse stdin JSON: %v\n", timestamp, err)
			logFile.Sync()
		}
		return fmt.Errorf("failed to parse stdin JSON: %w", err)
	}

	if input.SessionID == "" {
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] ERROR: session_id is empty in input\n", timestamp)
			logFile.Sync()
		}
		return fmt.Errorf("session_id is required in input")
	}

	// Log the input
	if logFile != nil {
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
		inputJSON, _ := json.Marshal(input)
		fmt.Fprintf(logFile, "[%s] stdin input: %s\n", timestamp, string(inputJSON))
		fmt.Fprintf(logFile, "[%s] session_id: %s\n", timestamp, input.SessionID)
		fmt.Fprintf(logFile, "[%s] stop_hook_active: %v\n", timestamp, input.StopHookActive)
		logFile.Sync()
	}

	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] session_id: %s\n", input.SessionID)
	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] stop_hook_active: %v\n", input.StopHookActive)

	// Check iteration count limit
	shouldContinue, count, err := supervisor.ShouldContinue(input.SessionID, supervisor.DefaultMaxIterations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking state: %v\n", err)
		// Continue anyway
	}
	if !shouldContinue {
		// Max iterations reached, allow stop
		fmt.Fprintf(os.Stderr, "Supervisor: max iterations (%d) reached, allowing stop\n", count)
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Max iterations (%d) reached, allowing stop\n", timestamp, count)
			logFile.Sync()
		}
		return nil
	}

	// Increment count
	newCount, err := supervisor.IncrementCount(input.SessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to increment count: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Supervisor: iteration %d/%d\n", newCount, supervisor.DefaultMaxIterations)
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Iteration count: %d/%d\n", timestamp, newCount, supervisor.DefaultMaxIterations)
			logFile.Sync()
		}
	}

	// Get supervisor prompt
	supervisorPrompt, err := getSupervisorPrompt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read supervisor prompt: %v\n", err)
		supervisorPrompt = getDefaultSupervisorPrompt()
	}

	// JSON Schema for structured output
	jsonSchema := `{"type":"object","properties":{"completed":{"type":"boolean"},"feedback":{"type":"string"}},"required":["completed","feedback"]}`

	// Build user prompt: supervisor prompt + specific instruction
	supervisorUserPrompt := supervisorPrompt + "\n\n" + "请检查上面的对话，评估任务是否已完成。如果完成，返回 completed=true；如果未完成，返回 completed=false 并在 feedback 中说明需要继续做什么。"

	// Build claude command using --fork-session (not --print)
	// Note: NOT using --system-prompt - supervisor prompt is part of user prompt
	args2 := []string{
		"claude",
		"--fork-session", // Create child session instead of --print
		"--resume", input.SessionID,
		"--verbose", // Required for stream-json output format
		"--output-format", "stream-json",
		"--json-schema", jsonSchema,
		supervisorUserPrompt, // User prompt as positional argument (supervisor prompt + instruction)
	}

	// Log the command being executed
	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] Executing: claude --fork-session --resume %s ...\n", input.SessionID)
	if logFile != nil {
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Executing claude command with %d args\n", timestamp, len(args2))
		fmt.Fprintf(logFile, "[%s] Resume session: %s\n", timestamp, input.SessionID)
		logFile.Sync()
	}

	// Execute command with CCC_SUPERVISOR_HOOK=1 to prevent infinite loop
	cmd := exec.Command(args2[0], args2[1:]...)
	cmd.Env = append(os.Environ(), "CCC_SUPERVISOR_HOOK=1")

	// Capture stdout for parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude command: %w", err)
	}

	// Open output file for appending
	outputFile := filepath.Join(stateDir, fmt.Sprintf("supervisor-%s-output.jsonl", input.SessionID))
	outF, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open output file: %v\n", err)
		outF = nil
	}
	defer func() {
		if outF != nil {
			outF.Close()
		}
	}()

	// Read stdout and stderr concurrently
	var result *supervisor.SupervisorResult
	var stderrContent strings.Builder

	// Start goroutine to read stderr
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		stderrBuf := make([]byte, 4096)
		for {
			n, err := stderr.Read(stderrBuf)
			if n > 0 {
				content := string(stderrBuf[:n])
				stderrContent.WriteString(content)
				fmt.Fprintf(os.Stderr, "%s", content)
			}
			if err == io.EOF || err != nil {
				break
			}
		}
	}()

	// Read stdout line by line using bufio.Scanner for proper line handling
	stdoutScanner := bufio.NewScanner(stdout)
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()

		// Write raw line to output file
		if outF != nil {
			if _, writeErr := outF.WriteString(line + "\n"); writeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write output: %v\n", writeErr)
			}
		}

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
			}
		}
	}

	if scanErr := stdoutScanner.Err(); scanErr != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdout: %v\n", scanErr)
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Error reading stdout: %v\n", timestamp, scanErr)
			logFile.Sync()
		}
	}

	// Wait for stderr goroutine to finish
	<-stderrDone

	// Wait for command to finish
	cmdErr := cmd.Wait()
	if cmdErr != nil {
		fmt.Fprintf(os.Stderr, "Claude command finished with error: %v\n", cmdErr)
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Claude command finished with error: %v\n", timestamp, cmdErr)
			if stderrContent.Len() > 0 {
				fmt.Fprintf(logFile, "[%s] Stderr: %s\n", timestamp, stderrContent.String())
			}
			logFile.Sync()
		}
	} else {
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Claude command completed successfully\n", timestamp)
			logFile.Sync()
		}
	}

	// Process result
	if result == nil {
		// No result found, allow stop
		fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] No result found, allowing stop\n")
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] No result found, allowing stop\n", timestamp)
			logFile.Sync()
		}
		return nil
	}

	// Log the result
	if logFile != nil {
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
		resultJSON, _ := json.Marshal(result)
		fmt.Fprintf(logFile, "[%s] Supervisor result: %s\n", timestamp, string(resultJSON))
		logFile.Sync()
	}

	if result.Completed {
		// Task completed, allow stop
		fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] Task completed, allowing stop\n")
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Task completed, allowing stop\n", timestamp)
			logFile.Sync()
		}
		return nil
	}

	// Task not completed, block with feedback
	if result.Feedback == "" {
		result.Feedback = "请继续完成任务"
	}

	output := HookOutput{
		Decision: "block",
		Reason:   result.Feedback,
	}
	outputJSON, _ := json.Marshal(output)
	fmt.Println(string(outputJSON))

	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] Blocking with feedback: %s\n", result.Feedback)
	if logFile != nil {
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Blocking with feedback: %s\n", timestamp, result.Feedback)
		fmt.Fprintf(logFile, "[%s] Output: %s\n", timestamp, string(outputJSON))
		logFile.Sync()
	}

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
	return `# Claude Code Supervisor

你是一个严格的 Supervisor，负责审查 Agent 的工作质量，确保任务真正完成。

## 核心原则

**你的职责是确保 Agent 完成实际工作，而不是把问题抛给用户。**

## 审查要点

1. **Agent 是否完成了实际工作？** - 如果 Agent 只是在问问题而没有做事，返回 completed: false
2. **Agent 是否做了应该自己做的事？** - 运行测试、检查构建、创建 PR 等应该自己做，不应该问用户
3. **代码质量** - 检查是否有 bug、边界情况、未完成的 TODO
4. **任务完整性** - 用户的所有需求是否都已满足

## 判断标准

### completed: true
- Agent 完成了实际工作
- 测试已运行且通过
- 用户需求已满足

### completed: false
- Agent 在等待用户确认
- Agent 问了应该自己解决的问题（如"是否运行测试？"）
- 测试未运行或未通过
- 任务未完成

## Feedback 要求

当 completed: false 时，feedback 必须是具体的行动指令，不要给选项：
- ✓ "运行 go test ./... 验证代码正确性"
- ✓ "不要问用户是否创建 PR，直接创建"
- ✗ "建议运行测试"
- ✗ "可以考虑..."

## 输出格式

{"completed": boolean, "feedback": "string"}
`
}
