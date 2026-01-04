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

// runClaude executes the claude command for the given provider.
// If supervisorMode is true, it generates settings with Stop hook configuration.
// This replaces the current process with claude using syscall.Exec.
func runClaude(cfg *config.Config, providerName string, claudeArgs []string, supervisorMode bool) error {
	var mergedSettings map[string]interface{}
	var err error

	// Generate settings based on mode
	if supervisorMode {
		// Supervisor mode: generate settings with Stop hook
		if err := provider.SwitchWithHook(cfg, providerName); err != nil {
			return fmt.Errorf("error generating settings with hook: %w", err)
		}
		fmt.Printf("[Supervisor Mode enabled]\n")
		fmt.Printf("Launching with provider: %s\n", providerName)
		// Get merged settings for auth token
		mergedSettings = config.DeepMerge(cfg.Settings, cfg.Providers[providerName])
	} else {
		// Normal mode: switch provider
		mergedSettings, err = provider.Switch(cfg, providerName)
		if err != nil {
			return fmt.Errorf("error switching provider: %w", err)
		}
		fmt.Printf("Launching with provider: %s\n", providerName)
	}

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
	execArgs = append(execArgs, claudeArgs...)

	// Build environment variables
	authToken := provider.GetAuthToken(mergedSettings)
	env := append(os.Environ(), fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", authToken))

	// Execute the process (replaces current process, does not return on success)
	return executeProcess(claudePath, execArgs, env)
}
