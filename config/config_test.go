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
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// readJSONFile reads and unmarshals a JSON file.
func readJSONFile(t *testing.T, path string, target interface{}) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("Failed to unmarshal JSON from %s: %v", path, err)
	}
}

// compareJSON compares two JSON objects deeply.
func compareJSON(t *testing.T, got, want interface{}) {
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
		testConfig := Config{
			Settings: Settings{
				Permissions: &Permissions{
					Allow: []string{"Edit", "Write"},
				},
				AlwaysThinkingEnabled: true,
				Env: Env{
					"API_TIMEOUT": "30000",
				},
			},
			CurrentProvider: "kimi",
			Providers: map[string]ProviderConfig{
				"kimi": {
					Env: Env{
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
		if cfg.Settings.Permissions == nil || len(cfg.Settings.Permissions.Allow) != 2 {
			t.Errorf("Permissions not loaded correctly")
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
		tmpDir, cleanup := setupTestDir(t)
		defer cleanup()

		cfg := &Config{
			Settings: Settings{
				AlwaysThinkingEnabled: true,
			},
			CurrentProvider: "default",
			Providers: map[string]ProviderConfig{
				"default": {},
			},
		}

		err := Save(cfg)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file exists
		configPath := filepath.Join(tmpDir, "ccc.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Save() should create ccc.json")
		}

		// Verify content
		var loaded Config
		readJSONFile(t, configPath, &loaded)
		if loaded.CurrentProvider != "default" {
			t.Errorf("CurrentProvider = %s, want default", loaded.CurrentProvider)
		}
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		_, cleanup := setupTestDir(t)
		defer cleanup()

		cfg := &Config{
			Settings:        Settings{},
			CurrentProvider: "test",
			Providers:       map[string]ProviderConfig{},
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

	settings := &Settings{
		Permissions: &Permissions{
			Allow: []string{"*"},
		},
		AlwaysThinkingEnabled: true,
		Env: Env{
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
	var loaded Settings
	readJSONFile(t, settingsPath, &loaded)
	if !loaded.AlwaysThinkingEnabled {
		t.Error("AlwaysThinkingEnabled should be true")
	}
}

func TestMergeEnv(t *testing.T) {
	tests := []struct {
		name string
		env1 Env
		env2 Env
		want Env
	}{
		{
			name: "merge two envs",
			env1: Env{"A": "1", "B": "2"},
			env2: Env{"B": "3", "C": "4"},
			want: Env{"A": "1", "B": "3", "C": "4"},
		},
		{
			name: "env1 is nil",
			env1: nil,
			env2: Env{"A": "1"},
			want: Env{"A": "1"},
		},
		{
			name: "env2 is nil",
			env1: Env{"A": "1"},
			env2: nil,
			want: Env{"A": "1"},
		},
		{
			name: "both nil",
			env1: nil,
			env2: nil,
			want: Env{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeEnv(tt.env1, tt.env2)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergePermissions(t *testing.T) {
	tests := []struct {
		name string
		p1   *Permissions
		p2   *Permissions
		want *Permissions
	}{
		{
			name: "both have values",
			p1: &Permissions{Allow: []string{"A"}, DefaultMode: "mode1"},
			p2: &Permissions{Allow: []string{"B"}, DefaultMode: "mode2"},
			want: &Permissions{Allow: []string{"B"}, DefaultMode: "mode2"},
		},
		{
			name: "p1 is nil",
			p1:   nil,
			p2:   &Permissions{Allow: []string{"A"}},
			want: &Permissions{Allow: []string{"A"}},
		},
		{
			name: "p2 is nil",
			p1:   &Permissions{Allow: []string{"A"}},
			p2:   nil,
			want: &Permissions{Allow: []string{"A"}},
		},
		{
			name: "both nil",
			p1:   nil,
			p2:   nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergePermissions(tt.p1, tt.p2)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergePermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeSettings(t *testing.T) {
	tests := []struct {
		name     string
		base     *Settings
		provider *ProviderConfig
		want     *Settings
	}{
		{
			name: "merge with provider env",
			base: &Settings{
				AlwaysThinkingEnabled: true,
				Env: Env{"A": "1", "B": "2"},
			},
			provider: &ProviderConfig{
				Env: Env{"B": "3", "C": "4"},
			},
			want: &Settings{
				AlwaysThinkingEnabled: true,
				Env: Env{"A": "1", "B": "3", "C": "4"},
			},
		},
		{
			name: "base is nil",
			base: nil,
			provider: &ProviderConfig{
				Env: Env{"A": "1"},
			},
			want: &Settings{
				Env: Env{"A": "1"},
			},
		},
		{
			name:     "provider is nil",
			base:     &Settings{Env: Env{"A": "1"}},
			provider: nil,
			want:     &Settings{Env: Env{"A": "1"}},
		},
		{
			name:     "both nil",
			base:     nil,
			provider: nil,
			want:     &Settings{Env: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeSettings(tt.base, tt.provider)
			if got.AlwaysThinkingEnabled != tt.want.AlwaysThinkingEnabled {
				t.Errorf("AlwaysThinkingEnabled = %v, want %v", got.AlwaysThinkingEnabled, tt.want.AlwaysThinkingEnabled)
			}
			if !reflect.DeepEqual(got.Env, tt.want.Env) {
				t.Errorf("Env = %v, want %v", got.Env, tt.want.Env)
			}
		})
	}
}
