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

// runClaude executes the claude command with the given settings.
// This replaces the current process with claude using syscall.Exec.
func runClaude(cfg *config.Config, settings map[string]interface{}, args []string) error {
	// Find claude executable path
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH: %w", err)
	}

	// Build arguments (argv[0] must be the program name)
	execArgs := []string{"claude"}
	if len(cfg.ClaudeArgs) > 0 {
		execArgs = append(execArgs, cfg.ClaudeArgs...)
	}
	execArgs = append(execArgs, args...)

	// Build environment variables
	authToken := provider.GetAuthToken(settings)
	env := append(os.Environ(), fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", authToken))

	// Execute the process (replaces current process, does not return on success)
	return executeProcess(claudePath, execArgs, env)
}

// runSupervisor runs claude in supervisor mode.
// It generates settings.json with Stop hook configuration and executes claude.
// The provider name is obtained from cfg.CurrentProvider.
func runSupervisor(cfg *config.Config, claudeArgs []string) error {
	// Get provider name from config
	providerName := cfg.CurrentProvider
	if providerName == "" {
		return fmt.Errorf("no current provider set")
	}

	// Generate settings with hook
	if err := provider.SwitchWithHook(cfg, providerName); err != nil {
		return fmt.Errorf("error generating settings with hook: %w", err)
	}

	// Show supervisor mode is enabled
	fmt.Printf("[Supervisor Mode enabled]\n")
	fmt.Printf("Launching with provider: %s\n", providerName)

	// Get merged settings for auth token
	mergedSettings := config.DeepMerge(cfg.Settings, cfg.Providers[providerName])

	// Execute claude with the settings
	return runClaude(cfg, mergedSettings, claudeArgs)
}
