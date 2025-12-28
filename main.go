package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version is set by build flags during release
var Version = "dev"

// getClaudeDirFunc is a variable that holds the function to get the Claude directory
// This allows tests to override it for testing purposes
var getClaudeDirFunc = func() string {
	if workDir := os.Getenv("CCC_CONFIG_DIR"); workDir != "" {
		return workDir
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(homeDir, ".claude")
}

// getClaudeDir returns the Claude configuration directory
func getClaudeDir() string {
	return getClaudeDirFunc()
}

// getUserInputFunc is a variable that holds the function to get user input
// This allows tests to override it for testing purposes
var getUserInputFunc = func(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}

// Config represents the structure of ccc.json
type Config struct {
	Settings        map[string]interface{}            `json:"settings"`
	CurrentProvider string                            `json:"current_provider"`
	Providers       map[string]map[string]interface{} `json:"providers"`
}

// getConfigPath returns the path to ccc.json
func getConfigPath() string {
	return filepath.Join(getClaudeDir(), "ccc.json")
}

// getSettingsPath returns the path to settings-{provider}.json
func getSettingsPath(providerName string) string {
	if providerName == "" {
		return filepath.Join(getClaudeDir(), "settings.json")
	}
	return filepath.Join(getClaudeDir(), fmt.Sprintf("settings-%s.json", providerName))
}

// showHelp displays usage information
func showHelp(config *Config, configErr error) {
	help := `Usage: ccc [provider] [args...]

Claude Code Configuration Switcher

Commands:
  ccc              Use the current provider (or the first provider if none is set)
  ccc <provider>   Switch to the specified provider and run Claude Code
  ccc --help       Show this help message

Environment Variables:
  CCC_CONFIG_DIR     Override the configuration directory (default: ~/.claude/)
`
	fmt.Print(help)

	// Display config path and status
	configPath := getConfigPath()
	if configErr != nil {
		// Extract short error message - take only the last part (most relevant)
		errMsg := configErr.Error()
		// Find the last colon for shorter error message
		if lastColon := strings.LastIndex(errMsg, ":"); lastColon > 0 && lastColon < len(errMsg)-2 {
			errMsg = errMsg[lastColon+2:]
		}
		// Still limit length
		if len(errMsg) > 40 {
			errMsg = errMsg[:37] + "..."
		}
		fmt.Printf("\nCurrent config: %s (%s)\n", configPath, errMsg)
	} else {
		fmt.Printf("\nCurrent config: %s\n", configPath)

		// Display provider list from config
		if len(config.Providers) > 0 {
			fmt.Println("\nAvailable Providers:")
			for name := range config.Providers {
				marker := ""
				if name == config.CurrentProvider {
					marker = " (current)"
				}
				fmt.Printf("  %s%s\n", name, marker)
			}
		}
	}
	fmt.Println()
}

// checkExistingSettings checks if ~/.claude/settings.json exists
func checkExistingSettings() bool {
	settingsPath := getSettingsPath("")
	_, err := os.Stat(settingsPath)
	return err == nil
}

// promptUserForMigration prompts the user to confirm migration from existing settings
func promptUserForMigration() bool {
	cccPath := getConfigPath()
	settingsPath := getSettingsPath("")

	fmt.Printf("ccc configuration not found: %s\n", cccPath)
	fmt.Printf("Found existing Claude configuration: %s\n", settingsPath)
	fmt.Println()

	input, err := getUserInputFunc("Would you like to create ccc config from existing settings? [y/N] ")
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

// migrateFromSettings creates a new ccc.json from existing settings.json
func migrateFromSettings() error {
	settingsPath := getSettingsPath("")

	// Read existing settings.json
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("failed to read settings file: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse settings file: %w", err)
	}

	// Extract env from settings
	var envConfig map[string]interface{}
	if envVal, exists := settings["env"]; exists {
		if envMap, ok := envVal.(map[string]interface{}); ok {
			envConfig = envMap
		}
	}

	// Remove env from settings to create base settings
	baseSettings := make(map[string]interface{})
	for k, v := range settings {
		if k != "env" {
			baseSettings[k] = v
		}
	}

	// Create new ccc config
	config := &Config{
		Settings:        baseSettings,
		CurrentProvider: "default",
		Providers:       make(map[string]map[string]interface{}),
	}

	// Always create default provider (even if env is empty)
	defaultProvider := make(map[string]interface{})
	if envConfig != nil {
		defaultProvider["env"] = envConfig
	}
	config.Providers["default"] = defaultProvider

	// Save the new config
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save ccc config: %w", err)
	}

	cccPath := getConfigPath()
	fmt.Printf("Created ccc config with 'default' provider: %s\n", cccPath)
	return nil
}

// loadConfig reads and parses ccc.json
func loadConfig() (*Config, error) {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// saveConfig writes the config back to ccc.json
func saveConfig(config *Config) error {
	configPath := getConfigPath()

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// saveSettings writes the merged settings to settings-{provider}.json
func saveSettings(settings map[string]interface{}, providerName string) error {
	settingsPath := getSettingsPath(providerName)

	// Ensure settings directory exists
	settingsDir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// deepCopy creates a deep copy of a map
func deepCopy(original map[string]interface{}) map[string]interface{} {
	if original == nil {
		return nil
	}

	copied := make(map[string]interface{})
	for k, v := range original {
		if nestedMap, ok := v.(map[string]interface{}); ok {
			copied[k] = deepCopy(nestedMap)
		} else {
			copied[k] = v
		}
	}
	return copied
}

// deepMerge recursively merges provider settings into base settings
// Provider settings override base settings for the same keys
func deepMerge(base, provider map[string]interface{}) map[string]interface{} {
	result := deepCopy(base)
	if result == nil {
		result = make(map[string]interface{})
	}

	for key, value := range provider {
		if existingVal, exists := result[key]; exists {
			// If both are maps, merge them recursively
			if existingMap, ok := existingVal.(map[string]interface{}); ok {
				if newMap, ok := value.(map[string]interface{}); ok {
					result[key] = deepMerge(existingMap, newMap)
					continue
				}
			}
		}
		// Otherwise, override with provider value
		result[key] = value
	}

	return result
}

// switchProvider switches to the specified provider by merging configurations
// Returns the merged settings for use by runClaude
func switchProvider(providerName string) (map[string]interface{}, error) {
	fmt.Printf("Launching with provider: %s\n", providerName)

	// Load the configuration
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Check if provider exists
	providerSettings, exists := config.Providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found in configuration", providerName)
	}

	// Create the merged settings
	// Start with the base settings template
	mergedSettings := deepCopy(config.Settings)
	if mergedSettings == nil {
		mergedSettings = make(map[string]interface{})
	}

	// Merge provider-specific settings
	mergedSettings = deepMerge(mergedSettings, providerSettings)

	// Save the merged settings to settings-{provider}.json
	if err := saveSettings(mergedSettings, providerName); err != nil {
		return nil, fmt.Errorf("failed to save settings: %w", err)
	}

	// Update current_provider in ccc.json
	config.CurrentProvider = providerName
	if err := saveConfig(config); err != nil {
		return nil, fmt.Errorf("failed to update current provider: %w", err)
	}

	return mergedSettings, nil
}

// runClaude executes the claude command with the settings file
func runClaude(providerName string, mergedSettings map[string]interface{}, args []string) error {
	settingsPath := getSettingsPath(providerName)

	// Extract ANTHROPIC_AUTH_TOKEN from env
	authToken := "PLEASE_SET_ANTHROPIC_AUTH_TOKEN"
	if envVal, exists := mergedSettings["env"]; exists {
		if envMap, ok := envVal.(map[string]interface{}); ok {
			if tokenVal, exists := envMap["ANTHROPIC_AUTH_TOKEN"]; exists {
				if tokenStr, ok := tokenVal.(string); ok && tokenStr != "" {
					authToken = tokenStr
				}
			}
		}
	}

	// Build the claude command
	cmd := exec.Command("claude", append([]string{"--settings", settingsPath}, args...)...)

	// Set up environment variables
	cmd.Env = append(os.Environ(), fmt.Sprintf("ANTHROPIC_AUTH_TOKEN=%s", authToken))

	// Set up stdin, stdout, stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run claude: %w", err)
	}

	return nil
}

func main() {
	// Parse command line arguments
	args := os.Args[1:]

	// Handle --version
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		fmt.Printf("ccc version %s\n", Version)
		os.Exit(0)
	}

	// Handle --help (try to load config for provider list, but show help anyway if it fails)
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		config, err := loadConfig()
		showHelp(config, err)
		os.Exit(0)
	}

	// Load configuration first to check if it exists and is valid
	config, err := loadConfig()
	if err != nil {
		// Try to migrate from existing settings.json
		if checkExistingSettings() && promptUserForMigration() {
			if err := migrateFromSettings(); err != nil {
				fmt.Fprintf(os.Stderr, "Error migrating from settings: %v\n", err)
				os.Exit(1)
			}
			// Reload config after migration
			config, err = loadConfig()
			if err != nil {
				showHelp(nil, err)
				os.Exit(1)
			}
		} else {
			showHelp(nil, err)
			os.Exit(1)
		}
	}

	// Determine which provider to use
	var providerName string
	var claudeArgs []string

	if len(args) == 0 {
		// No arguments - use current provider
		if config.CurrentProvider != "" {
			providerName = config.CurrentProvider
		} else {
			// Use the first available provider
			for name := range config.Providers {
				providerName = name
				break
			}
			if providerName == "" {
				fmt.Fprintf(os.Stderr, "No providers configured\n")
				os.Exit(1)
			}
		}
		claudeArgs = []string{}
	} else {
		// First argument might be a provider name
		potentialProvider := args[0]

		// Check if it's a valid provider
		if _, exists := config.Providers[potentialProvider]; exists {
			providerName = potentialProvider
			claudeArgs = args[1:]
		} else {
			// Not a provider name, use current provider and pass all args to claude
			if config.CurrentProvider != "" {
				providerName = config.CurrentProvider
				fmt.Printf("Using current provider: %s\n", providerName)
			} else {
				fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", potentialProvider)
				os.Exit(1)
			}
			claudeArgs = args
		}
	}

	// Switch to the provider (this will merge and save settings)
	mergedSettings, err := switchProvider(providerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error switching provider: %v\n", err)
		os.Exit(1)
	}

	// Run claude with the settings file
	if err := runClaude(providerName, mergedSettings, claudeArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error running claude: %v\n", err)
		os.Exit(1)
	}
}
