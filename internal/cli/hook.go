// Package cli implements the supervisor-hook subcommand.
package cli

import (
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
	// Parse arguments
	settingsPath := ""
	stateDir := ""

	for i, arg := range args {
		if arg == "--settings" && i+1 < len(args) {
			settingsPath = args[i+1]
		} else if arg == "--state-dir" && i+1 < len(args) {
			stateDir = args[i+1]
		}
	}

	if settingsPath == "" {
		return fmt.Errorf("--settings parameter is required")
	}

	// Default state dir
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, ".claude", "ccc")
	}

	// Ensure state directory exists and write initial log
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write initial log that hook was called
	logPath := filepath.Join(stateDir, "hook-invocation.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] supervisor-hook invoked\n", timestamp)
		fmt.Fprintf(logFile, "[%s] args: %v\n", timestamp, args)
		fmt.Fprintf(logFile, "[%s] settingsPath: %s\n", timestamp, settingsPath)
		fmt.Fprintf(logFile, "[%s] stateDir: %s\n", timestamp, stateDir)
		logFile.Sync()
	}

	// Also print to stderr for immediate visibility
	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] Called at %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] settingsPath: %s\n", settingsPath)
	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] stateDir: %s\n", stateDir)
	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] logPath: %s\n", logPath)

	// Read stdin JSON
	var input StopHookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		// Log the error
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
		// Log to file
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
		// Log to file
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Iteration count: %d/%d\n", timestamp, newCount, supervisor.DefaultMaxIterations)
			logFile.Sync()
		}
	}

	// Build supervisor claude command
	supervisorPrompt, err := getSupervisorPrompt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read supervisor prompt: %v\n", err)
		supervisorPrompt = getDefaultSupervisorPrompt()
	}

	// JSON Schema for structured output
	jsonSchema := `{"type":"object","properties":{"completed":{"type":"boolean"},"feedback":{"type":"string"}},"required":["completed","feedback"]}`

	// Build claude command
	args2 := []string{
		"claude",
		"--settings", settingsPath,
		"--fork-session",
		"--resume", input.SessionID,
		"--output-format", "stream-json",
		"--json-schema", jsonSchema,
		"--system-prompt", supervisorPrompt,
	}

	// Log the command being executed
	fmt.Fprintf(os.Stderr, "[ccc supervisor-hook] Executing: claude --settings %s --fork-session --resume %s --output-format stream-json --json-schema <schema> --system-prompt <prompt>\n", settingsPath, input.SessionID)
	if logFile != nil {
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
		fmt.Fprintf(logFile, "[%s] Executing claude command with %d args\n", timestamp, len(args2))
		fmt.Fprintf(logFile, "[%s] Fork session: %s\n", timestamp, input.SessionID)
		logFile.Sync()
	}

	// Execute command
	cmd := exec.Command(args2[0], args2[1:]...)

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

	// Read and process stream-json output
	var result *supervisor.SupervisorResult
	stdoutBuf := make([]byte, 4096)
	for {
		n, err := stdout.Read(stdoutBuf)
		if n > 0 {
			data := stdoutBuf[:n]
			// Write to output file
			if outF != nil {
				if _, writeErr := outF.Write(data); writeErr != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to write output: %v\n", writeErr)
				}
			}

			// Parse line by line
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				msg, parseErr := supervisor.ParseStreamJSONLine(line)
				if parseErr == nil && msg != nil {
					// Output text to stderr
					if msg.Type == "text" && msg.Content != "" {
						fmt.Fprintf(os.Stderr, "%s\n", msg.Content)
					}
					// Parse result
					if msg.Type == "result" && msg.Result != "" {
						sr, parseErr := supervisor.ParseSupervisorResult(msg.Result)
						if parseErr == nil {
							result = sr
						}
					}
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdout: %v\n", err)
			break
		}
	}

	// Copy stderr to our stderr
	stderrBuf := make([]byte, 4096)
	for {
		n, err := stderr.Read(stderrBuf)
		if n > 0 {
			fmt.Fprintf(os.Stderr, "%s", string(stderrBuf[:n]))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Claude command finished with error: %v\n", err)
		// Log to file
		if logFile != nil {
			timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
			fmt.Fprintf(logFile, "[%s] Claude command finished with error: %v\n", timestamp, err)
			logFile.Sync()
		}
	} else {
		// Log successful completion
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
	return "你是 Supervisor，负责检查 Agent 的工作质量。\n\n" +
		"## 输出格式要求\n\n" +
		"你必须严格按照以下 JSON Schema 返回结果：\n\n" +
		"```json\n" +
		"{\n" +
		"  \"type\": \"object\",\n" +
		"  \"properties\": {\n" +
		"    \"completed\": {\n" +
		"      \"type\": \"boolean\",\n" +
		"      \"description\": \"任务是否已完成\"\n" +
		"    },\n" +
		"    \"feedback\": {\n" +
		"      \"type\": \"string\",\n" +
		"      \"description\": \"当 completed 为 false 时，提供具体的反馈和改进建议\"\n" +
		"    }\n" +
		"  },\n" +
		"  \"required\": [\"completed\", \"feedback\"]\n" +
		"}\n" +
		"```\n\n" +
		"- 如果任务完成，设置 \"completed\": true，feedback 可以为空\n" +
		"- 如果任务未完成，设置 \"completed\": false，feedback 必须包含具体的反馈\n"
}
