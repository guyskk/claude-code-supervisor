// Integration tests for ccc
package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guyskk/ccc/internal/config"
)

// setupIntegrationTest creates a temporary directory with test configuration.
func setupIntegrationTest(t *testing.T) (string, func()) {
	t.Helper()

	// Save original function
	originalFunc := config.GetDirFunc

	// Create temp directory
	tmpDir := t.TempDir()

	// Override GetDirFunc
	config.GetDirFunc = func() string {
		return tmpDir
	}

	// Create test ccc.json with map-based config
	cfg := &config.Config{
		Settings: map[string]interface{}{
			"permissions": map[string]interface{}{
				"allow": []interface{}{"Edit", "Write", "WebFetch"},
			},
			"alwaysThinkingEnabled": true,
			"env": map[string]interface{}{
				"API_TIMEOUT": "300000",
			},
		},
		CurrentProvider: "kimi",
		Providers: map[string]map[string]interface{}{
			"kimi": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
					"ANTHROPIC_AUTH_TOKEN": "sk-kimi-test",
					"ANTHROPIC_MODEL":      "kimi-k2-thinking",
				},
			},
			"glm": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://open.bigmodel.cn/api/anthropic",
					"ANTHROPIC_AUTH_TOKEN": "sk-glm-test",
					"ANTHROPIC_MODEL":      "glm-4.7",
				},
			},
		},
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	cleanup := func() {
		config.GetDirFunc = originalFunc
	}

	return tmpDir, cleanup
}

func TestIntegrationBuild(t *testing.T) {
	// Verify that the binary builds
	if os.Getenv("CCC_SKIP_BUILD_TEST") == "1" {
		t.Skip("Skipping build test (CCC_SKIP_BUILD_TEST=1)")
	}

	// Build the binary
	buildDir := t.TempDir()
	outputPath := filepath.Join(buildDir, "ccc")

	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, output)
	}

	// Verify the binary exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Binary was not created")
	}

	// Test --version
	versionCmd := exec.Command(outputPath, "--version")
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Version command failed: %v", err)
	}
	if !strings.Contains(string(versionOutput), "claude-code-supervisor version") {
		t.Errorf("Version output unexpected: %s", versionOutput)
	}
}

func TestIntegrationFullFlow(t *testing.T) {
	tmpDir, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Load and verify config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify initial state
	if cfg.CurrentProvider != "kimi" {
		t.Errorf("CurrentProvider = %s, want kimi", cfg.CurrentProvider)
	}
	if len(cfg.Providers) != 2 {
		t.Errorf("Providers count = %d, want 2", len(cfg.Providers))
	}

	// Switch to glm provider
	settingsPath := filepath.Join(tmpDir, "settings-glm.json")
	if _, err := os.Stat(settingsPath); err == nil {
		t.Fatal("settings-glm.json should not exist yet")
	}

	// Get the glm provider config
	glmProvider := cfg.Providers["glm"]

	// Simulate provider switch using DeepMerge
	mergedSettings := config.DeepMerge(cfg.Settings, glmProvider)

	// Verify merged settings using helper functions
	mergedEnv := config.GetEnv(mergedSettings)
	if mergedEnv == nil {
		t.Fatal("Merged env should not be nil")
	}
	if mergedEnv["ANTHROPIC_BASE_URL"] != "https://open.bigmodel.cn/api/anthropic" {
		t.Error("BASE_URL not correctly merged")
	}
	if mergedEnv["API_TIMEOUT"] != "300000" {
		t.Error("Base API_TIMEOUT should be preserved")
	}
	if mergedEnv["ANTHROPIC_AUTH_TOKEN"] != "sk-glm-test" {
		t.Error("Provider AUTH_TOKEN should override")
	}

	// Verify permissions are preserved
	permissions, _ := mergedSettings["permissions"].(map[string]interface{})
	if permissions == nil {
		t.Error("Permissions should be preserved")
	}
}

func TestIntegrationMigrationFlow(t *testing.T) {
	tmpDir, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create old settings.json
	oldSettingsPath := filepath.Join(tmpDir, "settings.json")
	oldSettings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"*"},
		},
		"alwaysThinkingEnabled": true,
		"env": map[string]interface{}{
			"ANTHROPIC_BASE_URL":   "https://api.old.com",
			"ANTHROPIC_AUTH_TOKEN": "sk-old",
		},
	}

	data, err := json.MarshalIndent(oldSettings, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(oldSettingsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Remove ccc.json to trigger migration check
	cccPath := filepath.Join(tmpDir, "ccc.json")
	os.Remove(cccPath)

	// Check that old settings exist
	if _, err := os.Stat(oldSettingsPath); os.IsNotExist(err) {
		t.Fatal("Old settings should exist")
	}

	// Note: We can't fully test migration in automated tests
	// because it requires user input. The migration package
	// has its own comprehensive tests.
}

func TestIntegrationConfigBackwardCompatible(t *testing.T) {
	tmpDir, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create a config file that uses the old map-based format
	// and verify it can be loaded correctly
	oldStyleConfig := map[string]interface{}{
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
					"ANTHROPIC_BASE_URL":   "https://api.kimi.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-test",
				},
			},
		},
	}

	configPath := filepath.Join(tmpDir, "ccc.json")
	data, err := json.MarshalIndent(oldStyleConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Load using config package
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify it loaded correctly
	if cfg.CurrentProvider != "kimi" {
		t.Errorf("CurrentProvider = %s, want kimi", cfg.CurrentProvider)
	}

	// Access settings fields using map access
	if thinking, exists := cfg.Settings["alwaysThinkingEnabled"]; !exists || !thinking.(bool) {
		t.Error("AlwaysThinkingEnabled should be true")
	}

	// Access provider env using helper function
	kimiProvider := cfg.Providers["kimi"]
	kimiEnv := config.GetEnv(kimiProvider)
	if kimiEnv == nil || len(kimiEnv) != 2 {
		t.Error("Provider env should have 2 entries")
	}
}
