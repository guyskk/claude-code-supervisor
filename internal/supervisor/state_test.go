// Package supervisor provides unit tests for state management.
package supervisor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGetStateDir tests the GetStateDir function.
func TestGetStateDir(t *testing.T) {
	tests := []struct {
		name         string
		configDir    string
		wantContains string
	}{
		{
			name:         "default home directory",
			configDir:    "",
			wantContains: ".claude",
		},
		{
			name:         "custom config directory",
			configDir:    "/tmp/test-config",
			wantContains: "ccc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.configDir != "" {
				t.Setenv("CCC_CONFIG_DIR", tt.configDir)
			} else {
				// Clear the env for default case
				os.Unsetenv("CCC_CONFIG_DIR")
			}

			got, err := GetStateDir()
			if err != nil {
				t.Fatalf("GetStateDir() error = %v", err)
			}
			if filepath.Base(got) != "ccc" {
				t.Errorf("GetStateDir() basename = %s, want ccc", filepath.Base(got))
			}
		})
	}
}

// TestGetStatePath tests the GetStatePath function.
func TestGetStatePath(t *testing.T) {
	sessionID := "test-session-123"

	path, err := GetStatePath(sessionID)
	if err != nil {
		t.Fatalf("GetStatePath() error = %v", err)
	}

	expectedFilename := "supervisor-test-session-123.json"
	if filepath.Base(path) != expectedFilename {
		t.Errorf("GetStatePath() filename = %s, want %s", filepath.Base(path), expectedFilename)
	}
}

// TestLoadState tests the LoadState function.
func TestLoadState(t *testing.T) {
	t.Run("new state when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CCC_CONFIG_DIR", tmpDir)

		sessionID := "new-session-test"
		state, err := LoadState(sessionID)
		if err != nil {
			t.Fatalf("LoadState() error = %v", err)
		}

		if state.SessionID != sessionID {
			t.Errorf("LoadState() SessionID = %s, want %s", state.SessionID, sessionID)
		}
		if state.Count != 0 {
			t.Errorf("LoadState() Count = %d, want 0", state.Count)
		}
		if state.CreatedAt.IsZero() {
			t.Error("LoadState() CreatedAt should not be zero")
		}
	})

	t.Run("load existing state", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CCC_CONFIG_DIR", tmpDir)

		sessionID := "existing-session-test"

		// Create a state file
		expectedState := &State{
			SessionID: sessionID,
			Count:     5,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now(),
		}

		statePath, err := GetStatePath(sessionID)
		if err != nil {
			t.Fatalf("GetStatePath() error = %v", err)
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		// Write the state file using SaveState
		if err := SaveState(sessionID, expectedState); err != nil {
			t.Fatalf("SaveState() error = %v", err)
		}

		// Load the state
		loadedState, err := LoadState(sessionID)
		if err != nil {
			t.Fatalf("LoadState() error = %v", err)
		}

		if loadedState.SessionID != expectedState.SessionID {
			t.Errorf("LoadState() SessionID = %s, want %s", loadedState.SessionID, expectedState.SessionID)
		}
		if loadedState.Count != expectedState.Count {
			t.Errorf("LoadState() Count = %d, want %d", loadedState.Count, expectedState.Count)
		}
	})
}

// TestSaveState tests the SaveState function.
func TestSaveState(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CCC_CONFIG_DIR", tmpDir)

	sessionID := "save-session-test"
	state := &State{
		SessionID: sessionID,
		Count:     3,
	}

	if err := SaveState(sessionID, state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Verify the file was created
	statePath, err := GetStatePath(sessionID)
	if err != nil {
		t.Fatalf("GetStatePath() error = %v", err)
	}

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("SaveState() did not create state file")
	}

	// Verify we can load it back
	loadedState, err := LoadState(sessionID)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	if loadedState.Count != 3 {
		t.Errorf("LoadState() Count = %d, want 3", loadedState.Count)
	}
}

// TestIncrementCount tests the IncrementCount function.
func TestIncrementCount(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CCC_CONFIG_DIR", tmpDir)

	sessionID := "increment-session-test"

	// First increment
	count1, err := IncrementCount(sessionID)
	if err != nil {
		t.Fatalf("IncrementCount() error = %v", err)
	}
	if count1 != 1 {
		t.Errorf("IncrementCount() count = %d, want 1", count1)
	}

	// Second increment
	count2, err := IncrementCount(sessionID)
	if err != nil {
		t.Fatalf("IncrementCount() error = %v", err)
	}
	if count2 != 2 {
		t.Errorf("IncrementCount() count = %d, want 2", count2)
	}

	// Verify via GetCount
	getCount, err := GetCount(sessionID)
	if err != nil {
		t.Fatalf("GetCount() error = %v", err)
	}
	if getCount != 2 {
		t.Errorf("GetCount() count = %d, want 2", getCount)
	}
}

// TestGetCount tests the GetCount function.
func TestGetCount(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CCC_CONFIG_DIR", tmpDir)

	sessionID := "getcount-session-test"

	// New state should have count 0
	count, err := GetCount(sessionID)
	if err != nil {
		t.Fatalf("GetCount() error = %v", err)
	}
	if count != 0 {
		t.Errorf("GetCount() count = %d, want 0", count)
	}
}

// TestShouldContinue tests the ShouldContinue function.
func TestShouldContinue(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CCC_CONFIG_DIR", tmpDir)

	t.Run("should continue when count < max", func(t *testing.T) {
		sessionID := "continue-session-test"

		shouldContinue, count, err := ShouldContinue(sessionID, 10)
		if err != nil {
			t.Fatalf("ShouldContinue() error = %v", err)
		}
		if !shouldContinue {
			t.Error("ShouldContinue() shouldContinue = false, want true")
		}
		if count != 0 {
			t.Errorf("ShouldContinue() count = %d, want 0", count)
		}
	})

	t.Run("should not continue when count >= max", func(t *testing.T) {
		sessionID := "stop-session-test"

		// Set count to 10
		for i := 0; i < 10; i++ {
			IncrementCount(sessionID)
		}

		shouldContinue, count, err := ShouldContinue(sessionID, 10)
		if err != nil {
			t.Fatalf("ShouldContinue() error = %v", err)
		}
		if shouldContinue {
			t.Error("ShouldContinue() shouldContinue = true, want false")
		}
		if count != 10 {
			t.Errorf("ShouldContinue() count = %d, want 10", count)
		}
	})
}

// TestStateTimestamps tests that state timestamps are properly set.
func TestStateTimestamps(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CCC_CONFIG_DIR", tmpDir)

	sessionID := "timestamp-session-test"

	before := time.Now()
	state := &State{
		SessionID: sessionID,
		Count:     1,
	}
	if err := SaveState(sessionID, state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	after := time.Now()

	// Load the state
	loadedState, err := LoadState(sessionID)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	// Check CreatedAt was set
	if loadedState.CreatedAt.Before(before) || loadedState.CreatedAt.After(after) {
		t.Error("SaveState() CreatedAt not set correctly")
	}

	// Check UpdatedAt was set
	if loadedState.UpdatedAt.Before(before) || loadedState.UpdatedAt.After(after) {
		t.Error("SaveState() UpdatedAt not set correctly")
	}

	// Wait a bit and update
	time.Sleep(10 * time.Millisecond)
	updateTime := time.Now()
	loadedState.Count = 2
	if err := SaveState(sessionID, loadedState); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Load again
	reloadedState, err := LoadState(sessionID)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	// CreatedAt should not change
	if !reloadedState.CreatedAt.Equal(loadedState.CreatedAt) {
		t.Error("SaveState() CreatedAt changed on update")
	}

	// UpdatedAt should be updated
	if reloadedState.UpdatedAt.Before(updateTime) {
		t.Error("SaveState() UpdatedAt not updated")
	}
}
