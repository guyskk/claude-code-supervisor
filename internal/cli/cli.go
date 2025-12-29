// Package cli handles command-line parsing and execution.
package cli

import (
	"fmt"
	"os"
	"os/exec"
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
}

// ValidateCommand represents options for the validate command.
type ValidateCommand struct {
	Provider    string // Empty means current provider
	ValidateAll bool
}

// Parse parses command-line arguments.
func Parse(args []string) *Command {
	cmd := &Command{}

	for i, arg := range args {
		if arg == "--version" || arg == "-v" {
			cmd.Version = true
			return cmd
		}
		if arg == "--help" || arg == "-h" {
			cmd.Help = true
			return cmd
		}
		// First non-flag argument might be a provider name or validate command
		if i == 0 && !strings.HasPrefix(arg, "-") {
			if arg == "validate" {
				cmd.Validate = true
				cmd.ValidateOpts = parseValidateArgs(args[1:])
				return cmd
			}
			cmd.Provider = arg
			cmd.ClaudeArgs = args[1:]
			return cmd
		}
	}

	// No arguments - use current provider
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

	// Switch to the provider
	fmt.Printf("Launching with provider: %s\n", providerName)
	mergedSettings, err := provider.Switch(cfg, providerName)
	if err != nil {
		return fmt.Errorf("error switching provider: %w", err)
	}

	// Run claude with the settings file
	if err := runClaude(providerName, mergedSettings, cmd.ClaudeArgs); err != nil {
		return fmt.Errorf("error running claude: %w", err)
	}

	return nil
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

// runClaude executes the claude command with the settings file.
func runClaude(providerName string, settings map[string]interface{}, args []string) error {
	settingsPath := config.GetSettingsPath(providerName)

	// Extract ANTHROPIC_AUTH_TOKEN from env
	authToken := provider.GetAuthToken(settings)

	// Build the claude command
	cmdArgs := append([]string{"--settings", settingsPath}, args...)
	claudeCmd := exec.Command("claude", cmdArgs...)

	// Set up environment variables
	claudeCmd.Env = append(os.Environ(), fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", authToken))

	// Set up stdin, stdout, stderr
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	// Execute the command
	if err := claudeCmd.Run(); err != nil {
		return fmt.Errorf("failed to run claude: %w", err)
	}

	return nil
}

// Execute is the main entry point for the CLI.
func Execute() error {
	cmd := Parse(os.Args[1:])
	return Run(cmd)
}
