// Package cli handles command-line parsing and execution.
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/migration"
	"github.com/guyskk/ccc/internal/provider"
	"github.com/guyskk/ccc/internal/validate"
)

// Name is the project name.
const Name = "claude-code-supervisor"

// Version is set by build flags during release.
var Version = "dev"

// BuildTime is set by build flags during release (ISO 8601 format).
var BuildTime = "unknown"

// Command represents a parsed CLI command.
type Command struct {
	Version            bool
	Help               bool
	Provider           string
	ClaudeArgs         []string
	Validate           bool
	ValidateOpts       *ValidateCommand
	SupervisorHook     bool
	SupervisorHookOpts *SupervisorHookCommand
}

// ValidateCommand represents options for the validate command.
type ValidateCommand struct {
	Provider    string // Empty means current provider
	ValidateAll bool
}

type SupervisorHookCommand struct {
	SessionId string // Can be empty
}

// Parse parses command-line arguments.
func Parse(args []string) *Command {
	cmd := &Command{}
	// 根据第一个参数判断是否是ccc的参数，其余参数透传给claude
	firstArg := ""
	if len(args) > 0 {
		firstArg = args[0]
	}
	if firstArg == "--version" || firstArg == "-v" {
		cmd.Version = true
	} else if firstArg == "--help" || firstArg == "-h" {
		cmd.Help = true
	} else if firstArg == "validate" {
		cmd.Validate = true
		cmd.ValidateOpts = parseValidateArgs(args[1:])
	} else if firstArg == "supervisor-hook" {
		cmd.SupervisorHook = true
		cmd.SupervisorHookOpts = parseSupervisorHookArgs(args[1:])
	} else if !strings.HasPrefix(firstArg, "-") {
		cmd.Provider = firstArg
		if len(args) > 1 {
			cmd.ClaudeArgs = args[1:]
		}
	} else {
		cmd.ClaudeArgs = args
	}
	return cmd
}

// parseValidateArgs parses arguments for the validate command.
func parseValidateArgs(args []string) *ValidateCommand {
	opts := &ValidateCommand{}

	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.Usage = func() {} // Suppress default usage output
	all := fs.Bool("all", false, "validate all providers")

	if err := fs.Parse(args); err != nil {
		// On parse error, return options with defaults
		return opts
	}

	opts.ValidateAll = *all

	// Get remaining arguments as positional args
	remaining := fs.Args()
	if len(remaining) > 0 {
		opts.Provider = remaining[0]
	}

	return opts
}

// parseSupervisorHookArgs parses arguments for the supervisor-hook command.
func parseSupervisorHookArgs(args []string) *SupervisorHookCommand {
	opts := &SupervisorHookCommand{}

	fs := flag.NewFlagSet("supervisor-hook", flag.ContinueOnError)
	fs.Usage = func() {} // Suppress default usage output
	sessionId := fs.String("session-id", "", "supervisor session id")

	if err := fs.Parse(args); err != nil {
		// On parse error, return options with defaults
		return opts
	}

	opts.SessionId = *sessionId
	return opts
}

// ShowHelp displays usage information.
func ShowHelp(cfg *config.Config, cfgErr error) {
	help := `Usage: ccc [provider] [args...]
       ccc validate [provider] [--all]

Claude Code Supervisor and Configuration Switcher

Commands:
  ccc              Use the current provider (or the first provider if none is set)
  ccc <provider>   Switch to the specified provider and run Claude Code
  ccc validate     Validate the current provider configuration
  ccc validate <provider>   Validate a specific provider configuration
  ccc validate --all        Validate all provider configurations
  ccc --help       Show this help message
  ccc --version    Show version information

Environment Variables:
  CCC_CONFIG_DIR     Override the configuration directory (default: ~/.claude/)
  CCC_SUPERVISOR     Configure Supervisor mode (set "1" to enable, "0" to disable)

Supervisor Mode:
  When enabled, ccc automatically runs a Supervisor check
  after each Agent stop. The Supervisor reviews the work quality and provides
  feedback if incomplete. Creates an action-feedback loop until the Supervisor
  confirms task completion.

  Example:
    export CCC_SUPERVISOR=1
    ccc glm
`
	fmt.Print(help)

	// Display config path and status
	configPath := config.GetConfigPath()
	if cfgErr != nil {
		errMsg := provider.ShortenError(cfgErr, 40)
		fmt.Printf("\nCurrent config: %s (%s)\n", configPath, errMsg)
	} else {
		fmt.Printf("\nCurrent config: %s\n", configPath)

		// Display provider list from config
		if cfg != nil && len(cfg.Providers) > 0 {
			fmt.Println("\nAvailable Providers:")
			for name := range cfg.Providers {
				marker := ""
				if name == cfg.CurrentProvider {
					marker = " (current)"
				}
				fmt.Printf("  %s%s\n", name, marker)
			}
		}
	}
	fmt.Println()
}

// ShowVersion displays version information.
func ShowVersion() {
	fmt.Printf("%s version %s (built at %s)\n", Name, Version, BuildTime)
}

// Run executes the CLI command.
func Run(cmd *Command) error {
	// Handle supervisor-hook subcommand
	if cmd.SupervisorHook {
		return RunSupervisorHook(cmd.SupervisorHookOpts)
	}

	// Handle --version
	if cmd.Version {
		ShowVersion()
		return nil
	}

	// Handle --help
	if cmd.Help {
		cfg, err := config.Load()
		ShowHelp(cfg, err)
		return nil
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Try to migrate from existing settings.json
		if migration.CheckExisting() && migration.PromptUser() {
			if err := migration.MigrateFromSettings(); err != nil {
				return fmt.Errorf("error migrating from settings: %w", err)
			}
			// Reload config after migration
			cfg, err = config.Load()
			if err != nil {
				ShowHelp(nil, err)
				return err
			}
		} else {
			ShowHelp(nil, err)
			return err
		}
	}

	if cmd.Validate {
		return runValidate(cfg, cmd.ValidateOpts)
	}

	// Determine which provider to use
	providerName := determineProvider(cmd, cfg)
	if providerName == "" {
		return fmt.Errorf("no providers configured")
	}

	// Check if supervisor mode is enabled
	supervisorCfg, err := config.LoadSupervisorConfig()
	if err != nil {
		return fmt.Errorf("failed to load supervisor config: %w", err)
	}

	// Run claude with the provider (handles both normal and supervisor mode)
	return runClaude(cfg, providerName, cmd.ClaudeArgs, supervisorCfg.Enabled)
}

// runValidate executes the validate command.
func runValidate(cfg *config.Config, opts *ValidateCommand) error {
	// Create a config adapter for the validate package
	cfgAdapter := &configAdapter{cfg: cfg}

	validateOpts := &validate.RunOptions{
		Provider:    opts.Provider,
		ValidateAll: opts.ValidateAll,
	}

	return validate.Run(cfgAdapter, validateOpts)
}

// configAdapter adapts config.Config to the validate.Config interface.
type configAdapter struct {
	cfg *config.Config
}

func (a *configAdapter) Providers() map[string]map[string]interface{} {
	return a.cfg.Providers
}

func (a *configAdapter) CurrentProvider() string {
	return a.cfg.CurrentProvider
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

// Execute is the main entry point for the CLI.
func Execute() error {
	cmd := Parse(os.Args[1:])
	return Run(cmd)
}
