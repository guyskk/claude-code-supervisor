// Package config provides configuration management for ccc.
// Claude settings use dynamic map[string]interface{} to handle arbitrary fields.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetDirFunc is a function that returns the Claude configuration directory.
// This variable allows tests to override the default behavior.
var GetDirFunc = func() string {
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

// GetDir returns the Claude configuration directory.
func GetDir() string {
	return GetDirFunc()
}

// Config represents the ccc.json configuration structure.
// Settings and Providers use dynamic maps to handle arbitrary Claude settings fields.
type Config struct {
	Settings        map[string]interface{}            `json:"settings"`
	CurrentProvider string                            `json:"current_provider"`
	Providers       map[string]map[string]interface{} `json:"providers"`
}

// GetConfigPath returns the path to ccc.json.
func GetConfigPath() string {
	return filepath.Join(GetDir(), "ccc.json")
}

// GetSettingsPath returns the path to settings-{provider}.json.
// If providerName is empty, returns the path to settings.json.
func GetSettingsPath(providerName string) string {
	if providerName == "" {
		return filepath.Join(GetDir(), "settings.json")
	}
	return filepath.Join(GetDir(), fmt.Sprintf("settings-%s.json", providerName))
}

// Load reads and parses the ccc.json configuration file.
func Load() (*Config, error) {
	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to ccc.json.
func Save(cfg *Config) error {
	configPath := GetConfigPath()

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveSettings writes the settings to a provider-specific settings file.
func SaveSettings(settings map[string]interface{}, providerName string) error {
	settingsPath := GetSettingsPath(providerName)

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

// deepCopy creates a deep copy of a map[string]interface{}.
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

// DeepMerge recursively merges provider settings into base settings.
// Provider settings override base settings for the same keys.
// This function handles arbitrary Claude settings fields.
func DeepMerge(base, provider map[string]interface{}) map[string]interface{} {
	result := deepCopy(base)
	if result == nil {
		result = make(map[string]interface{})
	}

	for key, value := range provider {
		if existingVal, exists := result[key]; exists {
			// If both are maps, merge them recursively
			if existingMap, ok := existingVal.(map[string]interface{}); ok {
				if newMap, ok := value.(map[string]interface{}); ok {
					result[key] = DeepMerge(existingMap, newMap)
					continue
				}
			}
		}
		// Otherwise, override with provider value
		result[key] = value
	}

	return result
}

// GetEnv extracts the env map from settings.
// Returns nil if env doesn't exist or is not a map.
func GetEnv(settings map[string]interface{}) map[string]interface{} {
	if settings == nil {
		return nil
	}
	if envVal, exists := settings["env"]; exists {
		if envMap, ok := envVal.(map[string]interface{}); ok {
			return envMap
		}
	}
	return nil
}

// GetEnvString extracts a string value from settings.env.
// Returns defaultValue if the key doesn't exist.
func GetEnvString(settings map[string]interface{}, key, defaultValue string) string {
	env := GetEnv(settings)
	if env == nil {
		return defaultValue
	}
	if val, exists := env[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetAuthToken extracts the ANTHROPIC_AUTH_TOKEN from settings.
// Returns a placeholder if the token is not set.
func GetAuthToken(settings map[string]interface{}) string {
	token := GetEnvString(settings, "ANTHROPIC_AUTH_TOKEN", "")
	if token == "" {
		return "PLEASE_SET_ANTHROPIC_AUTH_TOKEN"
	}
	return token
}

// GetBaseURL extracts the ANTHROPIC_BASE_URL from settings.
// Returns empty string if not set.
func GetBaseURL(settings map[string]interface{}) string {
	return GetEnvString(settings, "ANTHROPIC_BASE_URL", "")
}

// GetModel extracts the ANTHROPIC_MODEL from settings.
// Returns empty string if not set.
func GetModel(settings map[string]interface{}) string {
	return GetEnvString(settings, "ANTHROPIC_MODEL", "")
}
