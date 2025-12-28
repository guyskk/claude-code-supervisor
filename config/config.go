// Package config provides type-safe configuration management for ccc.
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

// Env represents environment variables configuration.
type Env map[string]string

// Permissions represents Claude's permissions configuration.
type Permissions struct {
	Allow        []string `json:"allow,omitempty"`
	DefaultMode  string   `json:"defaultMode,omitempty"`
}

// Settings represents Claude's settings configuration.
type Settings struct {
	Permissions             *Permissions `json:"permissions,omitempty"`
	AlwaysThinkingEnabled   bool         `json:"alwaysThinkingEnabled,omitempty"`
	Env                     Env          `json:"env,omitempty"`
}

// ProviderConfig represents a single provider's configuration.
type ProviderConfig struct {
	Env Env `json:"env,omitempty"`
}

// Config represents the ccc.json configuration structure.
type Config struct {
	Settings        Settings                  `json:"settings"`
	CurrentProvider string                    `json:"current_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
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
func SaveSettings(settings *Settings, providerName string) error {
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

// Merge merges two Env maps, with env2 taking precedence for conflicting keys.
func MergeEnv(env1, env2 Env) Env {
	result := make(Env)
	for k, v := range env1 {
		result[k] = v
	}
	for k, v := range env2 {
		result[k] = v
	}
	return result
}

// MergePermissions merges two Permissions, with p2 taking precedence.
func MergePermissions(p1, p2 *Permissions) *Permissions {
	if p1 == nil && p2 == nil {
		return nil
	}
	if p2 == nil {
		return p1
	}
	if p1 == nil {
		return p2
	}

	result := &Permissions{
		Allow:       p1.Allow,
		DefaultMode: p1.DefaultMode,
	}
	if p2.Allow != nil {
		result.Allow = p2.Allow
	}
	if p2.DefaultMode != "" {
		result.DefaultMode = p2.DefaultMode
	}
	return result
}

// MergeSettings merges base settings with provider settings, provider takes precedence.
func MergeSettings(base *Settings, provider *ProviderConfig) *Settings {
	if base == nil {
		base = &Settings{}
	}

	result := &Settings{
		AlwaysThinkingEnabled: base.AlwaysThinkingEnabled,
		Permissions:           base.Permissions,
		Env:                   make(Env),
	}

	// Copy base env
	for k, v := range base.Env {
		result.Env[k] = v
	}

	// Merge provider env if present
	if provider != nil && provider.Env != nil {
		for k, v := range provider.Env {
			result.Env[k] = v
		}
	}

	// Handle nil env
	if len(result.Env) == 0 {
		result.Env = nil
	}

	return result
}
