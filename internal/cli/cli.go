// Package cli handles command-line parsing and execution.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/migration"
	"github.com/guyskk/ccc/internal/provider"
	"github.com/guyskk/ccc/internal/validate"
)

// Name is the project name.
const Name = "claude-code-config-switcher"

// Version is set by build flags during release.
var Version = "dev"

// BuildTime is set by build flags during release (ISO 8601 format).
var BuildTime = "unknown"

// Command represents a parsed CLI command.
type Command struct {
	Version    bool
	Help       bool
	Provider   string
	ClaudeArgs []string

	// Validate command options
	Validate     bool
	ValidateOpts *ValidateCommand

	// supervisor-hook subcommand
	HookSubcommand bool
}

// ValidateCommand represents options for the validate command.
type ValidateCommand struct {
	Provider    string // Empty means current provider
	ValidateAll bool
}

// Parse parses command-line arguments.
// Only recognizes --help, --version, and the first argument as provider name.
// All other arguments are passed through to Claude.
func Parse(args []string) *Command {
	cmd := &Command{}

	// Check for supervisor-hook subcommand first
	if len(args) > 0 && args[0] == "supervisor-hook" {
		cmd.HookSubcommand = true
		return cmd
	}

	for i, arg := range args {
		if arg == "--version" || arg == "-v" {
			cmd.Version = true
			return cmd
		}
		if arg == "--help" || arg == "-h" {
			cmd.Help = true
			return cmd
		}
		// First non-flag argument is provider name or validate command
		if !strings.HasPrefix(arg, "-") {
			if arg == "validate" {
				cmd.Validate = true
				cmd.ValidateOpts = parseValidateArgs(args[i+1:])
				return cmd
			}
			// First non-flag argument is the provider name
			if cmd.Provider == "" {
				cmd.Provider = arg
				// Everything after provider is passed to Claude
				if i+1 < len(args) {
					cmd.ClaudeArgs = args[i+1:]
				}
				return cmd
			}
		}
	}

	// No provider specified - use current provider
	// All arguments are passed to Claude (e.g., "ccc --debug" passes --debug to claude)
	cmd.ClaudeArgs = args
	return cmd
}

// parseValidateArgs parses arguments for the validate command.
func parseValidateArgs(args []string) *ValidateCommand {
	opts := &ValidateCommand{}

	for i, arg := range args {
		if arg == "--all" {
			opts.ValidateAll = true
		} else if !strings.HasPrefix(arg, "--") && i == 0 {
			// First non-flag argument is the provider name
			opts.Provider = arg
		}
	}

	return opts
}

// ShowHelp displays usage information.
func ShowHelp(cfg *config.Config, cfgErr error) {
	help := `Usage: ccc [provider] [args...]
       ccc validate [provider] [--all]

Claude Code Configuration Switcher

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
	if cmd.HookSubcommand {
		return RunSupervisorHook(os.Args[2:])
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

	// Handle validate command (needs config but doesn't run claude)
	if cmd.Validate {
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
		return runValidate(cfg, cmd.ValidateOpts)
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
