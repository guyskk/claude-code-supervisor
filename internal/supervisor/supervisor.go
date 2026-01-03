// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║           Supervisor Mode - Agent Auto-Loop              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// Keep prompting until we get non-empty input
	for {
		initialInput, err := s.readUserInput()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// If input is cancelled, return
		if strings.TrimSpace(initialInput) == "" {
			continue
		}

		s.userInputs = append(s.userInputs, initialInput)
		break
	}

	return s.loop()
}

// readUserInput reads user input using huh.
func (s *Supervisor) readUserInput() (string, error) {
	// Show prompt before form
	fmt.Print("> ")

	var input string

	// Create a minimal theme with no borders
	theme := huh.ThemeCharm()
	theme.Group.Base = lipgloss.NewStyle()
	theme.Group.Title = lipgloss.NewStyle()
	theme.Blurred.Base = lipgloss.NewStyle()
	theme.Focused.Base = lipgloss.NewStyle()

	inputField := huh.NewInput().
		Title("").
		Value(&input)

	err := huh.NewForm(
		huh.NewGroup(inputField),
	).WithTheme(theme).Run()

	if err != nil {
		if err == huh.ErrUserAborted {
			return "", fmt.Errorf("input cancelled")
		}
		return "", err
	}

	input = strings.TrimSpace(input)
	// Echo the input back (only if not empty)
	if input != "" {
		fmt.Printf("\n> %s\n\n", input)
	} else {
		fmt.Println()
	}

	return input, nil
}

// loop implements the main Agent-Supervisor loop.
func (s *Supervisor) loop() error {
	iteration := 0
	for {
		iteration++

		// Phase 1: Run Agent with accumulated input
		printSectionHeader("🤖", fmt.Sprintf("Agent Iteration %d", iteration), "cyan")
		sessionID, output, err := s.runAgentIteration()
		if err != nil {
			return fmt.Errorf("agent phase failed: %w", err)
		}

		s.sessionID = sessionID
		s.agentOutputs = append(s.agentOutputs, output)

		// Phase 2: Run Supervisor check
		printSectionHeader("👁️", "Supervisor Check", "yellow")
		completed, feedback, err := s.runSupervisorCheck()
		if err != nil {
			return fmt.Errorf("supervisor check failed: %w", err)
		}

		// Phase 3: Check if task is completed
		if completed {
			printSuccess("Task completed successfully!")
			break
		}

		// Phase 4: Feed feedback back to Agent for next iteration
		printSectionHeader("💬", "Supervisor Feedback", "magenta")
		fmt.Printf("%s\n\n", feedback)
		printInfo("Continuing with feedback...")

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
	// Must include --settings to ensure fork session has correct API configuration
	args := []string{
		"claude",
		"--settings", s.settingsPath,
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
	printSectionHeader("🔄", "Resuming Session", "green")

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
		fmt.Printf("\n")
		printInfo(fmt.Sprintf("Session resume ended: %v", err))
		printSuccess("Supervisor mode finished successfully.")
		return nil
	}

	return nil
}

// printSectionHeader prints a formatted section header with emoji and color.
func printSectionHeader(emoji, title, color string) {
	colors := map[string]string{
		"cyan":    "\033[36m",
		"yellow":  "\033[33m",
		"green":   "\033[32m",
		"magenta": "\033[35m",
		"blue":    "\033[34m",
		"red":     "\033[31m",
	}
	reset := "\033[0m"
	c := colors[color]
	if c == "" {
		c = "\033[0m"
	}

	fmt.Printf("\n%s%s %s %s%s\n", c, emoji, title, strings.Repeat("─", 60-len(title)), reset)
}

// printSuccess prints a success message.
func printSuccess(msg string) {
	fmt.Printf("\033[32m✓ %s\033[0m\n", msg)
}

// printInfo prints an info message.
func printInfo(msg string) {
	fmt.Printf("\033[36mℹ %s\033[0m\n", msg)
}

// printError prints an error message.
func printError(msg string) {
	fmt.Printf("\033[31m✗ %s\033[0m\n", msg)
}
