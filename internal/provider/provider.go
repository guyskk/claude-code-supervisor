// Package provider handles provider switching and configuration merging.
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/guyskk/ccc/internal/config"
)

// EnvPair represents a single environment variable key-value pair.
type EnvPair struct {
	Key   string
	Value string
}

// SwitchResult contains the result of switching providers.
// It includes the merged env variables that should be passed to the child process.
type SwitchResult struct {
	// Settings is the merged settings (without env) that was saved to settings.json
	Settings map[string]interface{}
	// EnvVars contains the merged environment variables (settings.env + provider.env)
	// that should be passed to the claude subprocess
	EnvVars []EnvPair
}

// SwitchWithHook switches to the specified provider and adds Stop hook configuration.
// It generates settings.json (without env) with Stop hook for Supervisor mode.
// It also creates slash command files for enabling/disabling Supervisor mode.
// Returns the merged env that should be passed to the claude subprocess.
func SwitchWithHook(cfg *config.Config, providerName string) (*SwitchResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	// Check if provider exists
	providerSettings, exists := cfg.Providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found in configuration", providerName)
	}

	// Create the merged settings
	mergedSettings := config.DeepMerge(cfg.Settings, providerSettings)

	// Extract env before removing it from settings
	envMap := config.GetEnv(mergedSettings)

	// Get ccc absolute path for hook command
	cccPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get ccc executable path: %w", err)
	}

	// Build hook command (no --state-dir parameter, state dir is handled internally)
	hookCommand := fmt.Sprintf("%s supervisor-hook", cccPath)

	// Build settings with hook, but without env
	settingsWithHook := make(map[string]interface{})
	for k, v := range mergedSettings {
		if k != "env" {
			settingsWithHook[k] = v
		}
	}

	// Ensure hooks are enabled (these settings may prevent hooks from executing)
	settingsWithHook["disableAllHooks"] = false
	settingsWithHook["allowManagedHooksOnly"] = false

	// Create hooks configuration
	hooks := map[string]interface{}{
		"Stop": []map[string]interface{}{
			{
				"hooks": []map[string]interface{}{
					{
						"type":    "command",
						"command": hookCommand,
						"timeout": 600,
					},
				},
			},
		},
	}
	settingsWithHook["hooks"] = hooks

	// Save settings with hook (without env) to settings.json
	settingsPath := config.GetSettingsPath()
	settingsData, err := json.MarshalIndent(settingsWithHook, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write settings: %w", err)
	}

	// Create slash command files for enabling/disabling Supervisor mode
	if err := createSupervisorCommandFiles(cccPath); err != nil {
		return nil, fmt.Errorf("failed to create supervisor command files: %w", err)
	}

	// Update current_provider in ccc.json
	cfg.CurrentProvider = providerName
	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to update current provider: %w", err)
	}

	// Convert env map to EnvPair slice
	envVars := envMapToPairs(envMap)

	return &SwitchResult{
		Settings: settingsWithHook,
		EnvVars:  envVars,
	}, nil
}

// createSupervisorCommandFiles creates the slash command files for enabling/disabling Supervisor mode.
func createSupervisorCommandFiles(cccPath string) error {
	// Get the commands directory
	commandsDir := config.GetDir() + "/commands"

	// Ensure commands directory exists
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	// Create supervisor.md (enable command)
	supervisorOnContent := fmt.Sprintf("---\ndescription: Enable supervisor mode\n---\n$ARGUMENTS!`%s supervisor-mode on`\n", cccPath)
	supervisorOnPath := commandsDir + "/supervisor.md"
	if err := os.WriteFile(supervisorOnPath, []byte(supervisorOnContent), 0644); err != nil {
		return fmt.Errorf("failed to write command supervisor.md: %w", err)
	}

	// Create supervisoroff.md (disable command)
	supervisorOffContent := fmt.Sprintf("---\ndescription: Disable supervisor mode\n---\n$ARGUMENTS!`%s supervisor-mode off`\n", cccPath)
	supervisorOffPath := commandsDir + "/supervisoroff.md"
	if err := os.WriteFile(supervisorOffPath, []byte(supervisorOffContent), 0644); err != nil {
		return fmt.Errorf("failed to write command supervisoroff.md: %w", err)
	}

	return nil
}

// envMapToPairs converts a map[string]interface{} to []EnvPair.
// It expands environment variable references like ${VAR}.
func envMapToPairs(envMap map[string]interface{}) []EnvPair {
	if envMap == nil {
		return nil
	}

	pairs := make([]EnvPair, 0, len(envMap))
	for k, v := range envMap {
		value := fmt.Sprintf("%v", v)
		// Expand environment variable references
		value = os.ExpandEnv(value)
		pairs = append(pairs, EnvPair{Key: k, Value: value})
	}
	return pairs
}

// EnvPairsToStrings converts []EnvPair to []string in "KEY=value" format.
func EnvPairsToStrings(pairs []EnvPair) []string {
	if pairs == nil {
		return nil
	}

	result := make([]string, len(pairs))
	for i, pair := range pairs {
		result[i] = fmt.Sprintf("%s=%s", pair.Key, pair.Value)
	}
	return result
}

// FormatProviderName formats a provider name for display.
// If the name is the current provider, adds a "(current)" suffix.
func FormatProviderName(name, currentProvider string) string {
	if name == currentProvider {
		return fmt.Sprintf("%s (current)", name)
	}
	return name
}

// ListProviders returns a list of all provider names from the config.
func ListProviders(cfg *config.Config) []string {
	if cfg == nil || len(cfg.Providers) == 0 {
		return []string{}
	}

	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	return names
}

// ValidateProvider checks if a provider name exists in the config.
func ValidateProvider(cfg *config.Config, providerName string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if _, exists := cfg.Providers[providerName]; !exists {
		return fmt.Errorf("provider '%s' not found", providerName)
	}
	return nil
}

// GetDefaultProvider returns the first provider name from the config.
// Returns empty string if no providers are configured.
func GetDefaultProvider(cfg *config.Config) string {
	if cfg == nil || len(cfg.Providers) == 0 {
		return ""
	}
	for name := range cfg.Providers {
		return name
	}
	return ""
}

// GetCurrentProvider returns the current provider from config.
// If current_provider is not set, returns the first available provider.
// Returns empty string if no providers are configured.
func GetCurrentProvider(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}

	// Try current_provider first
	if cfg.CurrentProvider != "" {
		if _, exists := cfg.Providers[cfg.CurrentProvider]; exists {
			return cfg.CurrentProvider
		}
	}

	// Fall back to first provider
	return GetDefaultProvider(cfg)
}

// ShortenError creates a shortened error message for display.
func ShortenError(err error, maxLength int) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()
	// Find the last colon for shorter error message
	if lastColon := strings.LastIndex(errMsg, ":"); lastColon > 0 && lastColon < len(errMsg)-2 {
		errMsg = errMsg[lastColon+2:]
	}
	// Limit length
	if len(errMsg) > maxLength {
		errMsg = errMsg[:maxLength-3] + "..."
	}
	return errMsg
}

// GetAuthToken extracts the ANTHROPIC_AUTH_TOKEN from merged settings.
// This is a convenience wrapper around config.GetAuthToken.
func GetAuthToken(settings map[string]interface{}) string {
	return config.GetAuthToken(settings)
}

// GetBaseURL extracts the ANTHROPIC_BASE_URL from merged settings.
// This is a convenience wrapper around config.GetBaseURL.
func GetBaseURL(settings map[string]interface{}) string {
	return config.GetBaseURL(settings)
}

// GetModel extracts the ANTHROPIC_MODEL from merged settings.
// This is a convenience wrapper around config.GetModel.
func GetModel(settings map[string]interface{}) string {
	return config.GetModel(settings)
}
