package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// setupTestDir creates a temporary directory and sets up getClaudeDirFunc to return it
// Returns the temp dir path and a cleanup function
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()

	// Save original function
	originalFunc := getClaudeDirFunc

	// Create temp directory
	tmpDir := t.TempDir()

	// Override getClaudeDirFunc
	getClaudeDirFunc = func() string {
		return tmpDir
	}

	// Return cleanup function
	cleanup := func() {
		getClaudeDirFunc = originalFunc
	}

	return tmpDir, cleanup
}

// writeJSONFile writes a JSON object to the specified file path
func writeJSONFile(t *testing.T, path string, data interface{}) {
	t.Helper()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	// Marshal and write
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// readJSONFile reads and unmarshals a JSON file
func readJSONFile(t *testing.T, path string) map[string]interface{} {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON from %s: %v", path, err)
	}

	return result
}

// compareJSON compares two JSON objects deeply
func compareJSON(t *testing.T, got, want map[string]interface{}) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		t.Errorf("JSON mismatch:\nGot:\n%s\n\nWant:\n%s", gotJSON, wantJSON)
	}
}

// TestCheckExistingSettings tests the checkExistingSettings function
func TestCheckExistingSettings(t *testing.T) {
	tests := []struct {
		name          string
		setupSettings bool // whether to create settings.json
		want          bool
	}{
		{
			name:          "settings.json exists",
			setupSettings: true,
			want:          true,
		},
		{
			name:          "settings.json does not exist",
			setupSettings: false,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := setupTestDir(t)
			defer cleanup()

			// Create settings.json if needed
			if tt.setupSettings {
				settingsPath := filepath.Join(tmpDir, "settings.json")
				writeJSONFile(t, settingsPath, map[string]interface{}{
					"permissions": map[string]interface{}{},
				})
			}

			// Test
			got := checkExistingSettings()
			if got != tt.want {
				t.Errorf("checkExistingSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPromptUserForMigration tests the promptUserForMigration function
func TestPromptUserForMigration(t *testing.T) {
	tests := []struct {
		name      string
		userInput string
		inputErr  error
		want      bool
	}{
		{
			name:      "user accepts with 'y'",
			userInput: "y\n",
			inputErr:  nil,
			want:      true,
		},
		{
			name:      "user accepts with 'yes'",
			userInput: "yes\n",
			inputErr:  nil,
			want:      true,
		},
		{
			name:      "user accepts with 'Y' (uppercase)",
			userInput: "Y\n",
			inputErr:  nil,
			want:      true,
		},
		{
			name:      "user accepts with 'YES' (uppercase)",
			userInput: "YES\n",
			inputErr:  nil,
			want:      true,
		},
		{
			name:      "user rejects with 'n'",
			userInput: "n\n",
			inputErr:  nil,
			want:      false,
		},
		{
			name:      "user rejects with 'no'",
			userInput: "no\n",
			inputErr:  nil,
			want:      false,
		},
		{
			name:      "user rejects with empty input",
			userInput: "\n",
			inputErr:  nil,
			want:      false,
		},
		{
			name:      "user rejects with random text",
			userInput: "maybe\n",
			inputErr:  nil,
			want:      false,
		},
		{
			name:      "input read error",
			userInput: "",
			inputErr:  errors.New("stdin closed"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTestDir(t)
			defer cleanup()

			// Save original function
			originalInputFunc := getUserInputFunc
			defer func() { getUserInputFunc = originalInputFunc }()

			// Mock getUserInputFunc
			getUserInputFunc = func(prompt string) (string, error) {
				if tt.inputErr != nil {
					return "", tt.inputErr
				}
				return tt.userInput, nil
			}

			// Test
			got := promptUserForMigration()
			if got != tt.want {
				t.Errorf("promptUserForMigration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMigrateFromSettings tests the migrateFromSettings function
func TestMigrateFromSettings(t *testing.T) {
	tests := []struct {
		name         string
		settingsData map[string]interface{}
		wantErr      bool
		errContains  string
		validate     func(t *testing.T, tmpDir string)
	}{
		{
			name: "standard migration with env",
			settingsData: map[string]interface{}{
				"permissions": map[string]interface{}{
					"allow": []interface{}{"*"},
				},
				"alwaysThinkingEnabled": true,
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://api.example.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-xxx",
					"ANTHROPIC_MODEL":      "claude-3",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string) {
				// Verify ccc.json was created correctly
				cccPath := filepath.Join(tmpDir, "ccc.json")
				got := readJSONFile(t, cccPath)

				want := map[string]interface{}{
					"settings": map[string]interface{}{
						"permissions": map[string]interface{}{
							"allow": []interface{}{"*"},
						},
						"alwaysThinkingEnabled": true,
					},
					"current_provider": "default",
					"providers": map[string]interface{}{
						"default": map[string]interface{}{
							"env": map[string]interface{}{
								"ANTHROPIC_BASE_URL":   "https://api.example.com",
								"ANTHROPIC_AUTH_TOKEN": "sk-xxx",
								"ANTHROPIC_MODEL":      "claude-3",
							},
						},
					},
				}

				compareJSON(t, got, want)

				// Verify settings.json was not modified
				settingsPath := filepath.Join(tmpDir, "settings.json")
				originalSettings := readJSONFile(t, settingsPath)
				if _, hasEnv := originalSettings["env"]; !hasEnv {
					t.Error("settings.json should still contain env field")
				}
			},
		},
		{
			name: "migration without env field",
			settingsData: map[string]interface{}{
				"permissions": map[string]interface{}{
					"allow": []interface{}{"*"},
				},
				"alwaysThinkingEnabled": true,
			},
			wantErr: false,
			validate: func(t *testing.T, tmpDir string) {
				cccPath := filepath.Join(tmpDir, "ccc.json")
				got := readJSONFile(t, cccPath)

				want := map[string]interface{}{
					"settings": map[string]interface{}{
						"permissions": map[string]interface{}{
							"allow": []interface{}{"*"},
						},
						"alwaysThinkingEnabled": true,
					},
					"current_provider": "default",
					"providers": map[string]interface{}{
						"default": map[string]interface{}{},
					},
				}

				compareJSON(t, got, want)
			},
		},
		{
			name:         "empty settings",
			settingsData: map[string]interface{}{},
			wantErr:      false,
			validate: func(t *testing.T, tmpDir string) {
				cccPath := filepath.Join(tmpDir, "ccc.json")
				got := readJSONFile(t, cccPath)

				want := map[string]interface{}{
					"settings":         map[string]interface{}{},
					"current_provider": "default",
					"providers": map[string]interface{}{
						"default": map[string]interface{}{},
					},
				}

				compareJSON(t, got, want)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := setupTestDir(t)
			defer cleanup()

			// Create settings.json
			settingsPath := filepath.Join(tmpDir, "settings.json")
			writeJSONFile(t, settingsPath, tt.settingsData)

			// Run migration
			err := migrateFromSettings()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("migrateFromSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("migrateFromSettings() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}

			// Run validation if provided
			if tt.validate != nil {
				tt.validate(t, tmpDir)
			}
		})
	}
}

// TestMigrateFromSettingsErrors tests error scenarios for migrateFromSettings
func TestMigrateFromSettingsErrors(t *testing.T) {
	t.Run("settings.json does not exist", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		// Don't create settings.json
		err := migrateFromSettings()

		if err == nil {
			t.Fatal("migrateFromSettings() should fail when settings.json doesn't exist")
		}

		if !strings.Contains(err.Error(), "failed to read settings file") {
			t.Errorf("Error should mention 'failed to read settings file', got: %v", err)
		}
	})

	t.Run("settings.json has invalid JSON", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create invalid JSON file
		settingsPath := filepath.Join(tmpDir, "settings.json")
		if err := os.WriteFile(settingsPath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		err := migrateFromSettings()

		if err == nil {
			t.Fatal("migrateFromSettings() should fail with invalid JSON")
		}

		if !strings.Contains(err.Error(), "failed to parse settings file") {
			t.Errorf("Error should mention 'failed to parse settings file', got: %v", err)
		}
	})
}

// TestMigrationFlowAccept tests the complete migration flow when user accepts
func TestMigrationFlowAccept(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Save original functions
	originalInputFunc := getUserInputFunc
	defer func() { getUserInputFunc = originalInputFunc }()

	// Mock user accepting migration
	getUserInputFunc = func(prompt string) (string, error) {
		return "y\n", nil
	}

	// Create settings.json
	settingsPath := filepath.Join(tmpDir, "settings.json")
	originalSettings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"*"},
		},
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL": "https://api.test.com",
		},
	}
	writeJSONFile(t, settingsPath, originalSettings)

	// Test the flow: check → prompt → migrate
	if !checkExistingSettings() {
		t.Fatal("checkExistingSettings() should return true")
	}

	if !promptUserForMigration() {
		t.Fatal("promptUserForMigration() should return true")
	}

	if err := migrateFromSettings(); err != nil {
		t.Fatalf("migrateFromSettings() failed: %v", err)
	}

	// Verify ccc.json was created
	cccPath := filepath.Join(tmpDir, "ccc.json")
	if _, err := os.Stat(cccPath); os.IsNotExist(err) {
		t.Fatal("ccc.json should exist after migration")
	}

	// Verify can load the config
	config, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() failed after migration: %v", err)
	}

	if config.CurrentProvider != "default" {
		t.Errorf("CurrentProvider = %q, want %q", config.CurrentProvider, "default")
	}

	// Verify settings.json was not modified
	currentSettings := readJSONFile(t, settingsPath)
	compareJSON(t, currentSettings, originalSettings)
}

// TestMigrationFlowReject tests the migration flow when user rejects
func TestMigrationFlowReject(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Save original functions
	originalInputFunc := getUserInputFunc
	defer func() { getUserInputFunc = originalInputFunc }()

	// Mock user rejecting migration
	getUserInputFunc = func(prompt string) (string, error) {
		return "n\n", nil
	}

	// Create settings.json
	settingsPath := filepath.Join(tmpDir, "settings.json")
	writeJSONFile(t, settingsPath, map[string]interface{}{
		"permissions": map[string]interface{}{},
	})

	// Test the flow
	if !checkExistingSettings() {
		t.Fatal("checkExistingSettings() should return true")
	}

	if promptUserForMigration() {
		t.Fatal("promptUserForMigration() should return false when user rejects")
	}

	// Verify ccc.json was NOT created
	cccPath := filepath.Join(tmpDir, "ccc.json")
	if _, err := os.Stat(cccPath); !os.IsNotExist(err) {
		t.Error("ccc.json should not exist when user rejects migration")
	}
}

// TestMigrationFlowErrors tests error handling in migration flow
func TestMigrationFlowErrors(t *testing.T) {
	t.Run("migration fails with invalid settings", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create invalid JSON
		settingsPath := filepath.Join(tmpDir, "settings.json")
		if err := os.WriteFile(settingsPath, []byte("not json"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Attempt migration
		err := migrateFromSettings()
		if err == nil {
			t.Fatal("migrateFromSettings() should fail with invalid JSON")
		}

		// Verify ccc.json was NOT created
		cccPath := filepath.Join(tmpDir, "ccc.json")
		if _, err := os.Stat(cccPath); !os.IsNotExist(err) {
			t.Error("ccc.json should not exist when migration fails")
		}
	})

	t.Run("check detects missing settings", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		// Don't create settings.json
		if checkExistingSettings() {
			t.Error("checkExistingSettings() should return false when settings.json doesn't exist")
		}
	})
}
