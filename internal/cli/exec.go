package cli

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/provider"
)

// executeProcess replaces the current process with the specified command.
// This uses syscall.Exec which does not return on success.
func executeProcess(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}

// runSupervisor runs claude in supervisor mode.
// It generates settings files with Stop hook configuration and executes claude.
func runSupervisor(cfg *config.Config, providerName string, claudeArgs []string) error {
	// Generate settings with hook
	settingsPath, supervisorSettingsPath, err := provider.SwitchWithHook(cfg, providerName)
	if err != nil {
		return fmt.Errorf("error generating settings with hook: %w", err)
	}

	fmt.Printf("Launching with provider: %s (Supervisor Mode)\n", providerName)
	fmt.Printf("Settings: %s\n", settingsPath)
	fmt.Printf("Supervisor Settings: %s\n", supervisorSettingsPath)

	// Find claude executable path
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Build arguments (argv[0] must be the program name)
	execArgs := []string{"claude", "--settings", settingsPath}
	if len(cfg.ClaudeArgs) > 0 {
		execArgs = append(execArgs, cfg.ClaudeArgs...)
	}
	execArgs = append(execArgs, claudeArgs...)

	// Build environment variables
	// Get auth token from the merged settings (we need to re-merge to get env)
	mergedSettings := config.DeepMerge(cfg.Settings, cfg.Providers[providerName])
	authToken := provider.GetAuthToken(mergedSettings)
	env := append(os.Environ(), fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", authToken))

	// Execute the process (replaces current process, does not return on success)
	return executeProcess(claudePath, execArgs, env)
}
