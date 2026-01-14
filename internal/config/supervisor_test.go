// Package config provides tests for supervisor configuration.
package config

import (
	"encoding/json"
	"testing"
)

func TestDefaultSupervisorConfig(t *testing.T) {
	cfg := DefaultSupervisorConfig()

	if cfg.MaxIterations != 20 {
		t.Errorf("default MaxIterations = %d, want 20", cfg.MaxIterations)
	}
	if cfg.TimeoutSeconds != 600 {
		t.Errorf("default TimeoutSeconds = %d, want 600", cfg.TimeoutSeconds)
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
			},
			wantErr: false,
		},
		{
			name: "max_iterations too low",
			cfg: &SupervisorConfig{
				MaxIterations:  0,
				TimeoutSeconds: 600,
			},
			wantErr: true,
		},
		{
			name: "max_iterations too high",
			cfg: &SupervisorConfig{
				MaxIterations:  101,
				TimeoutSeconds: 600,
			},
			wantErr: true,
		},
		{
			name: "timeout_seconds too low",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 5,
			},
			wantErr: true,
		},
		{
			name: "timeout_seconds too high",
			cfg: &SupervisorConfig{
				MaxIterations:  20,
				TimeoutSeconds: 4000,
			},
			wantErr: true,
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

func TestSupervisorConfig_MarshalJSON(t *testing.T) {
	cfg := &SupervisorConfig{
		MaxIterations:  30,
		TimeoutSeconds: 900,
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
	// JSON numbers are float64
	if result["max_iterations"] != float64(30) {
		t.Errorf("max_iterations = %v, want 30", result["max_iterations"])
	}
	if result["timeout_seconds"] != float64(900) {
		t.Errorf("timeout_seconds = %v, want 900", result["timeout_seconds"])
	}

	// Verify enabled is not included
	if _, exists := result["enabled"]; exists {
		t.Error("enabled should not be included in JSON")
	}
}
