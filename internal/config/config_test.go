package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// setupTestDir creates a temporary directory for testing.
func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()

	// Save original function
	originalFunc := GetDirFunc

	// Create temp directory
	tmpDir := t.TempDir()

	// Override GetDirFunc
	GetDirFunc = func() string {
		return tmpDir
	}

	// Return cleanup function
	cleanup := func() {
		GetDirFunc = originalFunc
	}

	return tmpDir, cleanup
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
	content, err := MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// readJSONFile reads and unmarshals a JSON file.
func readJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("Failed to unmarshal JSON from %s: %v", path, err)
	}
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

func TestGetConfigPath(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	path := GetConfigPath()
	expectedPath := filepath.Join(GetDir(), "ccc.json")
	if path != expectedPath {
		t.Errorf("GetConfigPath() = %s, want %s", path, expectedPath)
	}
}

func TestGetSettingsPath(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		want         string
	}{
		{
			name:         "with provider name",
			providerName: "kimi",
			want:         "settings-kimi.json",
		},
		{
			name:         "empty provider name",
			providerName: "",
			want:         "settings.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTestDir(t)
			defer cleanup()

			path := GetSettingsPath(tt.providerName)
			if !strings.Contains(path, tt.want) {
				t.Errorf("GetSettingsPath(%q) should contain %q, got %s", tt.providerName, tt.want, path)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create test config
		testConfig := map[string]interface{}{
			"settings": map[string]interface{}{
				"permissions": map[string]interface{}{
					"allow": []interface{}{"Edit", "Write"},
				},
				"alwaysThinkingEnabled": true,
				"env": map[string]interface{}{
					"API_TIMEOUT": "30000",
				},
			},
			"current_provider": "kimi",
			"providers": map[string]interface{}{
				"kimi": map[string]interface{}{
					"env": map[string]interface{}{
						"BASE_URL": "https://api.kimi.com",
					},
				},
			},
		}
		configPath := filepath.Join(tmpDir, "ccc.json")
		writeJSONFile(t, configPath, testConfig)

		// Load config
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Verify
		if cfg.CurrentProvider != "kimi" {
			t.Errorf("CurrentProvider = %s, want kimi", cfg.CurrentProvider)
		}
		if len(cfg.Providers) != 1 {
			t.Errorf("Providers count = %d, want 1", len(cfg.Providers))
		}
		if cfg.Settings == nil || len(cfg.Settings) == 0 {
			t.Errorf("Settings not loaded correctly")
		}
	})

	t.Run("config file does not exist", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		_, err := Load()
		if err == nil {
			t.Fatal("Load() should error when config doesn't exist")
		}
		if !strings.Contains(err.Error(), "failed to read config file") {
			t.Errorf("Error should mention 'failed to read config file', got: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create invalid JSON file
		configPath := filepath.Join(tmpDir, "ccc.json")
		if err := os.WriteFile(configPath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		_, err := Load()
		if err == nil {
			t.Fatal("Load() should error with invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed to parse config file") {
			t.Errorf("Error should mention 'failed to parse config file', got: %v", err)
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("save config", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		cfg := &Config{
			Settings: map[string]interface{}{
				"alwaysThinkingEnabled": true,
			},
			CurrentProvider: "default",
			Providers: map[string]map[string]interface{}{
				"default": {},
			},
		}

		err := Save(cfg)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file exists
		configPath := GetConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Save() should create ccc.json")
		}

		// Verify content
		loaded, err := Load()
		if err != nil {
			t.Fatalf("Failed to load: %v", err)
		}
		if loaded.CurrentProvider != "default" {
			t.Errorf("CurrentProvider = %s, want default", loaded.CurrentProvider)
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		cfg := &Config{
			Settings:        map[string]interface{}{},
			CurrentProvider: "test",
			Providers:       map[string]map[string]interface{}{},
		}

		err := Save(cfg)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify directory was created
		configPath := GetConfigPath()
		dir := filepath.Dir(configPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Fatal("Save() should create config directory")
		}
	})
}

func TestSaveSettings(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"*"},
		},
		"alwaysThinkingEnabled": true,
		"env": map[string]interface{}{
			"BASE_URL": "https://api.example.com",
		},
	}

	err := SaveSettings(settings, "kimi")
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	// Verify file exists
	settingsPath := GetSettingsPath("kimi")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("SaveSettings() should create settings-kimi.json")
	}

	// Verify content
	var loaded map[string]interface{}
	readJSONFile(t, settingsPath, &loaded)
	if loaded["alwaysThinkingEnabled"] != true {
		t.Error("alwaysThinkingEnabled should be true")
	}
}

func TestDeepCopy(t *testing.T) {
	tests := []struct {
		name     string
		original map[string]interface{}
	}{
		{
			name: "simple map",
			original: map[string]interface{}{
				"A": "1",
				"B": 2,
			},
		},
		{
			name: "nested map",
			original: map[string]interface{}{
				"env": map[string]interface{}{
					"A": "1",
					"B": "2",
				},
			},
		},
		{
			name:     "nil map",
			original: nil,
		},
		{
			name: "mixed types",
			original: map[string]interface{}{
				"str":   "value",
				"num":   42,
				"bool":  true,
				"slice": []interface{}{1, 2, 3},
				"nested": map[string]interface{}{
					"inner": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copied := deepCopy(tt.original)

			// Check equality
			if !reflect.DeepEqual(copied, tt.original) {
				t.Errorf("deepCopy() = %v, want %v", copied, tt.original)
			}

			// Check independence (modify copy shouldn't affect original)
			if copied != nil {
				copied["modified"] = true
			}
			if _, exists := tt.original["modified"]; exists {
				t.Error("Modifying copy should not affect original")
			}
		})
	}
}

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]interface{}
		provider map[string]interface{}
		want     map[string]interface{}
	}{
		{
			name: "merge env fields",
			base: map[string]interface{}{
				"env": map[string]interface{}{
					"A": "1",
					"B": "2",
				},
			},
			provider: map[string]interface{}{
				"env": map[string]interface{}{
					"B": "3",
					"C": "4",
				},
			},
			want: map[string]interface{}{
				"env": map[string]interface{}{
					"A": "1",
					"B": "3",
					"C": "4",
				},
			},
		},
		{
			name: "merge nested permissions",
			base: map[string]interface{}{
				"permissions": map[string]interface{}{
					"allow": []interface{}{"A"},
					"mode":  "mode1",
				},
			},
			provider: map[string]interface{}{
				"permissions": map[string]interface{}{
					"allow": []interface{}{"B"},
				},
			},
			want: map[string]interface{}{
				"permissions": map[string]interface{}{
					"allow": []interface{}{"B"},
					"mode":  "mode1",
				},
			},
		},
		{
			name: "provider has new field",
			base: map[string]interface{}{
				"existing": "value",
			},
			provider: map[string]interface{}{
				"newField": "newValue",
			},
			want: map[string]interface{}{
				"existing": "value",
				"newField": "newValue",
			},
		},
		{
			name:     "both nil",
			base:     nil,
			provider: nil,
			want:     map[string]interface{}{},
		},
		{
			name:     "base nil, provider has values",
			base:     nil,
			provider: map[string]interface{}{"A": "1"},
			want:     map[string]interface{}{"A": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeepMerge(tt.base, tt.provider)
			if !reflect.DeepEqual(got, tt.want) {
				gotJSON, _ := json.MarshalIndent(got, "", "  ")
				wantJSON, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("DeepMerge() =\n%s\n\nwant:\n%s", gotJSON, wantJSON)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]interface{}
		want     map[string]interface{}
	}{
		{
			name: "env exists",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"A": "1",
				},
			},
			want: map[string]interface{}{
				"A": "1",
			},
		},
		{
			name:     "env does not exist",
			settings: map[string]interface{}{},
			want:     nil,
		},
		{
			name:     "settings is nil",
			settings: nil,
			want:     nil,
		},
		{
			name: "env is not a map",
			settings: map[string]interface{}{
				"env": "not a map",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEnv(tt.settings)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvString(t *testing.T) {
	tests := []struct {
		name         string
		settings     map[string]interface{}
		key          string
		defaultValue string
		want         string
	}{
		{
			name: "key exists",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"KEY": "value",
				},
			},
			key:          "KEY",
			defaultValue: "default",
			want:         "value",
		},
		{
			name: "key does not exist",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"OTHER": "value",
				},
			},
			key:          "KEY",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "env does not exist",
			settings:     map[string]interface{}{},
			key:          "KEY",
			defaultValue: "default",
			want:         "default",
		},
		{
			name: "value is not a string",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"KEY": 123,
				},
			},
			key:          "KEY",
			defaultValue: "default",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEnvString(tt.settings, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetEnvString() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetAuthToken(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]interface{}
		want     string
	}{
		{
			name: "token exists",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_AUTH_TOKEN": "sk-xxx",
				},
			},
			want: "sk-xxx",
		},
		{
			name:     "token does not exist",
			settings: map[string]interface{}{},
			want:     "PLEASE_SET_ANTHROPIC_AUTH_TOKEN",
		},
		{
			name: "token is empty string",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_AUTH_TOKEN": "",
				},
			},
			want: "PLEASE_SET_ANTHROPIC_AUTH_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAuthToken(tt.settings)
			if got != tt.want {
				t.Errorf("GetAuthToken() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]interface{}
		want     string
	}{
		{
			name: "base url exists",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL": "https://api.example.com",
				},
			},
			want: "https://api.example.com",
		},
		{
			name:     "base url does not exist",
			settings: map[string]interface{}{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBaseURL(tt.settings)
			if got != tt.want {
				t.Errorf("GetBaseURL() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]interface{}
		want     string
	}{
		{
			name: "model exists",
			settings: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_MODEL": "claude-3",
				},
			},
			want: "claude-3",
		},
		{
			name:     "model does not exist",
			settings: map[string]interface{}{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetModel(tt.settings)
			if got != tt.want {
				t.Errorf("GetModel() = %s, want %s", got, tt.want)
			}
		})
	}
}

// MarshalIndent is a helper for JSON marshaling with indentation.
func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func TestGetSettingsJSONPath(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"default path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupTestDir(t)
			defer cleanup()

			path := GetSettingsJSONPath()
			if !strings.Contains(path, "settings.json") {
				t.Errorf("GetSettingsJSONPath() should contain 'settings.json', got %s", path)
			}
		})
	}
}

func TestClearEnvInSettings(t *testing.T) {
	t.Run("settings.json exists with env field", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create settings.json with env field
		settingsPath := filepath.Join(tmpDir, "settings.json")
		originalSettings := map[string]interface{}{
			"permissions": map[string]interface{}{
				"allow": []interface{}{"Edit", "Write"},
			},
			"alwaysThinkingEnabled": true,
			"env": map[string]interface{}{
				"ANTHROPIC_AUTH_TOKEN": "sk-old",
				"ANTHROPIC_BASE_URL":   "https://old.example.com",
			},
		}
		writeJSONFile(t, settingsPath, originalSettings)

		// Clear env
		cleared, err := ClearEnvInSettings()
		if err != nil {
			t.Fatalf("ClearEnvInSettings() error = %v", err)
		}
		if !cleared {
			t.Error("ClearEnvInSettings() should return true when env was cleared")
		}

		// Verify env is cleared but other fields preserved
		var loaded map[string]interface{}
		readJSONFile(t, settingsPath, &loaded)
		if loaded["permissions"] == nil {
			t.Error("permissions should be preserved")
		}
		if loaded["alwaysThinkingEnabled"] != true {
			t.Error("alwaysThinkingEnabled should be preserved")
		}
		env, ok := loaded["env"].(map[string]interface{})
		if !ok {
			t.Fatal("env should exist and be a map")
		}
		if len(env) != 0 {
			t.Errorf("env should be empty, got %v", env)
		}
	})

	t.Run("settings.json does not exist", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		cleared, err := ClearEnvInSettings()
		if err != nil {
			t.Fatalf("ClearEnvInSettings() error = %v", err)
		}
		if cleared {
			t.Error("ClearEnvInSettings() should return false when file doesn't exist")
		}
	})

	t.Run("settings.json exists without env field", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create settings.json without env field
		settingsPath := filepath.Join(tmpDir, "settings.json")
		originalSettings := map[string]interface{}{
			"permissions": map[string]interface{}{
				"allow": []interface{}{"Edit"},
			},
			"alwaysThinkingEnabled": true,
		}
		writeJSONFile(t, settingsPath, originalSettings)

		// Try to clear env
		cleared, err := ClearEnvInSettings()
		if err != nil {
			t.Fatalf("ClearEnvInSettings() error = %v", err)
		}
		if cleared {
			t.Error("ClearEnvInSettings() should return false when env field doesn't exist")
		}

		// Verify file unchanged
		var loaded map[string]interface{}
		readJSONFile(t, settingsPath, &loaded)
		compareJSON(t, loaded, originalSettings)
	})

	t.Run("settings.json has invalid JSON", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create invalid JSON file
		settingsPath := filepath.Join(tmpDir, "settings.json")
		if err := os.WriteFile(settingsPath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		_, err := ClearEnvInSettings()
		if err == nil {
			t.Fatal("ClearEnvInSettings() should error with invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed to parse settings.json") {
			t.Errorf("Error should mention 'failed to parse settings.json', got: %v", err)
		}
	})
}
