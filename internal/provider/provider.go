// Package provider handles provider switching and configuration merging.
package provider

import (
	"fmt"
	"strings"

	"github.com/guyskk/ccc/internal/config"
)

// Switch switches to the specified provider by merging configurations.
// It saves the merged settings to settings-{provider}.json, clears env in settings.json,
// and updates the current provider in ccc.json.
func Switch(cfg *config.Config, providerName string) (map[string]interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	// Check if provider exists
	providerSettings, exists := cfg.Providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found in configuration", providerName)
	}

	// Create the merged settings
	// Start with the base settings template
	mergedSettings := config.DeepMerge(cfg.Settings, providerSettings)

	// Save the merged settings to settings-{provider}.json
	if err := config.SaveSettings(mergedSettings, providerName); err != nil {
		return nil, fmt.Errorf("failed to save settings: %w", err)
	}

	// Clear env field in settings.json to prevent configuration pollution
	cleared, err := config.ClearEnvInSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to clear env in settings.json: %w", err)
	}
	if cleared {
		fmt.Println("Cleared env field in settings.json to prevent configuration pollution")
	}

	// Update current_provider in ccc.json
	cfg.CurrentProvider = providerName
	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to update current provider: %w", err)
	}

	return mergedSettings, nil
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
