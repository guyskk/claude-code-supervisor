// Package config provides supervisor configuration management for ccc.
package config

import (
	"encoding/json"
	"fmt"
	"time"
)

// SupervisorConfig holds the configuration for Supervisor Mode.
type SupervisorConfig struct {
	// MaxIterations is the maximum number of supervisor iterations before allowing stop.
	// Default is 20.
	MaxIterations int `json:"max_iterations"`

	// TimeoutSeconds is the timeout for each supervisor call in seconds.
	// Default is 600 (10 minutes).
	TimeoutSeconds int `json:"timeout_seconds"`
}

// DefaultSupervisorConfig returns the default supervisor configuration.
func DefaultSupervisorConfig() *SupervisorConfig {
	return &SupervisorConfig{
		MaxIterations:  20,
		TimeoutSeconds: 600,
	}
}

// LoadSupervisorConfig loads the supervisor configuration from the ccc.json.
// If the supervisor section doesn't exist, returns defaults.
func LoadSupervisorConfig() (*SupervisorConfig, error) {
	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Start with defaults
	supervisorCfg := DefaultSupervisorConfig()

	// Merge supervisor config from top level if it exists
	// We merge instead of replace to preserve defaults for unset fields
	if cfg.Supervisor != nil {
		// Only override non-zero values
		if cfg.Supervisor.MaxIterations > 0 {
			supervisorCfg.MaxIterations = cfg.Supervisor.MaxIterations
		}
		if cfg.Supervisor.TimeoutSeconds > 0 {
			supervisorCfg.TimeoutSeconds = cfg.Supervisor.TimeoutSeconds
		}
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
		"max_iterations":  c.MaxIterations,
		"timeout_seconds": c.TimeoutSeconds,
	})
}
