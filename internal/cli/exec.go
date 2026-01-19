package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

// determineProvider determines which provider to use based on the command and config.
func determineProvider(cmd *Command, cfg *config.Config) string {
	if cmd.Provider != "" {
		// User specified a provider, check if it's valid
		if _, exists := cfg.Providers[cmd.Provider]; exists {
			return cmd.Provider
		}
		// Not a valid provider, try using current provider
		if cfg.CurrentProvider != "" {
			fmt.Printf("Unknown provider: %s\n", cmd.Provider)
			fmt.Printf("Using current provider: %s\n", cfg.CurrentProvider)
			return cfg.CurrentProvider
		}
		return ""
	}

	// No provider specified, use current or first available
	if cfg.CurrentProvider != "" {
		return cfg.CurrentProvider
	}

	// Use the first available provider
	for name := range cfg.Providers {
		return name
	}

	return ""
}

// runClaude executes the claude command for the given provider.
// It always generates settings with Stop hook configuration.
// This replaces the current process with claude using syscall.Exec.
// Provider env variables are passed to the claude subprocess.
func runClaude(cfg *config.Config, cmd *Command) error {
	var switchResult *provider.SwitchResult

	// Determine which provider to use
	providerName := determineProvider(cmd, cfg)
	if providerName == "" {
		return fmt.Errorf("no providers configured")
	}

	// Check if supervisor ID is already set in environment
	// (e.g., from previous supervisor iteration or when ccc is called again)
	supervisorID := os.Getenv("CCC_SUPERVISOR_ID")
	if supervisorID == "" {
		// Generate new supervisor ID for this session
		supervisorID = uuid.New().String()
		os.Setenv("CCC_SUPERVISOR_ID", supervisorID)
	}

	// Open log file and write initial messages directly to file
	// (not to stderr, since hook hasn't started yet)
	logFile, err := supervisor.OpenLogFile(supervisorID)
	if err != nil {
		return fmt.Errorf("failed to open supervisor log file: %w", err)
	}
	defer logFile.Close()

	// Show log file path to user (only in debug mode)
	if cmd.Debug {
		logPath, err := supervisor.GetLogFilePath(supervisorID)
		if err != nil {
			return fmt.Errorf("failed to get log file path: %w", err)
		}
		fmt.Printf("Supervisor log: tail -f %s\n", logPath)
	}

	// Write initial log messages directly to file (not stderr)
	timestamp := time.Now().Format(time.RFC3339Nano)
	fmt.Fprintf(logFile, "%s INFO Supervisor started supervisor_id=%s\n", timestamp, supervisorID)
	fmt.Fprintf(logFile, "%s INFO Use /supervisor command to enable supervisor mode\n", timestamp)

	// Switch provider (always use SwitchWithHook to generate settings with Stop hook)
	result, err := provider.SwitchWithHook(cfg, providerName)
	if err != nil {
		return fmt.Errorf("error switching provider: %w", err)
	}
	switchResult = result
	fmt.Printf("Launching with provider: %s\n", providerName)

	// Find claude executable path
	// 优先使用 CCC_CLAUDE 环境变量（由包装脚本设置）
	var claudePath string
	if realPath := os.Getenv("CCC_CLAUDE"); realPath != "" {
		// 环境变量存在，验证路径是否有效
		// 使用 exec.LookPath 验证文件存在且可执行
		claudePath, err = exec.LookPath(realPath)
		if err != nil {
			return fmt.Errorf("CCC_CLAUDE environment variable points to invalid path: %s: %w", realPath, err)
		}
	} else {
		// 环境变量不存在，使用 LookPath 查找
		claudePath, err = exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude not found in PATH: %w", err)
		}
	}

	// Build arguments (argv[0] must be the program name)
	execArgs := []string{"claude"}
	if len(cfg.ClaudeArgs) > 0 {
		execArgs = append(execArgs, cfg.ClaudeArgs...)
	}
	execArgs = append(execArgs, cmd.ClaudeArgs...)

	// Build environment variables
	// Start with current process environment
	env := os.Environ()

	// Remove existing environment variables to ensure provider config takes precedence
	prefixes := []string{"CLAUDE_", "ANTHROPIC_"}
	env = filterEnvVars(env, func(key string) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				return false
			}
		}
		return true
	})

	// Add merged provider env variables
	if switchResult.EnvVars != nil {
		envPairs := provider.EnvPairsToStrings(switchResult.EnvVars)
		env = append(env, envPairs...)
	}

	// Execute the process (replaces current process, does not return on success)
	return executeProcess(claudePath, execArgs, env)
}

// filterEnvVars filters environment variables based on a predicate function
func filterEnvVars(env []string, shouldKeep func(string) bool) []string {
	var filtered []string
	for _, envVar := range env {
		if parts := strings.SplitN(envVar, "=", 2); len(parts) == 2 {
			if shouldKeep(parts[0]) {
				filtered = append(filtered, envVar)
			}
		}
	}
	return filtered
}
