//go:build integration
// +build integration

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSupervisorConfig_Integration tests real config file loading scenarios
func TestSupervisorConfig_Integration(t *testing.T) {
	// Save and clear CCC_SUPERVISOR env var for clean testing
	origEnv := os.Getenv("CCC_SUPERVISOR")
	os.Unsetenv("CCC_SUPERVISOR")
	defer func() {
		if origEnv != "" {
			os.Setenv("CCC_SUPERVISOR", origEnv)
		}
	}()

	testCases := []struct {
		name        string
		configJSON  string
		wantEnabled bool
		wantMaxIter int
		wantTimeout int
	}{
		{
			name: "with_supervisor_at_top_level",
			configJSON: `{
				"settings": {},
				"supervisor": {
					"enabled": true,
					"max_iterations": 20,
					"timeout_seconds": 600
				},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: true,
			wantMaxIter: 20,
			wantTimeout: 600,
		},
		{
			name: "no_supervisor_config",
			configJSON: `{
				"settings": {},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: false,
			wantMaxIter: 20,  // defaults
			wantTimeout: 600, // defaults
		},
		{
			name: "partial_supervisor_config_only_enabled",
			configJSON: `{
				"settings": {},
				"supervisor": {
					"enabled": true
				},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: true,
			wantMaxIter: 20,  // defaults
			wantTimeout: 600, // defaults
		},
		{
			name: "partial_supervisor_custom_max_iterations",
			configJSON: `{
				"settings": {},
				"supervisor": {
					"max_iterations": 10
				},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: false,
			wantMaxIter: 10,  // custom value
			wantTimeout: 600, // defaults
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original GetDirFunc
			origGetDirFunc := GetDirFunc

			// Create temporary config dir
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "ccc.json")

			// Write test config to temp dir
			if err := os.WriteFile(configPath, []byte(tc.configJSON), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			// Override GetDirFunc to use temp dir
			GetDirFunc = func() string { return tmpDir }
			defer func() { GetDirFunc = origGetDirFunc }()

			// Load supervisor config
			supervisorCfg, err := LoadSupervisorConfig()
			if err != nil {
				t.Fatalf("LoadSupervisorConfig() error = %v", err)
			}

			// Check values
			if supervisorCfg.Enabled != tc.wantEnabled {
				t.Errorf("Enabled = %v, want %v", supervisorCfg.Enabled, tc.wantEnabled)
			}
			if supervisorCfg.MaxIterations != tc.wantMaxIter {
				t.Errorf("MaxIterations = %v, want %v", supervisorCfg.MaxIterations, tc.wantMaxIter)
			}
			if supervisorCfg.TimeoutSeconds != tc.wantTimeout {
				t.Errorf("TimeoutSeconds = %v, want %v", supervisorCfg.TimeoutSeconds, tc.wantTimeout)
			}
		})
	}
}

// TestSupervisorConfig_NilHandling tests nil supervisor config handling
func TestSupervisorConfig_NilHandling(t *testing.T) {
	// Save and clear CCC_SUPERVISOR env var for clean testing
	origEnv := os.Getenv("CCC_SUPERVISOR")
	os.Unsetenv("CCC_SUPERVISOR")
	defer func() {
		if origEnv != "" {
			os.Setenv("CCC_SUPERVISOR", origEnv)
		}
	}()

	// Save original GetDirFunc
	origGetDirFunc := GetDirFunc

	// Create minimal config without supervisor field
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "ccc.json")

	minimalConfig := `{
		"settings": {},
		"current_provider": "kimi",
		"providers": {}
	}`

	if err := os.WriteFile(configPath, []byte(minimalConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	GetDirFunc = func() string { return tmpDir }
	defer func() { GetDirFunc = origGetDirFunc }()

	// Should not panic and should return defaults
	supervisorCfg, err := LoadSupervisorConfig()
	if err != nil {
		t.Fatalf("LoadSupervisorConfig() error = %v", err)
	}

	// Verify defaults are used
	if supervisorCfg.Enabled != false {
		t.Errorf("Enabled = %v, want false (default)", supervisorCfg.Enabled)
	}
	if supervisorCfg.MaxIterations != 20 {
		t.Errorf("MaxIterations = %v, want 20 (default)", supervisorCfg.MaxIterations)
	}
	if supervisorCfg.TimeoutSeconds != 600 {
		t.Errorf("TimeoutSeconds = %v, want 600 (default)", supervisorCfg.TimeoutSeconds)
	}
}

// TestSupervisorConfig_EdgeCases tests edge cases and boundary conditions
func TestSupervisorConfig_EdgeCases(t *testing.T) {
	// Save and clear CCC_SUPERVISOR env var for clean testing
	origEnv := os.Getenv("CCC_SUPERVISOR")
	os.Unsetenv("CCC_SUPERVISOR")
	defer func() {
		if origEnv != "" {
			os.Setenv("CCC_SUPERVISOR", origEnv)
		}
	}()

	// Save original GetDirFunc
	origGetDirFunc := GetDirFunc
	defer func() { GetDirFunc = origGetDirFunc }()

	testCases := []struct {
		name        string
		configJSON  string
		wantEnabled bool
		wantMaxIter int
		wantTimeout int
	}{
		{
			name: "empty_supervisor_object_uses_defaults",
			configJSON: `{
				"settings": {},
				"supervisor": {},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: false, // default
			wantMaxIter: 20,    // default (empty object means no values set)
			wantTimeout: 600,   // default
		},
		{
			name: "zero_values_in_config_uses_defaults",
			configJSON: `{
				"settings": {},
				"supervisor": {
					"enabled": false,
					"max_iterations": 0,
					"timeout_seconds": 0
				},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: false, // false is default
			wantMaxIter: 20,    // 0 is invalid, uses default
			wantTimeout: 600,   // 0 is invalid, uses default
		},
		{
			name: "only_timeout_customized",
			configJSON: `{
				"settings": {},
				"supervisor": {
					"timeout_seconds": 120
				},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: false, // default
			wantMaxIter: 20,    // default
			wantTimeout: 120,   // custom value
		},
		{
			name: "all_fields_customized",
			configJSON: `{
				"settings": {},
				"supervisor": {
					"enabled": true,
					"max_iterations": 50,
					"timeout_seconds": 900
				},
				"current_provider": "kimi",
				"providers": {}
			}`,
			wantEnabled: true,
			wantMaxIter: 50,
			wantTimeout: 900,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary config dir
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "ccc.json")

			// Write test config to temp dir
			if err := os.WriteFile(configPath, []byte(tc.configJSON), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			// Override GetDirFunc to use temp dir
			GetDirFunc = func() string { return tmpDir }

			// Load supervisor config
			supervisorCfg, err := LoadSupervisorConfig()
			if err != nil {
				t.Fatalf("LoadSupervisorConfig() error = %v", err)
			}

			// Check values
			if supervisorCfg.Enabled != tc.wantEnabled {
				t.Errorf("Enabled = %v, want %v", supervisorCfg.Enabled, tc.wantEnabled)
			}
			if supervisorCfg.MaxIterations != tc.wantMaxIter {
				t.Errorf("MaxIterations = %v, want %v", supervisorCfg.MaxIterations, tc.wantMaxIter)
			}
			if supervisorCfg.TimeoutSeconds != tc.wantTimeout {
				t.Errorf("TimeoutSeconds = %v, want %v", supervisorCfg.TimeoutSeconds, tc.wantTimeout)
			}
		})
	}
}

// TestSupervisorConfig_EnvironmentVariableOverride tests env var override
func TestSupervisorConfig_EnvironmentVariableOverride(t *testing.T) {
	// Save original GetDirFunc
	origGetDirFunc := GetDirFunc
	defer func() { GetDirFunc = origGetDirFunc }()

	testCases := []struct {
		name        string
		configJSON  string
		envVar      string
		wantEnabled bool
		description string
	}{
		{
			name: "env_var_1_enables_supervisor",
			configJSON: `{
				"settings": {},
				"current_provider": "kimi",
				"providers": {}
			}`,
			envVar:      "1",
			wantEnabled: true,
			description: "CCC_SUPERVISOR=1 should enable supervisor",
		},
		{
			name: "env_var_true_enables_supervisor",
			configJSON: `{
				"settings": {},
				"current_provider": "kimi",
				"providers": {}
			}`,
			envVar:      "true",
			wantEnabled: true,
			description: "CCC_SUPERVISOR=true should enable supervisor",
		},
		{
			name: "env_var_0_disables_supervisor",
			configJSON: `{
				"settings": {},
				"supervisor": {"enabled": true},
				"current_provider": "kimi",
				"providers": {}
			}`,
			envVar:      "0",
			wantEnabled: false,
			description: "CCC_SUPERVISOR=0 should disable supervisor even if config has enabled=true",
		},
		{
			name: "env_var_false_disables_supervisor",
			configJSON: `{
				"settings": {},
				"supervisor": {"enabled": true},
				"current_provider": "kimi",
				"providers": {}
			}`,
			envVar:      "false",
			wantEnabled: false,
			description: "CCC_SUPERVISOR=false should disable supervisor even if config has enabled=true",
		},
		{
			name: "env_var_random_does_not_enable",
			configJSON: `{
				"settings": {},
				"current_provider": "kimi",
				"providers": {}
			}`,
			envVar:      "random",
			wantEnabled: false,
			description: "CCC_SUPERVISOR=random should not enable supervisor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear env var first
			os.Unsetenv("CCC_SUPERVISOR")

			// Create temporary config dir
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "ccc.json")

			// Write test config to temp dir
			if err := os.WriteFile(configPath, []byte(tc.configJSON), 0644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			// Set env var
			os.Setenv("CCC_SUPERVISOR", tc.envVar)
			defer os.Unsetenv("CCC_SUPERVISOR")

			// Override GetDirFunc to use temp dir
			GetDirFunc = func() string { return tmpDir }

			// Load supervisor config
			supervisorCfg, err := LoadSupervisorConfig()
			if err != nil {
				t.Fatalf("LoadSupervisorConfig() error = %v", err)
			}

			// Check enabled value
			if supervisorCfg.Enabled != tc.wantEnabled {
				t.Errorf("%s: Enabled = %v, want %v", tc.description, supervisorCfg.Enabled, tc.wantEnabled)
			}
		})
	}
}
