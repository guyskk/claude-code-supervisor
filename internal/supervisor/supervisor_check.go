// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// SupervisorCheckResult represents the result of a Supervisor check.
type SupervisorCheckResult struct {
	Completed bool
	Feedback  string
	Error     error
}

// RunSupervisorCheck runs a Supervisor check using fork-session.
// It takes the session ID to fork and the user input context.
func RunSupervisorCheck(sessionID, userInput, completionMarker string) *SupervisorCheckResult {
	// Get Supervisor prompt
	supervisorPrompt, err := GetSupervisorPrompt()
	if err != nil {
		return &SupervisorCheckResult{
			Error: fmt.Errorf("failed to get supervisor prompt: %w", err),
		}
	}

	// Build the prompt for Supervisor
	// Include user input context
	checkPrompt := fmt.Sprintf("用户原始提问:\n%s\n\n请检查 Agent 的工作是否完成，是否满足用户需求。", userInput)

	// Build claude command for Supervisor
	args := []string{
		"claude",
		"--fork-session",
		"--resume", sessionID,
		"--system-prompt", supervisorPrompt,
		"--print",
		"--output-format", "stream-json",
		"--",
		checkPrompt,
	}

	// Execute command and capture output
	cmd := exec.Command(args[0], args[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &SupervisorCheckResult{
			Error: fmt.Errorf("supervisor command failed: %w, stderr: %s", err, stderr.String()),
		}
	}

	// Parse output
	output := stdout.String()

	// Check for completion marker
	completed := IsTaskCompleted(output, completionMarker)

	// Extract feedback (remove completion marker if present)
	feedback := strings.TrimSpace(output)
	if completed {
		// Remove the completion marker from feedback
		feedback = strings.ReplaceAll(feedback, completionMarker, "")
		feedback = strings.TrimSpace(feedback)
	}

	return &SupervisorCheckResult{
		Completed: completed,
		Feedback:  feedback,
	}
}
