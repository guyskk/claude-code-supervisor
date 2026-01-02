// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Supervisor manages the Agent-Supervisor automatic loop.
type Supervisor struct {
	settingsPath     string
	claudeArgs       []string
	completionMarker string
	sessionID        string
	userInputs       []string // Accumulate all user inputs
	agentOutputs     []string // Accumulate all agent outputs
}

// Config holds the configuration for Supervisor.
type Config struct {
	SettingsPath     string
	ClaudeArgs       []string
	CompletionMarker string
}

// New creates a new Supervisor instance.
func New(cfg *Config) *Supervisor {
	if cfg.CompletionMarker == "" {
		cfg.CompletionMarker = "[TASK_COMPLETED]"
	}

	return &Supervisor{
		settingsPath:     cfg.SettingsPath,
		claudeArgs:       cfg.ClaudeArgs,
		completionMarker: cfg.CompletionMarker,
		userInputs:       []string{},
		agentOutputs:     []string{},
	}
}

// Run starts the Supervisor loop.
func (s *Supervisor) Run() error {
	fmt.Println("Supervisor mode enabled")
	fmt.Println("Enter your task (press Ctrl+D when done):")

	// Read initial user input from stdin
	initialInput, err := s.readUserInput()
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if strings.TrimSpace(initialInput) == "" {
		return fmt.Errorf("empty input")
	}

	s.userInputs = append(s.userInputs, initialInput)

	return s.loop()
}

// readUserInput reads multi-line input from stdin until Ctrl+D.
func (s *Supervisor) readUserInput() (string, error) {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		fmt.Print("> ")
	}

	// Check for scanner errors (excluding EOF which is expected)
	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Ctrl+D pressed, join lines
	input := strings.Join(lines, "\n")
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("empty input")
	}

	return input, nil
}

// loop implements the main Agent-Supervisor loop.
func (s *Supervisor) loop() error {
	iteration := 0
	for {
		iteration++

		// Phase 1: Run Agent with accumulated input
		fmt.Printf("\n=== Agent Iteration %d ===\n", iteration)
		sessionID, output, err := s.runAgentIteration()
		if err != nil {
			return fmt.Errorf("agent phase failed: %w", err)
		}

		s.sessionID = sessionID
		s.agentOutputs = append(s.agentOutputs, output)

		// Phase 2: Run Supervisor check
		fmt.Printf("\n=== Supervisor Check ===\n")
		completed, feedback, err := s.runSupervisorCheck()
		if err != nil {
			return fmt.Errorf("supervisor check failed: %w", err)
		}

		// Phase 3: Check if task is completed
		if completed {
			fmt.Println("\n[Supervisor] Task completed!")
			break
		}

		// Phase 4: Feed feedback back to Agent for next iteration
		fmt.Printf("\n[Supervisor Feedback]\n%s\n", feedback)
		fmt.Println("\n[Continuing with feedback...]")

		// Add feedback as new user input for next iteration
		s.userInputs = append(s.userInputs, feedback)
	}

	// Final: Resume the original session for user interaction
	return s.resumeFinal()
}

// runAgentIteration runs one Agent iteration with current input.
func (s *Supervisor) runAgentIteration() (sessionID, output string, err error) {
	// Get the latest user input (or initial input for first iteration)
	input := s.userInputs[len(s.userInputs)-1]

	// Build claude command with --print mode
	args := []string{"claude", "--settings", s.settingsPath, "--print", "--output-format", "stream-json", "--verbose"}

	// Add initial args if provided
	if len(s.claudeArgs) > 0 {
		args = append(args, s.claudeArgs...)
	}

	// If resuming, add --resume flag
	if s.sessionID != "" {
		args = append(args, "--resume", s.sessionID)
	}

	// Create command
	cmd := exec.Command(args[0], args[1:]...)

	// Set input
	cmd.Stdin = strings.NewReader(input)

	// Execute and capture output (use CombinedOutput to see all output)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("agent execution failed: %w, output: %s", err, string(outputBytes))
	}

	output = string(outputBytes)

	// Show to user (may contain stream-json mixed with other output)
	fmt.Print(output)

	// Parse stream-json line by line to find session_id
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		msg, parseErr := ParseStreamJSONLine(line)
		if parseErr == nil && msg != nil && msg.SessionID != "" {
			sessionID = msg.SessionID
		}
	}

	return sessionID, output, nil
}

// runSupervisorCheck runs the Supervisor check phase.
func (s *Supervisor) runSupervisorCheck() (completed bool, feedback string, err error) {
	// Get Supervisor prompt
	supervisorPrompt, err := GetSupervisorPrompt()
	if err != nil {
		return false, "", fmt.Errorf("failed to get supervisor prompt: %w", err)
	}

	// Build context with all user inputs
	userInputContext := strings.Join(s.userInputs, "\n\n---\n\n")

	// Build the prompt for Supervisor
	checkPrompt := fmt.Sprintf("用户输入:\n%s\n\n请检查 Agent 的工作是否完成，是否满足用户需求。", userInputContext)

	// Build claude command for Supervisor
	args := []string{
		"claude",
		"--fork-session",
		"--resume", s.sessionID,
		"--system-prompt", supervisorPrompt,
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--",
		checkPrompt,
	}

	// Execute command and capture output
	// Use CombinedOutput to capture both stdout and stderr for better error debugging
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	// Parse stream-json output to extract actual text content
	// Even if command returns non-zero exit status, we may have valid output
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")

	// Extract text from result message (the actual supervisor response)
	var textContent strings.Builder
	for _, line := range lines {
		msg, parseErr := ParseStreamJSONLine(line)
		if parseErr == nil && msg != nil {
			// Get text from either content or result field
			var text string
			if msg.Type == "result" && msg.Result != "" {
				text = msg.Result
			} else if msg.Content != "" {
				text = msg.Content
			}

			if text != "" {
				// Check for completion marker
				if IsTaskCompleted(text, s.completionMarker) {
					completed = true
					// Remove marker from text
					text = strings.ReplaceAll(text, s.completionMarker, "")
				}
				// Accumulate text content
				if textContent.Len() > 0 {
					textContent.WriteString("\n")
				}
				textContent.WriteString(text)
			}
		}
	}

	feedback = strings.TrimSpace(textContent.String())

	// Only return error if we got no output at all
	if err != nil && feedback == "" {
		return false, "", fmt.Errorf("supervisor command failed: %w, output: %s", err, string(output))
	}

	// If we have feedback, return it even if command had non-zero exit status
	// (e.g., API errors still produce valid output in stream-json format)
	return completed, feedback, nil
}

// resumeFinal resumes the original session for user interaction.
func (s *Supervisor) resumeFinal() error {
	fmt.Println("\n=== Resuming session for interaction ===")

	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	args := []string{"claude", "--resume", s.sessionID, "--settings", s.settingsPath}

	// Run claude interactively
	cmd := exec.Command(claudePath, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command and wait for it to complete
	if err := cmd.Run(); err != nil {
		// Resume failed, but supervisor task is complete
		fmt.Printf("\nNote: Session resume ended: %v\n", err)
		fmt.Println("Supervisor mode finished successfully.")
		return nil
	}

	return nil
}
