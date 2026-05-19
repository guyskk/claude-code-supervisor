package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guyskk/ccc/internal/config"
)

// writeSettingsJSON writes a settings.json into the test config dir.
func writeSettingsJSON(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(config.GetDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}
}

func TestCheckSettingsEnvConflict_NoSettingsFile(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg := &config.Config{
		Settings: map[string]interface{}{},
		Providers: map[string]map[string]interface{}{
			"glm": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://example.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-x",
				},
			},
		},
	}

	if err := checkSettingsEnvConflict(cfg, "glm"); err != nil {
		t.Errorf("expected no error when settings.json missing, got: %v", err)
	}
}

func TestCheckSettingsEnvConflict_NoConflict(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	writeSettingsJSON(t, `{"env":{"MY_CUSTOM_VAR":"value"}}`)

	cfg := &config.Config{
		Settings: map[string]interface{}{},
		Providers: map[string]map[string]interface{}{
			"glm": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://example.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-x",
				},
			},
		},
	}

	if err := checkSettingsEnvConflict(cfg, "glm"); err != nil {
		t.Errorf("expected no error when only safe custom env present, got: %v", err)
	}
}

func TestCheckSettingsEnvConflict_PrefixHit(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	writeSettingsJSON(t, `{"env":{"ANTHROPIC_BASE_URL":"https://old.example.com"}}`)

	cfg := &config.Config{
		Settings: map[string]interface{}{},
		Providers: map[string]map[string]interface{}{
			"glm": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://new.example.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-x",
				},
			},
		},
	}

	err := checkSettingsEnvConflict(cfg, "glm")
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_BASE_URL") {
		t.Errorf("error should mention conflict key, got: %v", err)
	}
	if strings.Contains(err.Error(), "https://old.example.com") {
		t.Errorf("error must not contain user value (leak risk), got: %v", err)
	}
}

func TestCheckSettingsEnvConflict_ManagedHit(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	writeSettingsJSON(t, `{"env":{"API_TIMEOUT":"60000"}}`)

	cfg := &config.Config{
		Settings: map[string]interface{}{
			"env": map[string]interface{}{
				"API_TIMEOUT": "30000",
			},
		},
		Providers: map[string]map[string]interface{}{
			"glm": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://example.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-x",
				},
			},
		},
	}

	err := checkSettingsEnvConflict(cfg, "glm")
	if err == nil {
		t.Fatal("expected conflict error for managed key, got nil")
	}
	if !strings.Contains(err.Error(), "API_TIMEOUT") {
		t.Errorf("error should mention conflict key API_TIMEOUT, got: %v", err)
	}
}

func TestCheckSettingsEnvConflict_UnknownProvider(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	writeSettingsJSON(t, `{"env":{"ANTHROPIC_BASE_URL":"https://x"}}`)

	cfg := &config.Config{
		Settings:  map[string]interface{}{},
		Providers: map[string]map[string]interface{}{},
	}

	// Unknown provider: guard skips silently, leaves the real error to SwitchWithHook.
	if err := checkSettingsEnvConflict(cfg, "missing"); err != nil {
		t.Errorf("expected guard to skip on unknown provider, got: %v", err)
	}
}

func TestCheckValidateEnvConflict_NoSettingsFile(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	cfg := &config.Config{
		Settings: map[string]interface{}{},
		Providers: map[string]map[string]interface{}{
			"glm": {"env": map[string]interface{}{"ANTHROPIC_BASE_URL": "https://x"}},
		},
	}

	if err := checkValidateEnvConflict(cfg, &ValidateCommand{}); err != nil {
		t.Errorf("expected no error when settings.json missing, got: %v", err)
	}
}

func TestCheckValidateEnvConflict_AllProvidersStrict(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// settings.json only has a key that overlaps with provider "kimi"'s env,
	// not "glm"'s. With --all the guard must still detect it.
	writeSettingsJSON(t, `{"env":{"ANTHROPIC_SMALL_FAST_MODEL":"old"}}`)

	cfg := &config.Config{
		Settings: map[string]interface{}{},
		Providers: map[string]map[string]interface{}{
			"glm": {"env": map[string]interface{}{"ANTHROPIC_BASE_URL": "https://glm"}},
			"kimi": {"env": map[string]interface{}{
				"ANTHROPIC_BASE_URL":         "https://kimi",
				"ANTHROPIC_SMALL_FAST_MODEL": "kimi-fast",
			}},
		},
	}

	err := checkValidateEnvConflict(cfg, &ValidateCommand{ValidateAll: true})
	if err == nil {
		t.Fatal("expected conflict for ANTHROPIC_SMALL_FAST_MODEL under --all, got nil")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_SMALL_FAST_MODEL") {
		t.Errorf("error should mention conflict key, got: %v", err)
	}
}

func TestCheckValidateEnvConflict_SingleProviderScope(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// settings.json has ANTHROPIC_BASE_URL which always triggers prefix rule,
	// regardless of which provider is targeted.
	writeSettingsJSON(t, `{"env":{"ANTHROPIC_BASE_URL":"https://old"}}`)

	cfg := &config.Config{
		CurrentProvider: "glm",
		Settings:        map[string]interface{}{},
		Providers: map[string]map[string]interface{}{
			"glm": {"env": map[string]interface{}{"ANTHROPIC_BASE_URL": "https://glm"}},
		},
	}

	err := checkValidateEnvConflict(cfg, &ValidateCommand{Provider: "glm"})
	if err == nil {
		t.Fatal("expected conflict, got nil")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_BASE_URL") {
		t.Errorf("error should mention conflict key, got: %v", err)
	}
}

func TestCheckValidateEnvConflict_NoConflict(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	writeSettingsJSON(t, `{"env":{"MY_CUSTOM":"x"}}`)

	cfg := &config.Config{
		Settings: map[string]interface{}{},
		Providers: map[string]map[string]interface{}{
			"glm": {"env": map[string]interface{}{"ANTHROPIC_BASE_URL": "https://x"}},
		},
	}

	if err := checkValidateEnvConflict(cfg, &ValidateCommand{ValidateAll: true}); err != nil {
		t.Errorf("expected no error when only safe env present, got: %v", err)
	}
}
