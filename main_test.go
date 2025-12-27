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

// TestValidateProvider tests the validateProvider function
func TestValidateProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		provider  string
		testAPI   bool
		wantValid bool
		wantErrs  []string
	}{
		{
			name: "valid provider with all required fields",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"kimi": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
							"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
							"ANTHROPIC_MODEL":      "claude-3-5-sonnet-20241022",
						},
					},
				},
			},
			provider:  "kimi",
			testAPI:   false,
			wantValid: true,
			wantErrs:  nil,
		},
		{
			name: "provider missing ANTHROPIC_BASE_URL",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"glm": {
						"env": map[string]interface{}{
							"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
						},
					},
				},
			},
			provider:  "glm",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Missing required environment variable: ANTHROPIC_BASE_URL"},
		},
		{
			name: "provider missing ANTHROPIC_AUTH_TOKEN",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"m2": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
						},
					},
				},
			},
			provider:  "m2",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Missing required environment variable: ANTHROPIC_AUTH_TOKEN"},
		},
		{
			name: "provider with invalid URL format",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"broken": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "not-a-valid-url",
							"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
						},
					},
				},
			},
			provider:  "broken",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Invalid Base URL format: must use http:// or https:// scheme"},
		},
		{
			name: "provider not found",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"kimi": {},
				},
			},
			provider:  "unknown",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Provider 'unknown' not found in configuration"},
		},
		{
			name: "provider with minimal valid config",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"minimal": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.example.com",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
				},
			},
			provider:  "minimal",
			testAPI:   false,
			wantValid: true,
			wantErrs:  nil,
		},
		{
			name: "provider without env field",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"noenv": {},
				},
			},
			provider:  "noenv",
			testAPI:   false,
			wantValid: false,
			wantErrs: []string{
				"Missing required environment variable: ANTHROPIC_BASE_URL",
				"Missing required environment variable: ANTHROPIC_AUTH_TOKEN",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateProvider(tt.config, tt.provider, tt.testAPI)

			if result.Valid != tt.wantValid {
				t.Errorf("validateProvider() Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if len(result.Errors) != len(tt.wantErrs) {
				t.Errorf("validateProvider() got %d errors, want %d", len(result.Errors), len(tt.wantErrs))
			}

			for i, wantErr := range tt.wantErrs {
				if i >= len(result.Errors) {
					t.Errorf("validateProvider() missing expected error %d: %q", i, wantErr)
					continue
				}
				if !strings.Contains(result.Errors[i], wantErr) {
					t.Errorf("validateProvider() error %d = %q, want to contain %q", i, result.Errors[i], wantErr)
				}
			}
		})
	}
}

// TestValidateAllProviders tests the validateAllProviders function
func TestValidateAllProviders(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		testAPI     bool
		wantTotal   int
		wantValid   int
		wantInvalid int
	}{
		{
			name: "all providers valid",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"kimi": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
					"glm": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://open.bigmodel.cn/api/anthropic",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
				},
			},
			testAPI:     false,
			wantTotal:   2,
			wantValid:   2,
			wantInvalid: 0,
		},
		{
			name: "mixed valid and invalid providers",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"valid": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.example.com",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
					"invalid": {
						"env": map[string]interface{}{
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
				},
			},
			testAPI:     false,
			wantTotal:   2,
			wantValid:   1,
			wantInvalid: 1,
		},
		{
			name: "all providers invalid",
			config: &Config{
				Providers: map[string]map[string]interface{}{
					"broken1": {
						"env": map[string]interface{}{
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
					"broken2": {},
				},
			},
			testAPI:     false,
			wantTotal:   2,
			wantValid:   0,
			wantInvalid: 2,
		},
		{
			name: "no providers configured",
			config: &Config{
				Providers: map[string]map[string]interface{}{},
			},
			testAPI:     false,
			wantTotal:   0,
			wantValid:   0,
			wantInvalid: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := validateAllProviders(tt.config, tt.testAPI)

			if summary.Total != tt.wantTotal {
				t.Errorf("validateAllProviders() Total = %v, want %v", summary.Total, tt.wantTotal)
			}
			if summary.Valid != tt.wantValid {
				t.Errorf("validateAllProviders() Valid = %v, want %v", summary.Valid, tt.wantValid)
			}
			if summary.Invalid != tt.wantInvalid {
				t.Errorf("validateAllProviders() Invalid = %v, want %v", summary.Invalid, tt.wantInvalid)
			}
		})
	}
}

// TestValidationResultFields tests that ValidationResult contains expected fields
func TestValidationResultFields(t *testing.T) {
	config := &Config{
		Providers: map[string]map[string]interface{}{
			"test": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://api.example.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
					"ANTHROPIC_MODEL":      "claude-3-opus-20240229",
				},
			},
		},
	}

	result := validateProvider(config, "test", false)

	if result.Provider != "test" {
		t.Errorf("Result.Provider = %q, want %q", result.Provider, "test")
	}

	if result.BaseURL != "https://api.example.com" {
		t.Errorf("Result.BaseURL = %q, want %q", result.BaseURL, "https://api.example.com")
	}

	if result.Model != "claude-3-opus-20240229" {
		t.Errorf("Result.Model = %q, want %q", result.Model, "claude-3-opus-20240229")
	}

	if result.APIStatus != "skipped" {
		t.Errorf("Result.APIStatus = %q, want %q", result.APIStatus, "skipped")
	}
}
