// Package supervisor provides state management for supervisor mode.
package supervisor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// getStateDirFunc is a function that returns the state directory.
// This is a variable to allow testing to override it.
var getStateDirFunc = getDefaultStateDir

// getDefaultStateDir returns the default state directory.
func getDefaultStateDir() (string, error) {
	if configDir := os.Getenv("CCC_CONFIG_DIR"); configDir != "" {
		return filepath.Join(configDir, "ccc"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".claude", "ccc"), nil
}

// State represents the supervisor state for a session.
type State struct {
	SessionID string    `json:"session_id"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DefaultMaxIterations is the maximum number of supervisor iterations before allowing stop.
// Updated from 10 to 20 to give supervisor more chances to complete the task.
const DefaultMaxIterations = 20

// GetStateDir returns the directory for supervisor state files.
func GetStateDir() (string, error) {
	return getStateDirFunc()
}

// GetStatePath returns the path to the state file for a given session.
func GetStatePath(sessionID string) (string, error) {
	stateDir, err := GetStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, fmt.Sprintf("supervisor-%s.json", sessionID)), nil
}

// LoadState loads the supervisor state for a given session.
// If the state file doesn't exist, returns a new state with count 0.
func LoadState(sessionID string) (*State, error) {
	statePath, err := GetStatePath(sessionID)
	if err != nil {
		return nil, err
	}

	// Try to read existing state
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing state, return new state
			return &State{
				SessionID: sessionID,
				Count:     0,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// SaveState saves the supervisor state for a given session.
func SaveState(sessionID string, state *State) error {
	statePath, err := GetStatePath(sessionID)
	if err != nil {
		return err
	}

	// Ensure state directory exists
	stateDir, err := GetStateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Update timestamp
	state.UpdatedAt = time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}

	// Marshal and write
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// IncrementCount increments the iteration count for a session.
// Returns the new count and any error.
func IncrementCount(sessionID string) (int, error) {
	state, err := LoadState(sessionID)
	if err != nil {
		return 0, err
	}

	state.Count++
	if err := SaveState(sessionID, state); err != nil {
		return 0, err
	}

	return state.Count, nil
}

// GetCount returns the current iteration count for a session.
func GetCount(sessionID string) (int, error) {
	state, err := LoadState(sessionID)
	if err != nil {
		return 0, err
	}
	return state.Count, nil
}

// ShouldContinue returns true if the supervisor should continue (count < max).
func ShouldContinue(sessionID string, max int) (bool, int, error) {
	count, err := GetCount(sessionID)
	if err != nil {
		return false, 0, err
	}
	return count < max, count, nil
}

// GetLogFilePath returns the path to the supervisor log file for a given supervisorID.
func GetLogFilePath(supervisorID string) (string, error) {
	stateDir, err := GetStateDir()
	if err != nil {
		return "", fmt.Errorf("failed to get state directory: %w", err)
	}
	return filepath.Join(stateDir, fmt.Sprintf("supervisor-%s.log", supervisorID)), nil
}

// OpenLogFile opens and returns the supervisor log file for a given supervisorID.
// It creates the state directory if it doesn't exist.
// The caller is responsible for closing the file.
func OpenLogFile(supervisorID string) (*os.File, error) {
	stateDir, err := GetStateDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get state directory: %w", err)
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	logFilePath, err := GetLogFilePath(supervisorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get log file path: %w", err)
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return logFile, nil
}
