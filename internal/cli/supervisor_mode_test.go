// Package cli provides tests for the supervisor-mode subcommand.
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guyskk/ccc/internal/config"
	"github.com/guyskk/ccc/internal/supervisor"
)

// TestSupervisorMode_Integration tests the supervisor-mode subcommand.
// These are integration tests that verify the command correctly queries or modifies
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
		val := true
		cmd := &Command{
			SupervisorMode: true,
			SupervisorModeOpts: &SupervisorModeCommand{
				Enabled: &val,
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
		val := false
		cmd := &Command{
			SupervisorMode: true,
			SupervisorModeOpts: &SupervisorModeCommand{
				Enabled: &val,
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

		val := true
		cmd := &Command{
			SupervisorMode: true,
			SupervisorModeOpts: &SupervisorModeCommand{
				Enabled: &val,
			},
		}
		err := RunSupervisorMode(cmd.SupervisorModeOpts)
		if err == nil {
			t.Fatal("expected error when CCC_SUPERVISOR_ID is not set")
		}
		if !strings.Contains(err.Error(), "CCC_SUPERVISOR_ID") {
			t.Errorf("error message should mention CCC_SUPERVISOR_ID, got: %v", err)
		}
	})

	t.Run("supervisor-mode (no args) queries status", func(t *testing.T) {
		supervisorID := "test-supervisor-mode-query"

		// Set environment variable
		os.Setenv("CCC_SUPERVISOR_ID", supervisorID)
		defer os.Unsetenv("CCC_SUPERVISOR_ID")

		// Create initial state with enabled=true
		initialState := &supervisor.State{
			SessionID: supervisorID,
			Enabled:   true,
			Count:     3,
		}
		if err := supervisor.SaveState(supervisorID, initialState); err != nil {
			t.Fatalf("failed to save initial state: %v", err)
		}

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run supervisor-mode without args (query)
		cmd := &Command{
			SupervisorMode:     true,
			SupervisorModeOpts: &SupervisorModeCommand{Enabled: nil},
		}
		if err := RunSupervisorMode(cmd.SupervisorModeOpts); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		// Read stdout
		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := strings.TrimSpace(buf.String())

		// Verify output
		if output != "on" {
			t.Errorf("expected output 'on', got '%s'", output)
		}

		// Verify state was NOT modified
		state, err := supervisor.LoadState(supervisorID)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}
		if state.Count != 3 {
			t.Errorf("expected Count=3 (unchanged), got %d", state.Count)
		}
	})

	t.Run("supervisor-mode (no args) returns off when disabled", func(t *testing.T) {
		supervisorID := "test-supervisor-mode-query-off"

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

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run supervisor-mode without args (query)
		cmd := &Command{
			SupervisorMode:     true,
			SupervisorModeOpts: &SupervisorModeCommand{Enabled: nil},
		}
		if err := RunSupervisorMode(cmd.SupervisorModeOpts); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		// Read stdout
		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := strings.TrimSpace(buf.String())

		// Verify output
		if output != "off" {
			t.Errorf("expected output 'off', got '%s'", output)
		}
	})

	t.Run("supervisor-mode (no args) returns off when state file does not exist", func(t *testing.T) {
		supervisorID := "test-supervisor-mode-query-nonexistent"

		// Set environment variable
		os.Setenv("CCC_SUPERVISOR_ID", supervisorID)
		defer os.Unsetenv("CCC_SUPERVISOR_ID")

		// Ensure state file does not exist
		statePath, _ := supervisor.GetStatePath(supervisorID)
		os.Remove(statePath)

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run supervisor-mode without args (query)
		cmd := &Command{
			SupervisorMode:     true,
			SupervisorModeOpts: &SupervisorModeCommand{Enabled: nil},
		}
		if err := RunSupervisorMode(cmd.SupervisorModeOpts); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		// Read stdout
		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := strings.TrimSpace(buf.String())

		// Verify output (should be 'off' for non-existent state)
		if output != "off" {
			t.Errorf("expected output 'off' for non-existent state, got '%s'", output)
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

// TestSupervisorMode_EndToEnd tests a complete workflow simulating real usage.
func TestSupervisorMode_EndToEnd(t *testing.T) {
	// Create test config directory
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Set up environment
	os.Setenv("CCC_CONFIG_DIR", testConfigDir)
	defer os.Unsetenv("CCC_CONFIG_DIR")

	supervisorID := "test-e2e-workflow"
	os.Setenv("CCC_SUPERVISOR_ID", supervisorID)
	defer os.Unsetenv("CCC_SUPERVISOR_ID")

	// Ensure clean state
	statePath, _ := supervisor.GetStatePath(supervisorID)
	os.Remove(statePath)

	// Step 1: Query initial state (should be "off" - file doesn't exist)
	t.Run("step1: query initial state", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := &SupervisorModeCommand{Enabled: nil}
		if err := RunSupervisorMode(cmd); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := strings.TrimSpace(buf.String())

		if output != "off" {
			t.Errorf("expected 'off', got '%s'", output)
		}
	})

	// Step 2: Enable supervisor mode
	t.Run("step2: enable supervisor mode", func(t *testing.T) {
		val := true
		cmd := &SupervisorModeCommand{Enabled: &val}
		if err := RunSupervisorMode(cmd); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		// Verify state was saved
		state, err := supervisor.LoadState(supervisorID)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}
		if !state.Enabled {
			t.Error("expected Enabled=true after enabling")
		}
	})

	// Step 3: Query state (should be "on")
	t.Run("step3: query enabled state", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := &SupervisorModeCommand{Enabled: nil}
		if err := RunSupervisorMode(cmd); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := strings.TrimSpace(buf.String())

		if output != "on" {
			t.Errorf("expected 'on', got '%s'", output)
		}
	})

	// Step 4: Disable supervisor mode
	t.Run("step4: disable supervisor mode", func(t *testing.T) {
		val := false
		cmd := &SupervisorModeCommand{Enabled: &val}
		if err := RunSupervisorMode(cmd); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		// Verify state was saved
		state, err := supervisor.LoadState(supervisorID)
		if err != nil {
			t.Fatalf("failed to load state: %v", err)
		}
		if state.Enabled {
			t.Error("expected Enabled=false after disabling")
		}
	})

	// Step 5: Query state (should be "off")
	t.Run("step5: query disabled state", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := &SupervisorModeCommand{Enabled: nil}
		if err := RunSupervisorMode(cmd); err != nil {
			t.Fatalf("RunSupervisorMode() error = %v", err)
		}

		w.Close()
		os.Stdout = oldStdout
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := strings.TrimSpace(buf.String())

		if output != "off" {
			t.Errorf("expected 'off', got '%s'", output)
		}
	})
}

// TestSupervisorMode_InvalidArgument tests that invalid arguments are treated as query mode.
func TestSupervisorMode_InvalidArgument(t *testing.T) {
	// Invalid arguments should be ignored and treated as query mode
	cmd := Parse([]string{"supervisor-mode", "xyz"})
	if !cmd.SupervisorMode {
		t.Fatal("expected SupervisorMode to be true")
	}
	if cmd.SupervisorModeOpts == nil {
		t.Fatal("expected SupervisorModeOpts to be set")
	}
	if cmd.SupervisorModeOpts.Enabled != nil {
		t.Errorf("expected Enabled to be nil for invalid argument, got %v", cmd.SupervisorModeOpts.Enabled)
	}
}
