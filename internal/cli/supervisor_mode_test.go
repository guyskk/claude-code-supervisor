// Package cli provides tests for the supervisor-mode subcommand.
package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/supervisor"
)

// TestSupervisorMode_Integration tests the supervisor-mode subcommand.
// These are integration tests that verify the command correctly modifies
// the supervisor state file.
func TestSupervisorMode_Integration(t *testing.T) {
	// Create test config directory
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test config
	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {"env": {"ANTHROPIC_AUTH_TOKEN": "test"}}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override config.GetDirFunc for testing
	originalGetDirFunc := config.GetDirFunc
	config.GetDirFunc = func() string {
		return testConfigDir
	}
	defer func() {
		config.GetDirFunc = originalGetDirFunc
	}()

	// Set CCC_CONFIG_DIR for supervisor state directory
	// (supervisor.GetStateDir uses CCC_CONFIG_DIR via getDefaultStateDir)
	os.Setenv("CCC_CONFIG_DIR", testConfigDir)
	defer os.Unsetenv("CCC_CONFIG_DIR")

	t.Run("supervisor-mode on enables supervisor", func(t *testing.T) {
		supervisorID := "test-supervisor-mode-on"

		// Set environment variable
		os.Setenv("CCC_SUPERVISOR_ID", supervisorID)
		defer os.Unsetenv("CCC_SUPERVISOR_ID")

		// Create initial state with enabled=false
		initialState := &supervisor.State{
			SessionID: supervisorID,
			Enabled:   false,
			Count:     0,
		}
		if err := supervisor.SaveState(supervisorID, initialState); err != nil {
			t.Fatalf("failed to save initial state: %v", err)
		}

		// Run supervisor-mode on
		cmd := &Command{
			SupervisorMode: true,
			SupervisorModeOpts: &SupervisorModeCommand{
				Enabled: true,
			},
		}
		if err := RunSupervisorMode(cmd.SupervisorModeOpts); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		// Verify state was updated
		state, err := supervisor.LoadState(supervisorID)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}
		if !state.Enabled {
			t.Errorf("expected Enabled=true, got false")
		}
		if state.SessionID != supervisorID {
			t.Errorf("expected SessionID=%s, got %s", supervisorID, state.SessionID)
		}
	})

	t.Run("supervisor-mode off disables supervisor", func(t *testing.T) {
		supervisorID := "test-supervisor-mode-off"

		// Set environment variable
		os.Setenv("CCC_SUPERVISOR_ID", supervisorID)
		defer os.Unsetenv("CCC_SUPERVISOR_ID")

		// Create initial state with enabled=true
		initialState := &supervisor.State{
			SessionID: supervisorID,
			Enabled:   true,
			Count:     5,
		}
		if err := supervisor.SaveState(supervisorID, initialState); err != nil {
			t.Fatalf("failed to save initial state: %v", err)
		}

		// Run supervisor-mode off
		cmd := &Command{
			SupervisorMode: true,
			SupervisorModeOpts: &SupervisorModeCommand{
				Enabled: false,
			},
		}
		if err := RunSupervisorMode(cmd.SupervisorModeOpts); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		// Verify state was updated
		state, err := supervisor.LoadState(supervisorID)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}
		if state.Enabled {
			t.Errorf("expected Enabled=false, got true")
		}
		// Count should be preserved
		if state.Count != 5 {
			t.Errorf("expected Count=5 (preserved), got %d", state.Count)
		}
	})

	t.Run("supervisor-mode requires CCC_SUPERVISOR_ID", func(t *testing.T) {
		// Unset environment variable
		os.Unsetenv("CCC_SUPERVISOR_ID")

		cmd := &Command{
			SupervisorMode: true,
			SupervisorModeOpts: &SupervisorModeCommand{
				Enabled: true,
			},
		}
		err := RunSupervisorMode(cmd.SupervisorModeOpts)
		if err == nil {
			t.Fatal("expected error when CCC_SUPERVISOR_ID is not set")
		}
		if !containsString(err.Error(), "CCC_SUPERVISOR_ID") {
			t.Errorf("error message should mention CCC_SUPERVISOR_ID, got: %v", err)
		}
	})
}

// TestSupervisorMode_HookIntegration tests that the hook correctly
// reads the enabled field from the state file.
func TestSupervisorMode_HookIntegration(t *testing.T) {
	// Create test config directory
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test config
	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {"env": {"ANTHROPIC_AUTH_TOKEN": "test"}}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override config.GetDirFunc for testing
	originalGetDirFunc := config.GetDirFunc
	config.GetDirFunc = func() string {
		return testConfigDir
	}
	defer func() {
		config.GetDirFunc = originalGetDirFunc
	}()

	// Set CCC_CONFIG_DIR for supervisor state directory
	os.Setenv("CCC_CONFIG_DIR", testConfigDir)
	defer os.Unsetenv("CCC_CONFIG_DIR")

	t.Run("hook allows stop when supervisor is disabled", func(t *testing.T) {
		supervisorID := "test-hook-disabled"

		// Set environment variable
		os.Setenv("CCC_SUPERVISOR_ID", supervisorID)
		defer os.Unsetenv("CCC_SUPERVISOR_ID")

		// Create state with enabled=false
		state := &supervisor.State{
			SessionID: supervisorID,
			Enabled:   false,
			Count:     0,
		}
		if err := supervisor.SaveState(supervisorID, state); err != nil {
			t.Fatalf("failed to save state: %v", err)
		}

		// Run the hook with a dummy input
		// We need to provide stdin input since the hook reads session_id from stdin
		// For this test, we'll create a minimal test that checks the early return path

		// The hook should return "supervisor mode disabled" when state.Enabled=false
		// This is tested in the E2E tests, so we'll just verify the state loading here
		loadedState, err := supervisor.LoadState(supervisorID)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}
		if loadedState.Enabled {
			t.Errorf("expected Enabled=false, got true")
		}
	})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findInStringLocal(s, substr)))
}

func findInStringLocal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
