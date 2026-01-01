// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetSupervisorPrompt reads the Supervisor prompt from project or global location.
// It first tries ./SUPERVISOR.md in the current directory,
// then falls back to ~/.claude/SUPERVISOR.md.
func GetSupervisorPrompt() (string, error) {
	// Try project-level SUPERVISOR.md first
	if content, err := os.ReadFile("./SUPERVISOR.md"); err == nil {
		return string(content), nil
	}

	// Fallback to global SUPERVISOR.md
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	globalPath := filepath.Join(homeDir, ".claude", "SUPERVISOR.md")
	content, err := os.ReadFile(globalPath)
	if err != nil {
		return "", fmt.Errorf("no supervisor prompt found (tried ./SUPERVISOR.md and %s)", globalPath)
	}

	return string(content), nil
}
