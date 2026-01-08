// Package config provides supervisor configuration management for ccc.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// SupervisorConfig holds the configuration for Supervisor Mode.
type SupervisorConfig struct {
	// Enabled indicates if Supervisor Mode is enabled.
	// Can be overridden by CCC_SUPERVISOR environment variable.
	Enabled bool

	// MaxIterations is the maximum number of supervisor iterations before allowing stop.
	// Default is 20.
	MaxIterations int

	// TimeoutSeconds is the timeout for each supervisor call in seconds.
	// Default is 600 (10 minutes).
	TimeoutSeconds int
}

// DefaultSupervisorConfig returns the default supervisor configuration.
func DefaultSupervisorConfig() *SupervisorConfig {
	return &SupervisorConfig{
		Enabled:        false,
		MaxIterations:  20,
		TimeoutSeconds: 600,
	}
}

// LoadSupervisorConfig loads the supervisor configuration from the ccc.json.
// If the supervisor section doesn't exist, returns defaults.
// Environment variable CCC_SUPERVISOR can override the enabled flag.
func LoadSupervisorConfig() (*SupervisorConfig, error) {
	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Start with defaults
	supervisorCfg := DefaultSupervisorConfig()

	// Extract supervisor section from config if it exists
	if supervisorMap, exists := cfg.Settings["supervisor"]; exists {
		if supervisorSection, ok := supervisorMap.(map[string]interface{}); ok {
			// Parse enabled
			if enabledVal, exists := supervisorSection["enabled"]; exists {
				if enabled, ok := enabledVal.(bool); ok {
					supervisorCfg.Enabled = enabled
				}
			}

			// Parse max_iterations
			if maxIterVal, exists := supervisorSection["max_iterations"]; exists {
				switch v := maxIterVal.(type) {
				case float64:
					supervisorCfg.MaxIterations = int(v)
				case int:
					supervisorCfg.MaxIterations = v
				case string:
					if i, err := strconv.Atoi(v); err == nil {
						supervisorCfg.MaxIterations = i
					}
				}
			}

			// Parse timeout_seconds
			if timeoutVal, exists := supervisorSection["timeout_seconds"]; exists {
				switch v := timeoutVal.(type) {
				case float64:
					supervisorCfg.TimeoutSeconds = int(v)
				case int:
					supervisorCfg.TimeoutSeconds = v
				case string:
					if i, err := strconv.Atoi(v); err == nil {
						supervisorCfg.TimeoutSeconds = i
					}
				}
			}
		}
	}

	// Apply environment variable override for enabled flag only
	if enabledEnv := os.Getenv("CCC_SUPERVISOR"); enabledEnv != "" {
		supervisorCfg.Enabled = enabledEnv == "1" || enabledEnv == "true"
	}

	return supervisorCfg, nil
}

// Timeout returns the timeout as a time.Duration.
func (c *SupervisorConfig) Timeout() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// Validate checks if the supervisor configuration is valid.
func (c *SupervisorConfig) Validate() error {
	if c.MaxIterations < 1 {
		return fmt.Errorf("max_iterations must be at least 1, got %d", c.MaxIterations)
	}
	if c.MaxIterations > 100 {
		return fmt.Errorf("max_iterations must be at most 100, got %d", c.MaxIterations)
	}
	if c.TimeoutSeconds < 10 {
		return fmt.Errorf("timeout_seconds must be at least 10, got %d", c.TimeoutSeconds)
	}
	if c.TimeoutSeconds > 3600 {
		return fmt.Errorf("timeout_seconds must be at most 3600 (1 hour), got %d", c.TimeoutSeconds)
	}
	return nil
}

// MarshalJSON implements json.Marshaler for SupervisorConfig.
func (c *SupervisorConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"enabled":         c.Enabled,
		"max_iterations":  c.MaxIterations,
		"timeout_seconds": c.TimeoutSeconds,
	})
}
