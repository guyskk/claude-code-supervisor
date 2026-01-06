// Package config provides tests for supervisor configuration.
package config

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDefaultSupervisorConfig(t *testing.T) {
	cfg := DefaultSupervisorConfig()

	if cfg.Enabled {
		t.Error("default Enabled should be false")
	}
	if cfg.MaxIterations != 20 {
		t.Errorf("default MaxIterations = %d, want 20", cfg.MaxIterations)
	}
	if cfg.TimeoutSeconds != 600 {
		t.Errorf("default TimeoutSeconds = %d, want 600", cfg.TimeoutSeconds)
	}
	if cfg.PromptPath != "~/.claude/SUPERVISOR.md" {
		t.Errorf("default PromptPath = %s, want ~/.claude/SUPERVISOR.md", cfg.PromptPath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel = %s, want info", cfg.LogLevel)
	}
}

func TestSupervisorConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *SupervisorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 600,
				LogLevel:       "info",
			},
			wantErr: false,
		},
		{
			name: "max_iterations too low",
			cfg: &SupervisorConfig{
				MaxIterations:  0,
				TimeoutSeconds: 600,
				LogLevel:       "info",
			},
			wantErr: true,
		},
		{
			name: "max_iterations too high",
			cfg: &SupervisorConfig{
				MaxIterations:  101,
				TimeoutSeconds: 600,
				LogLevel:       "info",
			},
			wantErr: true,
		},
		{
			name: "timeout_seconds too low",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 5,
				LogLevel:       "info",
			},
			wantErr: true,
		},
		{
			name: "timeout_seconds too high",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 4000,
				LogLevel:       "info",
			},
			wantErr: true,
		},
		{
			name: "invalid log_level",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 600,
				LogLevel:       "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid debug log_level",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 600,
				LogLevel:       "debug",
			},
			wantErr: false,
		},
		{
			name: "valid warn log_level",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 600,
				LogLevel:       "warn",
			},
			wantErr: false,
		},
		{
			name: "valid error log_level",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 600,
				LogLevel:       "error",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SupervisorConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSupervisorConfig_Timeout(t *testing.T) {
	cfg := &SupervisorConfig{
		TimeoutSeconds: 300,
	}

	timeout := cfg.Timeout()
	if timeout != 300000000000 { // 300 seconds in nanoseconds
		t.Errorf("Timeout() = %v, want 300s", timeout)
	}
}

func TestSupervisorConfig_GetResolvedPromptPath(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *SupervisorConfig
		wantErr bool
	}{
		{
			name: "default path with tilde",
			cfg: &SupervisorConfig{
				PromptPath: "~/.claude/SUPERVISOR.md",
			},
			wantErr: false,
		},
		{
			name: "absolute path",
			cfg: &SupervisorConfig{
				PromptPath: "/tmp/supervisor.md",
			},
			wantErr: false,
		},
		{
			name: "empty path uses default",
			cfg: &SupervisorConfig{
				PromptPath: "",
			},
			wantErr: false,
		},
		{
			name: "just tilde",
			cfg: &SupervisorConfig{
				PromptPath: "~",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.cfg.GetResolvedPromptPath()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetResolvedPromptPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && path == "" {
				t.Error("GetResolvedPromptPath() returned empty path")
			}
			if !tt.wantErr && path == tt.cfg.PromptPath && tt.cfg.PromptPath[0] == '~' {
				t.Error("GetResolvedPromptPath() did not expand tilde")
			}
		})
	}
}

func TestLoadSupervisorConfig_EnvOverride(t *testing.T) {
	// Save original env values
	origEnabled := os.Getenv("CCC_SUPERVISOR")
	origMaxIter := os.Getenv("CCC_SUPERVISOR_MAX_ITERATIONS")
	origTimeout := os.Getenv("CCC_SUPERVISOR_TIMEOUT")

	// Restore after test
	defer func() {
		if origEnabled == "" {
			os.Unsetenv("CCC_SUPERVISOR")
		} else {
			os.Setenv("CCC_SUPERVISOR", origEnabled)
		}
		if origMaxIter == "" {
			os.Unsetenv("CCC_SUPERVISOR_MAX_ITERATIONS")
		} else {
			os.Setenv("CCC_SUPERVISOR_MAX_ITERATIONS", origMaxIter)
		}
		if origTimeout == "" {
			os.Unsetenv("CCC_SUPERVISOR_TIMEOUT")
		} else {
			os.Setenv("CCC_SUPERVISOR_TIMEOUT", origTimeout)
		}
	}()

	// This test requires a valid config file to exist
	// For now, we skip if config doesn't exist
	t.Skip("requires valid ccc.json config file")

	// Test cases for env override would go here
	// Set env vars, call LoadSupervisorConfig, verify overrides
}

func TestSupervisorConfig_MarshalJSON(t *testing.T) {
	cfg := &SupervisorConfig{
		Enabled:        true,
		MaxIterations:  30,
		TimeoutSeconds: 900,
		PromptPath:     "~/.claude/SUPERVISOR.md",
		LogLevel:       "debug",
	}

	data, err := cfg.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf(".Unmarshal() error = %v", err)
	}

	// Verify fields
	if result["enabled"] != true {
		t.Errorf("enabled = %v, want true", result["enabled"])
	}
	// JSON numbers are float64
	if result["max_iterations"] != float64(30) {
		t.Errorf("max_iterations = %v, want 30", result["max_iterations"])
	}
}
