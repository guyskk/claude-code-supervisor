package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/guyskk/ccc/internal/config"
)

// setupTestConfig creates a test configuration.
func setupTestConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		Settings: map[string]interface{}{
			"alwaysThinkingEnabled": true,
			"env": map[string]interface{}{
				"API_TIMEOUT":       "30000",
				"DISABLE_TELEMETRY": "1",
			},
		},
		CurrentProvider: "kimi",
		Providers: map[string]map[string]interface{}{
			"kimi": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":         "https://api.moonshot.cn/anthropic",
					"ANTHROPIC_AUTH_TOKEN":       "sk-kimi-xxx",
					"ANTHROPIC_MODEL":            "kimi-k2-thinking",
					"ANTHROPIC_SMALL_FAST_MODEL": "kimi-k2-0905-preview",
				},
			},
			"glm": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://open.bigmodel.cn/api/anthropic",
					"ANTHROPIC_AUTH_TOKEN": "sk-glm-xxx",
					"ANTHROPIC_MODEL":      "glm-4.7",
				},
			},
		},
	}
}

func setupTestDir(t *testing.T) func() {
	t.Helper()

	// Save original function
	originalFunc := config.GetDirFunc

	// Set to temp directory
	tmpDir := t.TempDir()
	config.GetDirFunc = func() string {
		return tmpDir
	}

	return func() {
		config.GetDirFunc = originalFunc
	}
}

func TestSwitchWithHook(t *testing.T) {
	t.Run("switch to existing provider", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		cfg := setupTestConfig(t)

		// Save initial config
		if err := config.Save(cfg); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Switch to glm
		result, err := SwitchWithHook(cfg, "glm")
		if err != nil {
			t.Fatalf("SwitchWithHook() error = %v", err)
		}

		// Verify env vars
		if result.EnvVars == nil {
			t.Fatal("SwitchWithHook() result.EnvVars should not be nil")
		}

		// Build env map for easier testing
		envMap := make(map[string]string)
		for _, pair := range result.EnvVars {
			envMap[pair.Key] = pair.Value
		}

		if envMap["ANTHROPIC_BASE_URL"] != "https://open.bigmodel.cn/api/anthropic" {
			t.Errorf("BASE_URL = %v, want glm URL", envMap["ANTHROPIC_BASE_URL"])
		}
		// Base env should be preserved
		if envMap["API_TIMEOUT"] != "30000" {
			t.Errorf("API_TIMEOUT = %v, want 30000", envMap["API_TIMEOUT"])
		}

		// Verify settings does not contain env
		if _, exists := result.Settings["env"]; exists {
			t.Error("Settings should not contain 'env' field")
		}

		// Verify hooks are present
		if _, exists := result.Settings["hooks"]; !exists {
			t.Error("Settings should contain 'hooks' field")
		}

		// Verify current_provider updated
		if cfg.CurrentProvider != "glm" {
			t.Errorf("CurrentProvider = %s, want glm", cfg.CurrentProvider)
		}

		// Verify settings file was created
		settingsPath := config.GetSettingsPath()
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			t.Errorf("Settings file should exist at %s", settingsPath)
		}

		// Verify supervisor command files were created
		commandsDir := config.GetDir() + "/commands"
		supervisorOnPath := commandsDir + "/supervisor.md"
		supervisorOffPath := commandsDir + "/supervisoroff.md"
		if _, err := os.Stat(supervisorOnPath); os.IsNotExist(err) {
			t.Errorf("supervisor.md should exist at %s", supervisorOnPath)
		}
		if _, err := os.Stat(supervisorOffPath); os.IsNotExist(err) {
			t.Errorf("supervisoroff.md should exist at %s", supervisorOffPath)
		}
	})

	t.Run("switch to non-existing provider", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		cfg := setupTestConfig(t)

		_, err := SwitchWithHook(cfg, "unknown")
		if err == nil {
			t.Fatal("SwitchWithHook() should error for unknown provider")
		}
		if !strings.Contains(err.Error(), "provider 'unknown' not found") {
			t.Errorf("Error message should mention 'provider 'unknown' not found', got: %v", err)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		_, err := SwitchWithHook(nil, "kimi")
		if err == nil {
			t.Fatal("SwitchWithHook() should error for nil config")
		}
	})
}

func TestEnvPairsToStrings(t *testing.T) {
	pairs := []EnvPair{
		{Key: "FOO", Value: "bar"},
		{Key: "BAZ", Value: "qux"},
	}

	result := EnvPairsToStrings(pairs)
	if len(result) != 2 {
		t.Fatalf("EnvPairsToStrings() returned %d items, want 2", len(result))
	}

	// Check format - order may vary since we're using a slice
	foundFoo := false
	foundBaz := false
	for _, s := range result {
		if s == "FOO=bar" {
			foundFoo = true
		}
		if s == "BAZ=qux" {
			foundBaz = true
		}
	}

	if !foundFoo {
		t.Error("EnvPairsToStrings() should contain FOO=bar")
	}
	if !foundBaz {
		t.Error("EnvPairsToStrings() should contain BAZ=qux")
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
			name:     "nil settings",
			settings: nil,
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

func TestFormatProviderName(t *testing.T) {
	tests := []struct {
		name            string
		providerName    string
		currentProvider string
		want            string
	}{
		{
			name:            "current provider",
			providerName:    "kimi",
			currentProvider: "kimi",
			want:            "kimi (current)",
		},
		{
			name:            "not current provider",
			providerName:    "glm",
			currentProvider: "kimi",
			want:            "glm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatProviderName(tt.providerName, tt.currentProvider)
			if got != tt.want {
				t.Errorf("FormatProviderName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestListProviders(t *testing.T) {
	t.Run("list providers", func(t *testing.T) {
		cfg := setupTestConfig(t)

		providers := ListProviders(cfg)
		if len(providers) != 2 {
			t.Errorf("ListProviders() returned %d providers, want 2", len(providers))
		}
	})

	t.Run("nil config", func(t *testing.T) {
		providers := ListProviders(nil)
		if len(providers) != 0 {
			t.Errorf("ListProviders(nil) returned %d providers, want 0", len(providers))
		}
	})

	t.Run("empty providers", func(t *testing.T) {
		cfg := &config.Config{
			Providers: map[string]map[string]interface{}{},
		}
		providers := ListProviders(cfg)
		if len(providers) != 0 {
			t.Errorf("ListProviders() with empty config returned %d providers, want 0", len(providers))
		}
	})
}

func TestValidateProvider(t *testing.T) {
	t.Run("valid provider", func(t *testing.T) {
		cfg := setupTestConfig(t)

		err := ValidateProvider(cfg, "kimi")
		if err != nil {
			t.Errorf("ValidateProvider() error = %v", err)
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		cfg := setupTestConfig(t)

		err := ValidateProvider(cfg, "unknown")
		if err == nil {
			t.Fatal("ValidateProvider() should error for unknown provider")
		}
		if !strings.Contains(err.Error(), "provider 'unknown' not found") {
			t.Errorf("Error message should mention 'provider 'unknown' not found', got: %v", err)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		err := ValidateProvider(nil, "kimi")
		if err == nil {
			t.Fatal("ValidateProvider() should error for nil config")
		}
	})
}

func TestGetDefaultProvider(t *testing.T) {
	t.Run("returns first provider", func(t *testing.T) {
		cfg := setupTestConfig(t)

		provider := GetDefaultProvider(cfg)
		// Since map iteration order is random, just check it's one of the valid providers
		if provider != "kimi" && provider != "glm" {
			t.Errorf("GetDefaultProvider() = %s, want kimi or glm", provider)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		provider := GetDefaultProvider(nil)
		if provider != "" {
			t.Errorf("GetDefaultProvider(nil) = %s, want empty string", provider)
		}
	})

	t.Run("empty providers", func(t *testing.T) {
		cfg := &config.Config{
			Providers: map[string]map[string]interface{}{},
		}
		provider := GetDefaultProvider(cfg)
		if provider != "" {
			t.Errorf("GetDefaultProvider() with empty config = %s, want empty string", provider)
		}
	})
}

func TestGetCurrentProvider(t *testing.T) {
	t.Run("returns current provider", func(t *testing.T) {
		cfg := setupTestConfig(t)

		provider := GetCurrentProvider(cfg)
		if provider != "kimi" {
			t.Errorf("GetCurrentProvider() = %s, want kimi", provider)
		}
	})

	t.Run("current provider not set, returns first", func(t *testing.T) {
		cfg := setupTestConfig(t)
		cfg.CurrentProvider = ""

		provider := GetCurrentProvider(cfg)
		// Since map iteration order is random, just check it's one of the valid providers
		if provider != "kimi" && provider != "glm" {
			t.Errorf("GetCurrentProvider() with empty current = %s, want kimi or glm", provider)
		}
	})

	t.Run("nil config", func(t *testing.T) {
		provider := GetCurrentProvider(nil)
		if provider != "" {
			t.Errorf("GetCurrentProvider(nil) = %s, want empty string", provider)
		}
	})

	t.Run("current provider invalid, falls back to first", func(t *testing.T) {
		cfg := setupTestConfig(t)
		cfg.CurrentProvider = "invalid"

		provider := GetCurrentProvider(cfg)
		// Since map iteration order is random, just check it's one of the valid providers
		if provider != "kimi" && provider != "glm" {
			t.Errorf("GetCurrentProvider() with invalid current = %s, want kimi or glm", provider)
		}
	})
}

func TestShortenError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		maxLength    int
		wantContains string
	}{
		{
			name:         "nil error",
			err:          nil,
			maxLength:    40,
			wantContains: "",
		},
		{
			name:         "short error",
			err:          fmt.Errorf("simple error"),
			maxLength:    40,
			wantContains: "simple error",
		},
		{
			name:         "long error with colon",
			err:          fmt.Errorf("failed to read config file: open /path/to/file: no such file or directory"),
			maxLength:    40,
			wantContains: "no such file or directory",
		},
		{
			name:         "error truncated",
			err:          fmt.Errorf("this is a very long error message that should be truncated because it exceeds the maximum length"),
			maxLength:    30,
			wantContains: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShortenError(tt.err, tt.maxLength)
			if tt.wantContains == "" {
				if got != "" {
					t.Errorf("ShortenError() = %s, want empty string", got)
				}
			} else {
				if !strings.Contains(got, tt.wantContains) {
					t.Errorf("ShortenError() = %s, should contain %s", got, tt.wantContains)
				}
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
