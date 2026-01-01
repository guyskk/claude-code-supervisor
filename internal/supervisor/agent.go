// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/creack/pty"
)

// AgentSession manages an Agent session via pty.
type AgentSession struct {
	settingsPath string
	args         []string
	ptyFile      *os.File
	cmd          *exec.Cmd
	sessionID    string
	userInput    strings.Builder
	captured     strings.Builder
}

// AgentResult represents the result of an Agent session.
type AgentResult struct {
	SessionID string
	UserInput string
	Error     error
}

// StartAgent starts a new Agent session using pty.
func StartAgent(settingsPath string, claudeArgs []string) (*AgentSession, error) {
	// Build claude command
	args := []string{"claude", "--settings", settingsPath, "--print", "--output-format", "stream-json"}
	args = append(args, claudeArgs...)

	cmd := exec.Command(args[0], args[1:]...)

	// Start pty
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	return &AgentSession{
		settingsPath: settingsPath,
		args:         claudeArgs,
		ptyFile:      ptyFile,
		cmd:          cmd,
	}, nil
}

// Run runs the Agent session until it stops (waits for user input).
// It returns the session ID and captured user input.
func (a *AgentSession) Run() (*AgentResult, error) {
	// Read output line by line
	buf := make([]byte, 4096)
	for {
		n, err := a.ptyFile.Read(buf)
		if n > 0 {
			line := string(buf[:n])

			// Write to stdout (user sees Agent output)
			fmt.Print(line)

			// Capture for parsing
			a.captured.Write(buf[:n])

			// Try to parse stream-json
			msg, parseErr := ParseStreamJSONLine(line)
			if parseErr == nil && msg != nil {
				// Extract session ID
				if msg.SessionID != "" {
					a.sessionID = msg.SessionID
				}

				// Check if Agent is waiting for input
				if DetectAgentWaiting(msg) {
					// Agent stopped, return result
					return &AgentResult{
						SessionID: a.sessionID,
						UserInput: a.userInput.String(),
					}, nil
				}
			}
		}

		if err != nil {
			// Check if process exited
			if a.cmd.Process != nil {
				// Try to get exit status
				if exitErr, ok := err.(*os.PathError); ok {
					if exitErr.Err == syscall.EIO {
						// EIO means pty closed, which is expected when Agent stops
						break
					}
				}
			}

			// Check if we have session ID (we can proceed even with error)
			if a.sessionID != "" {
				return &AgentResult{
					SessionID: a.sessionID,
					UserInput: a.userInput.String(),
				}, nil
			}

			return nil, fmt.Errorf("agent session error: %w", err)
		}
	}

	return &AgentResult{
		SessionID: a.sessionID,
		UserInput: a.userInput.String(),
	}, nil
}

// WriteInput writes user input to the Agent pty.
func (a *AgentSession) WriteInput(input string) error {
	a.userInput.WriteString(input + "\n")
	_, err := a.ptyFile.WriteString(input + "\n")
	return err
}

// Close closes the pty and waits for the process to end.
func (a *AgentSession) Close() error {
	if a.ptyFile != nil {
		a.ptyFile.Close()
	}
	if a.cmd.Process != nil {
		a.cmd.Process.Wait()
	}
	return nil
}

// ResumeAgent resumes an existing Agent session.
func ResumeAgent(sessionID, settingsPath string, feedback string) error {
	// Build claude command to resume
	args := []string{"claude", "--resume", sessionID, "--settings", settingsPath}

	// Replace current process with claude
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Build environment
	env := os.Environ()

	// Use syscall.Exec to replace current process
	return syscall.Exec(claudePath, args, env)
}
