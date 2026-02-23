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
	_, cleanup := setupTestDir(t)
	defer cleanup()

	path := GetSettingsPath()
	expectedPath := filepath.Join(GetDir(), "settings.json")
	if path != expectedPath {
		t.Errorf("GetSettingsPath() = %s, want %s", path, expectedPath)
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

	err := SaveSettings(settings)
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	// Verify file exists
	settingsPath := GetSettingsPath()
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Fatal("SaveSettings() should create settings.json")
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

func TestLoadSettings(t *testing.T) {
	t.Run("valid settings.json", func(t *testing.T) {
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		// Create test settings.json
		testSettings := map[string]interface{}{
			"permissions": map[string]interface{}{
				"allow": []interface{}{"Edit", "Write"},
			},
			"alwaysThinkingEnabled": true,
			"env": map[string]interface{}{
				"MY_CUSTOM_VAR": "value",
			},
			"enabledPlugins": map[string]interface{}{
				"example-skills@anthropic-agent-skills": true,
			},
		}
		settingsPath := filepath.Join(tmpDir, "settings.json")
		writeJSONFile(t, settingsPath, testSettings)

		// Load settings
		settings, err := LoadSettings()
		if err != nil {
			t.Fatalf("LoadSettings() error = %v", err)
		}
		if settings == nil {
			t.Fatal("LoadSettings() should not return nil for valid settings")
		}

		// Verify content
		if settings["alwaysThinkingEnabled"] != true {
			t.Error("alwaysThinkingEnabled should be true")
		}
		if settings["permissions"] == nil {
			t.Error("permissions should be present")
		}
		if settings["enabledPlugins"] == nil {
			t.Error("enabledPlugins should be present")
		}
	})

	t.Run("settings.json does not exist", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		// Don't create settings.json
		_, err := LoadSettings()
		if err != nil {
			t.Fatalf("LoadSettings() should not error when file doesn't exist, got: %v", err)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		// Create invalid JSON file
		settingsPath := GetSettingsPath()
		if err := os.WriteFile(settingsPath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		_, err := LoadSettings()
		if err == nil {
			t.Fatal("LoadSettings() should error with invalid JSON")
		}
		if !strings.Contains(err.Error(), "failed to parse settings file") {
			t.Errorf("Error should mention 'failed to parse settings file', got: %v", err)
		}
	})
}

func TestCleanEnvInSettings(t *testing.T) {
	t.Run("removes ANTHROPIC_ prefixed keys", func(t *testing.T) {
		settings := map[string]interface{}{
			"env": map[string]interface{}{
				"ANTHROPIC_MODEL":    "claude-3.7-sonnet",
				"ANTHROPIC_BASE_URL": "https://old-url.com",
				"MY_CUSTOM_VAR":      "value",
			},
		}
		providerEnvKeys := []string{"BASE_URL"}

		result := CleanEnvInSettings(settings, providerEnvKeys)

		env := result["env"].(map[string]interface{})
		if _, exists := env["ANTHROPIC_MODEL"]; exists {
			t.Error("ANTHROPIC_MODEL should be removed")
		}
		if _, exists := env["ANTHROPIC_BASE_URL"]; exists {
			t.Error("ANTHROPIC_BASE_URL should be removed")
		}
		if _, exists := env["MY_CUSTOM_VAR"]; !exists {
			t.Error("MY_CUSTOM_VAR should be kept")
		}
	})

	t.Run("removes CLAUDE_ prefixed keys", func(t *testing.T) {
		settings := map[string]interface{}{
			"env": map[string]interface{}{
				"CLAUDE_MODEL": "claude-3",
				"CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR": "1",
				"MY_CUSTOM_VAR": "value",
			},
		}
		providerEnvKeys := []string{}

		result := CleanEnvInSettings(settings, providerEnvKeys)

		env := result["env"].(map[string]interface{})
		if _, exists := env["CLAUDE_MODEL"]; exists {
			t.Error("CLAUDE_MODEL should be removed")
		}
		if _, exists := env["CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR"]; exists {
			t.Error("CLAUDE_BASH_MAINTAIN_PROJECT_WORKING_DIR should be removed")
		}
		if _, exists := env["MY_CUSTOM_VAR"]; !exists {
			t.Error("MY_CUSTOM_VAR should be kept")
		}
	})

	t.Run("removes provider env keys", func(t *testing.T) {
		settings := map[string]interface{}{
			"env": map[string]interface{}{
				"BASE_URL":      "old-url",
				"AUTH_TOKEN":    "old-token",
				"MY_CUSTOM_VAR": "value",
			},
		}
		providerEnvKeys := []string{"BASE_URL", "AUTH_TOKEN"}

		result := CleanEnvInSettings(settings, providerEnvKeys)

		env := result["env"].(map[string]interface{})
		if _, exists := env["BASE_URL"]; exists {
			t.Error("BASE_URL should be removed")
		}
		if _, exists := env["AUTH_TOKEN"]; exists {
			t.Error("AUTH_TOKEN should be removed")
		}
		if _, exists := env["MY_CUSTOM_VAR"]; !exists {
			t.Error("MY_CUSTOM_VAR should be kept")
		}
	})

	t.Run("keeps non-matching keys", func(t *testing.T) {
		settings := map[string]interface{}{
			"env": map[string]interface{}{
				"MY_CUSTOM_VAR_1": "value1",
				"MY_CUSTOM_VAR_2": "value2",
			},
		}
		providerEnvKeys := []string{"BASE_URL"}

		result := CleanEnvInSettings(settings, providerEnvKeys)

		env := result["env"].(map[string]interface{})
		if len(env) != 2 {
			t.Errorf("env should have 2 keys, got %d", len(env))
		}
		if env["MY_CUSTOM_VAR_1"] != "value1" {
			t.Error("MY_CUSTOM_VAR_1 should be kept")
		}
		if env["MY_CUSTOM_VAR_2"] != "value2" {
			t.Error("MY_CUSTOM_VAR_2 should be kept")
		}
	})

	t.Run("handles empty env", func(t *testing.T) {
		settings := map[string]interface{}{
			"env": map[string]interface{}{},
		}
		providerEnvKeys := []string{"BASE_URL"}

		result := CleanEnvInSettings(settings, providerEnvKeys)

		env, exists := result["env"]
		if !exists || len(env.(map[string]interface{})) != 0 {
			t.Error("empty env should remain empty")
		}
	})

	t.Run("handles missing env", func(t *testing.T) {
		settings := map[string]interface{}{
			"otherKey": "value",
		}
		providerEnvKeys := []string{}

		result := CleanEnvInSettings(settings, providerEnvKeys)

		if _, exists := result["env"]; exists {
			t.Error("missing env should not be created")
		}
	})

	t.Run("does not modify input", func(t *testing.T) {
		settings := map[string]interface{}{
			"env": map[string]interface{}{
				"ANTHROPIC_MODEL": "value",
			},
			"otherKey": "value",
		}
		providerEnvKeys := []string{"BASE_URL"}

		_ = CleanEnvInSettings(settings, providerEnvKeys)

		// Original should not be modified
		if env, exists := settings["env"]; exists {
			if _, exists := env.(map[string]interface{})["ANTHROPIC_MODEL"]; !exists {
				t.Error("Original settings should not be modified")
			}
		}
	})
}

func TestMergeWithPriority(t *testing.T) {
	t.Run("user settings have highest priority", func(t *testing.T) {
		baseSettings := map[string]interface{}{
			"field1": "base",
			"field2": "base",
		}
		providerSettings := map[string]interface{}{
			"field1": "provider",
			"field3": "provider",
		}
		userSettings := map[string]interface{}{
			"field1": "user",
			"field4": "user",
		}

		result := MergeWithPriority(baseSettings, providerSettings, userSettings)

		if result["field1"] != "user" {
			t.Errorf("field1 should be 'user' (userSettings highest priority), got: %v", result["field1"])
		}
		if result["field2"] != "base" {
			t.Errorf("field2 should be 'base', got: %v", result["field2"])
		}
		if result["field3"] != "provider" {
			t.Errorf("field3 should be 'provider', got: %v", result["field3"])
		}
		if result["field4"] != "user" {
			t.Errorf("field4 should be 'user', got: %v", result["field4"])
		}
	})

	t.Run("provider overrides base when user is nil", func(t *testing.T) {
		baseSettings := map[string]interface{}{
			"field1": "base",
		}
		providerSettings := map[string]interface{}{
			"field1": "provider",
		}
		userSettings := map[string]interface{}(nil)

		result := MergeWithPriority(baseSettings, providerSettings, userSettings)

		if result["field1"] != "provider" {
			t.Errorf("field1 should be 'provider', got: %v", result["field1"])
		}
	})

	t.Run("deep merge of nested maps", func(t *testing.T) {
		baseSettings := map[string]interface{}{
			"nested": map[string]interface{}{
				"a": "base-a",
				"b": "base-b",
			},
		}
		providerSettings := map[string]interface{}{
			"nested": map[string]interface{}{
				"b": "provider-b",
				"c": "provider-c",
			},
		}
		userSettings := map[string]interface{}{
			"nested": map[string]interface{}{
				"c": "user-c",
			},
		}

		result := MergeWithPriority(baseSettings, providerSettings, userSettings)

		nested := result["nested"].(map[string]interface{})
		if nested["a"] != "base-a" {
			t.Errorf("nested.a should be 'base-a', got: %v", nested["a"])
		}
		if nested["b"] != "provider-b" {
			t.Errorf("nested.b should be 'provider-b', got: %v", nested["b"])
		}
		if nested["c"] != "user-c" {
			t.Errorf("nested.c should be 'user-c', got: %v", nested["c"])
		}
	})

	t.Run("all nil returns empty map", func(t *testing.T) {
		result := MergeWithPriority(nil, nil, nil)

		if result == nil || len(result) != 0 {
			t.Errorf("result should be empty map, got: %v", result)
		}
	})

	t.Run("preserves independence of inputs", func(t *testing.T) {
		baseSettings := map[string]interface{}{"a": "base"}
		providerSettings := map[string]interface{}{"b": "provider"}
		userSettings := map[string]interface{}{"c": "user"}

		result := MergeWithPriority(baseSettings, providerSettings, userSettings)

		// Modify result should not affect inputs
		result["newKey"] = "new"

		if _, exists := baseSettings["newKey"]; exists {
			t.Error("baseSettings should not be modified")
		}
		if _, exists := providerSettings["newKey"]; exists {
			t.Error("providerSettings should not be modified")
		}
		if _, exists := userSettings["newKey"]; exists {
			t.Error("userSettings should not be modified")
		}
	})
}

func TestEnsureStopHook(t *testing.T) {
	hookCommand := "/usr/local/bin/ccc supervisor-hook"

	t.Run("adds Stop hook when hooks does not exist", func(t *testing.T) {
		settings := map[string]interface{}{
			"otherField": "value",
		}

		result := EnsureStopHook(settings, hookCommand)

		// Should have hooks
		hooks, exists := result["hooks"]
		if !exists {
			t.Fatal("hooks should exist")
		}

		hooksMap := hooks.(map[string]interface{})
		stopHook, exists := hooksMap["Stop"]
		if !exists {
			t.Fatal("Stop hook should exist")
		}

		// Stop hook should be an array with one element
		stopArray, ok := stopHook.([]interface{})
		if !ok {
			t.Fatal("Stop hook should be an array")
		}
		if len(stopArray) != 1 {
			t.Errorf("Stop hook should have 1 element, got %d", len(stopArray))
		}

		// Verify Stop hook content
		stopConfig, ok := stopArray[0].(map[string]interface{})
		if !ok {
			t.Fatal("Stop hook config should be a map")
		}

		hooksArray, ok := stopConfig["hooks"].([]interface{})
		if !ok {
			t.Fatal("hooks should be an array")
		}
		if len(hooksArray) != 1 {
			t.Errorf("hooks array should have 1 element, got %d", len(hooksArray))
		}

		hookEntry, ok := hooksArray[0].(map[string]interface{})
		if !ok {
			t.Fatal("hook entry should be a map")
		}

		if hookEntry["type"] != "command" {
			t.Errorf("hook type should be 'command', got: %v", hookEntry["type"])
		}
		if hookEntry["command"] != hookCommand {
			t.Errorf("hook command should be '%s', got: %v", hookCommand, hookEntry["command"])
		}
		if hookEntry["timeout"] != float64(600) {
			t.Errorf("hook timeout should be 600, got: %v", hookEntry["timeout"])
		}

		// Other field should be preserved
		if result["otherField"] != "value" {
			t.Error("otherField should be preserved")
		}
	})

	t.Run("replaces existing Stop hook", func(t *testing.T) {
		settings := map[string]interface{}{
			"hooks": map[string]interface{}{
				"Stop": []interface{}{
					map[string]interface{}{
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "other-type",
								"command": "old-command",
							},
						},
					},
				},
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "Bash",
					},
				},
			},
		}

		result := EnsureStopHook(settings, hookCommand)

		hooks := result["hooks"].(map[string]interface{})
		stopArray := hooks["Stop"].([]interface{})
		stopConfig := stopArray[0].(map[string]interface{})
		hooksArray := stopConfig["hooks"].([]interface{})
		hookEntry := hooksArray[0].(map[string]interface{})

		if hookEntry["command"] != hookCommand {
			t.Errorf("Stop hook command should be replaced, got: %v", hookEntry["command"])
		}

		// PreToolUse should be preserved
		_, exists := hooks["PreToolUse"]
		if !exists {
			t.Error("PreToolUse should be preserved")
		}
	})

	t.Run("sets disableAllHooks to false", func(t *testing.T) {
		settings := map[string]interface{}{
			"disableAllHooks": true,
		}

		result := EnsureStopHook(settings, hookCommand)

		if result["disableAllHooks"] != false {
			t.Errorf("disableAllHooks should be false, got: %v", result["disableAllHooks"])
		}
	})

	t.Run("sets allowManagedHooksOnly to false", func(t *testing.T) {
		settings := map[string]interface{}{
			"allowManagedHooksOnly": true,
		}

		result := EnsureStopHook(settings, hookCommand)

		if result["allowManagedHooksOnly"] != false {
			t.Errorf("allowManagedHooksOnly should be false, got: %v", result["allowManagedHooksOnly"])
		}
	})

	t.Run("does not modify input", func(t *testing.T) {
		settings := map[string]interface{}{
			"otherKey": "value",
		}

		_ = EnsureStopHook(settings, hookCommand)

		// Original should not be modified
		if settings["newKey"] != nil {
			t.Error("Original settings should not be modified")
		}
	})

	t.Run("preserves other fields", func(t *testing.T) {
		settings := map[string]interface{}{
			"field1": "value1",
			"field2": 123,
			"enabledPlugins": map[string]interface{}{
				"plugin@marketplace": true,
			},
		}

		result := EnsureStopHook(settings, hookCommand)

		if result["field1"] != "value1" {
			t.Error("field1 should be preserved")
		}
		if result["field2"] != 123 {
			t.Error("field2 should be preserved")
		}
		if result["enabledPlugins"] == nil {
			t.Error("enabledPlugins should be preserved")
		}
	})
}
