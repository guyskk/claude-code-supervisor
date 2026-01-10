package cli

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/provider"
	"github.com/guyskk/ccc/internal/supervisor"
)

// executeProcess replaces the current process with the specified command.
// This uses syscall.Exec which does not return on success.
func executeProcess(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}

// runClaude executes the claude command for the given provider.
// If supervisorMode is true, it generates settings with Stop hook configuration.
// This replaces the current process with claude using syscall.Exec.
// Provider env variables are passed to the claude subprocess.
func runClaude(cfg *config.Config, providerName string, claudeArgs []string, supervisorMode bool) error {
	var switchResult *provider.SwitchResult
	var supervisorID string

	if supervisorMode {
		os.Setenv("CCC_SUPERVISOR", "1")
	} else {
		os.Setenv("CCC_SUPERVISOR", "0")
	}

	// Generate settings for supervisor mode
	if supervisorMode {
		// Generate supervisor ID for this session
		supervisorID = uuid.New().String()
		// Set environment variable for hook to use
		os.Setenv("CCC_SUPERVISOR_ID", supervisorID)

		// Open log file and write initial messages directly to file
		// (not to stderr, since hook hasn't started yet)
		logFile, err := supervisor.OpenLogFile(supervisorID)
		if err != nil {
			return fmt.Errorf("failed to open supervisor log file: %w", err)
		}
		defer logFile.Close()

		// Show log file path to user
		logPath, err := supervisor.GetLogFilePath(supervisorID)
		if err != nil {
			return fmt.Errorf("failed to get log file path: %w", err)
		}
		fmt.Printf("Supervisor enabled: tail -f %s\n", logPath)

		// Write initial log messages directly to file (not stderr)
		timestamp := time.Now().Format(time.RFC3339Nano)
		fmt.Fprintf(logFile, "%s INFO Supervisor started supervisor_id=%s\n", timestamp, supervisorID)
		fmt.Fprintf(logFile, "%s INFO Waiting for Stop hook to trigger\n", timestamp)
	}

	// switch provider
	result, err := provider.SwitchWithHook(cfg, providerName)
	if err != nil {
		return fmt.Errorf("error switching provider: %w", err)
	}
	switchResult = result
	fmt.Printf("Launching with provider: %s\n", providerName)

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
	// Start with current process environment
	env := os.Environ()

	// Add merged provider env variables
	if switchResult.EnvVars != nil {
		envPairs := provider.EnvPairsToStrings(switchResult.EnvVars)
		env = append(env, envPairs...)
	}

	// Execute the process (replaces current process, does not return on success)
	return executeProcess(claudePath, execArgs, env)
}
