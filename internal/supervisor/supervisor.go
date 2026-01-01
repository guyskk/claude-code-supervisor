// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"fmt"
)

// Supervisor manages the Agent-Supervisor automatic loop.
type Supervisor struct {
	settingsPath       string
	claudeArgs         []string
	completionMarker   string
	sessionID          string
	userInput          string
	supervisorFeedback string
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
	}
}

// Run starts the Supervisor loop.
func (s *Supervisor) Run() error {
	fmt.Println("Supervisor mode enabled")

	return s.loop()
}

// loop implements the main Agent-Supervisor loop.
func (s *Supervisor) loop() error {
	for {
		// Phase 1: Run Agent
		sessionID, userInput, err := s.runAgent()
		if err != nil {
			return fmt.Errorf("agent phase failed: %w", err)
		}

		s.sessionID = sessionID

		// If this is a feedback iteration, replace userInput
		if s.supervisorFeedback != "" {
			s.userInput = s.supervisorFeedback
		} else {
			s.userInput = userInput
		}

		// Phase 2: Run Supervisor check
		completed, feedback, err := s.runSupervisorCheck()
		if err != nil {
			return fmt.Errorf("supervisor check failed: %w", err)
		}

		// Phase 3: Check if task is completed
		if completed {
			fmt.Println("\n[Supervisor] Task completed")
			break
		}

		// Phase 4: Feed feedback back to Agent
		s.supervisorFeedback = feedback
		fmt.Printf("\n[Supervisor Feedback]\n%s\n", feedback)
		fmt.Println("\n[Continuing with feedback...]")
	}

	// Final: Resume the original session
	return s.resumeFinal()
}

// runAgent runs the Agent phase.
func (s *Supervisor) runAgent() (sessionID, userInput string, err error) {
	// Start Agent session
	agent, err := StartAgent(s.settingsPath, s.claudeArgs)
	if err != nil {
		return "", "", err
	}
	defer agent.Close()

	// Run Agent until it stops
	result, err := agent.Run()
	if err != nil {
		return "", "", err
	}

	return result.SessionID, result.UserInput, nil
}

// runSupervisorCheck runs the Supervisor check phase.
func (s *Supervisor) runSupervisorCheck() (completed bool, feedback string, err error) {
	fmt.Println("\n[Supervisor] Checking work quality...")

	result := RunSupervisorCheck(s.sessionID, s.userInput, s.completionMarker)
	if result.Error != nil {
		return false, "", result.Error
	}

	return result.Completed, result.Feedback, nil
}

// resumeFinal resumes the original session for user interaction.
func (s *Supervisor) resumeFinal() error {
	return ResumeAgent(s.sessionID, s.settingsPath, "")
}
