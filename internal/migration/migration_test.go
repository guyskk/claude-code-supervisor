package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/user/ccc/internal/config"
)

// setupTestDir creates a temporary directory for testing.
func setupTestDir(t *testing.T) func() {
	t.Helper()

	// Save original function
	originalFunc := config.GetDirFunc
	originalInputFunc := GetUserInputFunc

	// Create temp directory
	tmpDir := t.TempDir()

	// Override GetDirFunc
	config.GetDirFunc = func() string {
		return tmpDir
	}

	// Return cleanup function
	cleanup := func() {
		config.GetDirFunc = originalFunc
		GetUserInputFunc = originalInputFunc
	}

	return cleanup
}

// writeJSONFile writes a JSON object to the specified file path.
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

// readJSONFile reads and unmarshals a JSON file.
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

// compareJSON compares two JSON objects deeply.
func compareJSON(t *testing.T, got, want map[string]interface{}) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		t.Errorf("JSON mismatch:\nGot:\n%s\n\nWant:\n%s", gotJSON, wantJSON)
	}
}

func TestCheckExisting(t *testing.T) {
	tests := []struct {
		name          string
		setupSettings bool
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
			cleanup := setupTestDir(t)
			defer cleanup()

			// Create settings.json if needed
			if tt.setupSettings {
				settingsPath := config.GetSettingsPath("")
				writeJSONFile(t, settingsPath, map[string]interface{}{
					"permissions": map[string]interface{}{},
				})
			}

			// Test
			got := CheckExisting()
			if got != tt.want {
				t.Errorf("CheckExisting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPromptUser(t *testing.T) {
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
			inputErr:  os.ErrClosed,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestDir(t)
			defer cleanup()

			// Mock GetUserInputFunc
			GetUserInputFunc = func(prompt string) (string, error) {
				if tt.inputErr != nil {
					return "", tt.inputErr
				}
				return tt.userInput, nil
			}

			// Test
			got := PromptUser()
			if got != tt.want {
				t.Errorf("PromptUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
					"allow": []interface{}{"Edit", "Write"},
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
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Failed to load ccc.json: %v", err)
				}

				// Check settings
				env := config.GetEnv(cfg.Settings)
				if env != nil {
					t.Error("Settings should not contain env (should be in provider)")
				}

				// Check permissions exist in settings
				if _, exists := cfg.Settings["permissions"]; !exists {
					t.Error("Permissions should be in settings")
				}

				// Check alwaysThinkingEnabled exists
				if thinking, exists := cfg.Settings["alwaysThinkingEnabled"]; !exists || !thinking.(bool) {
					t.Error("alwaysThinkingEnabled should be true in settings")
				}

				// Check providers
				if len(cfg.Providers) != 1 {
					t.Errorf("Providers count = %d, want 1", len(cfg.Providers))
				}
				if cfg.CurrentProvider != "default" {
					t.Errorf("CurrentProvider = %s, want default", cfg.CurrentProvider)
				}

				defaultProvider := cfg.Providers["default"]
				defaultEnv := config.GetEnv(defaultProvider)
				if defaultEnv == nil {
					t.Error("Default provider env should not be nil")
				} else {
					if defaultEnv["ANTHROPIC_BASE_URL"] != "https://api.example.com" {
						t.Error("BASE_URL not migrated correctly")
					}
					if defaultEnv["ANTHROPIC_AUTH_TOKEN"] != "sk-xxx" {
						t.Error("AUTH_TOKEN not migrated correctly")
					}
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
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Failed to load ccc.json: %v", err)
				}

				if cfg.CurrentProvider != "default" {
					t.Errorf("CurrentProvider = %s, want default", cfg.CurrentProvider)
				}

				if len(cfg.Providers) != 1 {
					t.Errorf("Providers count = %d, want 1", len(cfg.Providers))
				}
			},
		},
		{
			name:         "empty settings",
			settingsData: map[string]interface{}{},
			wantErr:      false,
			validate: func(t *testing.T, tmpDir string) {
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Failed to load ccc.json: %v", err)
				}

				if cfg.CurrentProvider != "default" {
					t.Errorf("CurrentProvider = %s, want default", cfg.CurrentProvider)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestDir(t)
			defer cleanup()

			// Create settings.json
			settingsPath := config.GetSettingsPath("")
			writeJSONFile(t, settingsPath, tt.settingsData)

			// Run migration
			err := MigrateFromSettings()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("MigrateFromSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("MigrateFromSettings() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}

			// Run validation if provided
			if tt.validate != nil {
				tt.validate(t, config.GetDir())
			}
		})
	}
}

func TestMigrateFromSettingsErrors(t *testing.T) {
	t.Run("settings.json does not exist", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		// Don't create settings.json
		err := MigrateFromSettings()

		if err == nil {
			t.Fatal("MigrateFromSettings() should fail when settings.json doesn't exist")
		}

		if !strings.Contains(err.Error(), "failed to read settings file") {
			t.Errorf("Error should mention 'failed to read settings file', got: %v", err)
		}
	})

	t.Run("settings.json has invalid JSON", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		// Create invalid JSON file
		settingsPath := config.GetSettingsPath("")
		if err := os.WriteFile(settingsPath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		err := MigrateFromSettings()

		if err == nil {
			t.Fatal("MigrateFromSettings() should fail with invalid JSON")
		}

		if !strings.Contains(err.Error(), "failed to parse settings file") {
			t.Errorf("Error should mention 'failed to parse settings file', got: %v", err)
		}
	})
}

func TestMigrationFlowAccept(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Mock user accepting migration
	GetUserInputFunc = func(prompt string) (string, error) {
		return "y\n", nil
	}

	// Create settings.json
	settingsPath := config.GetSettingsPath("")
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
	if !CheckExisting() {
		t.Fatal("CheckExisting() should return true")
	}

	if !PromptUser() {
		t.Fatal("PromptUser() should return true")
	}

	if err := MigrateFromSettings(); err != nil {
		t.Fatalf("MigrateFromSettings() failed: %v", err)
	}

	// Verify ccc.json was created
	cccPath := config.GetConfigPath()
	if _, err := os.Stat(cccPath); os.IsNotExist(err) {
		t.Fatal("ccc.json should exist after migration")
	}

	// Verify can load the config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed after migration: %v", err)
	}

	if cfg.CurrentProvider != "default" {
		t.Errorf("CurrentProvider = %q, want %q", cfg.CurrentProvider, "default")
	}
}

func TestMigrationFlowReject(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Mock user rejecting migration
	GetUserInputFunc = func(prompt string) (string, error) {
		return "n\n", nil
	}

	// Create settings.json
	settingsPath := config.GetSettingsPath("")
	writeJSONFile(t, settingsPath, map[string]interface{}{
		"permissions": map[string]interface{}{},
	})

	// Test the flow
	if !CheckExisting() {
		t.Fatal("CheckExisting() should return true")
	}

	if PromptUser() {
		t.Fatal("PromptUser() should return false when user rejects")
	}

	// Verify ccc.json was NOT created
	cccPath := config.GetConfigPath()
	if _, err := os.Stat(cccPath); !os.IsNotExist(err) {
		t.Error("ccc.json should not exist when user rejects migration")
	}
}

func TestMigrationFlowErrors(t *testing.T) {
	t.Run("migration fails with invalid settings", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		// Create invalid JSON
		settingsPath := config.GetSettingsPath("")
		if err := os.WriteFile(settingsPath, []byte("not json"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Attempt migration
		err := MigrateFromSettings()
		if err == nil {
			t.Fatal("MigrateFromSettings() should fail with invalid JSON")
		}

		// Verify ccc.json was NOT created
		cccPath := config.GetConfigPath()
		if _, err := os.Stat(cccPath); !os.IsNotExist(err) {
			t.Error("ccc.json should not exist when migration fails")
		}
	})

	t.Run("check detects missing settings", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		// Don't create settings.json
		if CheckExisting() {
			t.Error("CheckExisting() should return false when settings.json doesn't exist")
		}
	})
}

func TestTrimToLower(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trim spaces",
			input: "  hello  ",
			want:  "hello",
		},
		{
			name:  "trim newline",
			input: "yes\n",
			want:  "yes",
		},
		{
			name:  "uppercase to lowercase",
			input: "YES",
			want:  "yes",
		},
		{
			name:  "mixed case",
			input: "  YeS  ",
			want:  "yes",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "  \t\n\r  ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimToLower(tt.input)
			if got != tt.want {
				t.Errorf("trimToLower() = %q, want %q", got, tt.want)
			}
		})
	}
}
