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

// ============================================================================
// PreToolUse Hook Configuration Tests
// ============================================================================

func TestSwitchWithHook_PreToolUseConfiguration(t *testing.T) {
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

	// Verify hooks are present in settings
	hooks, ok := result.Settings["hooks"]
	if !ok {
		t.Fatal("Settings should contain 'hooks' field")
	}

	hooksMap, ok := hooks.(map[string]interface{})
	if !ok {
		t.Fatal("hooks should be a map")
	}

	// Verify Stop hook exists
	stopHook, ok := hooksMap["Stop"]
	if !ok {
		t.Error("hooks should contain 'Stop' hook")
	} else {
		stopHookList, ok := stopHook.([]map[string]interface{})
		if !ok {
			t.Error("Stop hook should be a list of maps")
		} else if len(stopHookList) == 0 {
			t.Error("Stop hook list should not be empty")
		}
	}

	// Verify PreToolUse hook exists
	preToolUseHook, ok := hooksMap["PreToolUse"]
	if !ok {
		t.Error("hooks should contain 'PreToolUse' hook")
	} else {
		preToolUseList, ok := preToolUseHook.([]map[string]interface{})
		if !ok {
			t.Error("PreToolUse hook should be a list of maps")
		} else if len(preToolUseList) == 0 {
			t.Error("PreToolUse hook list should not be empty")
		} else {
			// Verify PreToolUse hook configuration structure
			preToolUseConfig := preToolUseList[0]

			// Verify matcher is set to "AskUserQuestion"
			matcher, ok := preToolUseConfig["matcher"]
			if !ok {
				t.Error("PreToolUse hook config should contain 'matcher' field")
			} else if matcher != "AskUserQuestion" {
				t.Errorf("PreToolUse matcher = %v, want 'AskUserQuestion'", matcher)
			}

			// Verify hooks array exists
			hooksArray, ok := preToolUseConfig["hooks"]
			if !ok {
				t.Error("PreToolUse hook config should contain 'hooks' array")
			} else {
				hooksList, ok := hooksArray.([]map[string]interface{})
				if !ok || len(hooksList) == 0 {
					t.Error("PreToolUse hooks array should not be empty")
				} else {
					// Verify hook command structure
					hookConfig := hooksList[0]

					// Verify type is "command"
					if hookType, ok := hookConfig["type"]; !ok || hookType != "command" {
						t.Error("Hook type should be 'command'")
					}

					// Verify timeout is set
					if timeout, ok := hookConfig["timeout"]; !ok {
						t.Error("Hook timeout should be set")
					} else {
						// Timeout can be int or float64 depending on JSON unmarshaling
						var timeoutVal int
						switch v := timeout.(type) {
						case float64:
							timeoutVal = int(v)
						case int:
							timeoutVal = v
						}
						if timeoutVal != 600 {
							t.Errorf("Hook timeout = %v, want 600", timeout)
						}
					}

					// Verify command field exists and contains "supervisor-hook"
					if command, ok := hookConfig["command"]; !ok {
						t.Error("Hook command should be set")
					} else if commandStr, ok := command.(string); !ok || !strings.Contains(commandStr, "supervisor-hook") {
						t.Errorf("Hook command = %v, should contain 'supervisor-hook'", command)
					}
				}
			}
		}
	}
}

// TestSwitchWithHook_PreToolUseAndStopCoexist verifies that PreToolUse and Stop hooks
// can coexist without conflicts and both use the same supervisor command.
func TestSwitchWithHook_PreToolUseAndStopCoexist(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg := setupTestConfig(t)

	// Save initial config
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Switch to kimi
	result, err := SwitchWithHook(cfg, "kimi")
	if err != nil {
		t.Fatalf("SwitchWithHook() error = %v", err)
	}

	// Get hooks configuration
	hooks, ok := result.Settings["hooks"]
	if !ok {
		t.Fatal("Settings should contain 'hooks' field")
	}

	hooksMap, ok := hooks.(map[string]interface{})
	if !ok {
		t.Fatal("hooks should be a map")
	}

	// Both Stop and PreToolUse should exist
	if _, ok := hooksMap["Stop"]; !ok {
		t.Error("hooks should contain 'Stop' hook")
	}
	if _, ok := hooksMap["PreToolUse"]; !ok {
		t.Error("hooks should contain 'PreToolUse' hook")
	}

	// Extract the command from Stop hook
	stopHookList := hooksMap["Stop"].([]map[string]interface{})
	stopConfig := stopHookList[0]
	stopHooks := stopConfig["hooks"].([]map[string]interface{})
	stopHookCommand := stopHooks[0]["command"].(string)

	// Extract the command from PreToolUse hook
	preToolUseList := hooksMap["PreToolUse"].([]map[string]interface{})
	preToolUseConfig := preToolUseList[0]
	preToolUseHooks := preToolUseConfig["hooks"].([]map[string]interface{})
	preToolUseHookCommand := preToolUseHooks[0]["command"].(string)

	// Both should use the same command
	if stopHookCommand != preToolUseHookCommand {
		t.Errorf("Stop and PreToolUse hooks should use the same command: Stop=%s, PreToolUse=%s",
			stopHookCommand, preToolUseHookCommand)
	}

	// Both commands should contain "supervisor-hook"
	if !strings.Contains(stopHookCommand, "supervisor-hook") {
		t.Errorf("Stop hook command should contain 'supervisor-hook': %s", stopHookCommand)
	}
	if !strings.Contains(preToolUseHookCommand, "supervisor-hook") {
		t.Errorf("PreToolUse hook command should contain 'supervisor-hook': %s", preToolUseHookCommand)
	}
}

// TestSwitchWithHook_PreToolUseMatcherIsAskUserQuestion verifies that the PreToolUse
// hook only triggers for AskUserQuestion tool calls, not for other tools.
func TestSwitchWithHook_PreToolUseMatcherIsAskUserQuestion(t *testing.T) {
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

	// Get hooks configuration
	hooks := result.Settings["hooks"].(map[string]interface{})
	preToolUseList := hooks["PreToolUse"].([]map[string]interface{})
	preToolUseConfig := preToolUseList[0]

	// Verify matcher is exactly "AskUserQuestion"
	matcher := preToolUseConfig["matcher"]
	if matcher != "AskUserQuestion" {
		t.Errorf("PreToolUse matcher should be 'AskUserQuestion', got: %v", matcher)
	}

	// Verify matcher is a string (not a regex pattern or complex object)
	if _, ok := matcher.(string); !ok {
		t.Error("PreToolUse matcher should be a string")
	}

	// Verify that common tool names are NOT in the configuration
	// (this ensures we're using a matcher, not a blacklist)
	for _, toolName := range []string{"Browser", "Bash", "Edit", "Write"} {
		if matcher == toolName {
			t.Errorf("PreToolUse matcher should not be '%s', it should be 'AskUserQuestion'", toolName)
		}
	}
}
