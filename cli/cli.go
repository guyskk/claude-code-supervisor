// Package cli handles command-line parsing and execution.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/user/ccc/config"
	"github.com/user/ccc/migration"
	"github.com/user/ccc/provider"
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
		// First non-flag argument might be a provider name
		if i == 0 && !strings.HasPrefix(arg, "-") {
			cmd.Provider = arg
			cmd.ClaudeArgs = args[1:]
			return cmd
		}
	}

	// No arguments - use current provider
	return cmd
}

// ShowHelp displays usage information.
func ShowHelp(cfg *config.Config, cfgErr error) {
	help := `Usage: ccc [provider] [args...]

Claude Code Configuration Switcher

Commands:
  ccc              Use the current provider (or the first provider if none is set)
  ccc <provider>   Switch to the specified provider and run Claude Code
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
