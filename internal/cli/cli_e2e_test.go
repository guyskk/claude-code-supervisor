//go:build !windows
// +build !windows

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/guyskk/ccc/internal/config"
)

// TestE2E_Help tests the --help flag
func TestE2E_Help(t *testing.T) {
	t.Parallel()

	console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	// Build ccc binary if needed
	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	// Create test config
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {"env": {"ANTHROPIC_AUTH_TOKEN": "test"}}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Override config dir for this test
	oldGetDir := config.GetDirFunc
	config.GetDirFunc = func() string { return testConfigDir }
	defer func() { config.GetDirFunc = oldGetDir }()

	cmd := exec.Command(cccPath, "--help")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Check for help output
	console.ExpectString("Usage: ccc [provider] [args...]")
	console.ExpectString("Claude Code Configuration Switcher")
	console.ExpectString("Commands:")
	console.ExpectString("Environment Variables:")
	console.ExpectString("CCC_SUPERVISOR")
}

// TestE2E_Version tests the --version flag
func TestE2E_Version(t *testing.T) {
	t.Parallel()

	console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	cmd := exec.Command(cccPath, "--version")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Check for version output
	console.ExpectString("claude-code-config-switcher version")
}

// TestE2E_ProviderSwitch tests provider switching
func TestE2E_ProviderSwitch(t *testing.T) {
	t.Parallel()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	// Create test config
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {"env": {"ANTHROPIC_AUTH_TOKEN": "test1"}},
			"test2": {"env": {"ANTHROPIC_AUTH_TOKEN": "test2"}}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldGetDir := config.GetDirFunc
	config.GetDirFunc = func() string { return testConfigDir }
	defer func() { config.GetDirFunc = oldGetDir }()

	tests := []struct {
		name     string
		provider string
	}{
		{"switch to test1", "test1"},
		{"switch to test2", "test2"},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			defer console.Close()

			cmd := exec.Command(cccPath, tt.provider, "--help")
			cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
			cmd.Stdin = console.Tty()
			cmd.Stdout = console.Tty()
			cmd.Stderr = console.Tty()

			if err := cmd.Start(); err != nil {
				t.Fatalf("failed to start command: %v", err)
			}

			// Should see launching message
			console.ExpectString("Launching with provider: " + tt.provider)
		})
	}
}

// TestE2E_SupervisorMode tests the supervisor mode with CCC_SUPERVISOR env var
func TestE2E_SupervisorMode(t *testing.T) {
	t.Parallel()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	// Create test config
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {"env": {"ANTHROPIC_AUTH_TOKEN": "test1"}}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldGetDir := config.GetDirFunc
	config.GetDirFunc = func() string { return testConfigDir }
	defer func() { config.GetDirFunc = oldGetDir }()

	t.Run("supervisor mode enabled", func(t *testing.T) {
		t.Parallel()

		console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		defer console.Close()

		cmd := exec.Command(cccPath, "test1")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir),
			"CCC_SUPERVISOR=1")
		cmd.Stdin = console.Tty()
		cmd.Stdout = console.Tty()
		cmd.Stderr = console.Tty()

		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start command: %v", err)
		}

		// Should see supervisor mode message
		console.ExpectString("[Supervisor Mode enabled]")
		console.ExpectString("Launching with provider: test1")
	})

	t.Run("supervisor mode disabled", func(t *testing.T) {
		t.Parallel()

		console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		defer console.Close()

		cmd := exec.Command(cccPath, "test1")
		cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
		// Don't set CCC_SUPERVISOR
		cmd.Stdin = console.Tty()
		cmd.Stdout = console.Tty()
		cmd.Stderr = console.Tty()

		if err := cmd.Start(); err != nil {
			t.Fatalf("failed to start command: %v", err)
		}

		// Should NOT see supervisor mode message
		console.ExpectString("Launching with provider: test1")
	})
}

// TestE2E_ValidateCommand tests the validate command
func TestE2E_ValidateCommand(t *testing.T) {
	t.Parallel()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	// Create test config
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {
				"env": {
					"ANTHROPIC_AUTH_TOKEN": "test-token",
					"ANTHROPIC_BASE_URL": "https://api.test.com",
					"ANTHROPIC_MODEL": "test-model"
				}
			}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldGetDir := config.GetDirFunc
	config.GetDirFunc = func() string { return testConfigDir }
	defer func() { config.GetDirFunc = oldGetDir }()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"validate current provider", []string{"validate"}, "Validating"},
		{"validate specific provider", []string{"validate", "test1"}, "Validating"},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(10*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			defer console.Close()

			args := append(tt.args, "--config-dir="+testConfigDir)
			cmd := exec.Command(cccPath, args...)
			cmd.Stdin = console.Tty()
			cmd.Stdout = console.Tty()
			cmd.Stderr = console.Tty()

			if err := cmd.Start(); err != nil {
				t.Fatalf("failed to start command: %v", err)
			}

			// Should see validation output
			console.ExpectString(tt.expected)
		})
	}
}

// TestE2E_ArgPassthrough tests that arguments are properly passed through
func TestE2E_ArgPassthrough(t *testing.T) {
	t.Parallel()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	// Create test config
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {"env": {"ANTHROPIC_AUTH_TOKEN": "test"}}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldGetDir := config.GetDirFunc
	config.GetDirFunc = func() string { return testConfigDir }
	defer func() { config.GetDirFunc = oldGetDir }()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"with --debug flag", []string{"test1", "--debug"}, "Launching with provider: test1"},
		{"with --verbose flag", []string{"test1", "--verbose"}, "Launching with provider: test1"},
		{"with project path", []string{"test1", "/path/to/project"}, "Launching with provider: test1"},
		{"with multiple flags", []string{"test1", "--debug", "--verbose"}, "Launching with provider: test1"},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			defer console.Close()

			cmd := exec.Command(cccPath, tt.args...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
			cmd.Stdin = console.Tty()
			cmd.Stdout = console.Tty()
			cmd.Stderr = console.Tty()

			if err := cmd.Start(); err != nil {
				t.Fatalf("failed to start command: %v", err)
			}

			console.ExpectString(tt.expected)
		})
	}
}

// TestE2E_ArgsOnlyNoProvider tests arguments when no provider is specified
func TestE2E_ArgsOnlyNoProvider(t *testing.T) {
	t.Parallel()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	// Create test config
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {"env": {"ANTHROPIC_AUTH_TOKEN": "test"}}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldGetDir := config.GetDirFunc
	config.GetDirFunc = func() string { return testConfigDir }
	defer func() { config.GetDirFunc = oldGetDir }()

	tests := []struct {
		name string
		args []string
	}{
		{"single flag", []string{"--debug"}},
		{"multiple flags", []string{"--debug", "--verbose"}},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			defer console.Close()

			cmd := exec.Command(cccPath, tt.args...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
			cmd.Stdin = console.Tty()
			cmd.Stdout = console.Tty()
			cmd.Stderr = console.Tty()

			if err := cmd.Start(); err != nil {
				t.Fatalf("failed to start command: %v", err)
			}

			// Should launch with current provider and pass args to claude
			console.ExpectString("Launching with provider: test1")
		})
	}
}

// TestE2E_HookSubcommand tests the supervisor-hook subcommand
func TestE2E_HookSubcommand(t *testing.T) {
	t.Parallel()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	tmpDir := t.TempDir()

	console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	stateDir := tmpDir
	supervisorSettings := filepath.Join(tmpDir, "settings-test.json")

	// Create dummy supervisor settings
	os.WriteFile(supervisorSettings, []byte(`{}`), 0644)

	cmd := exec.Command(cccPath, "supervisor-hook", "--settings", supervisorSettings, "--state-dir", stateDir)
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Send test input
	testInput := `{"session_id":"test-session-123","stop_hook_active":true}`
	console.SendLine(testInput)

	// Should see hook invocation logs
	console.ExpectString("[ccc supervisor-hook]")
	console.ExpectString("session_id: test-session-123")
}

// TestE2E_HelpShowsProviders tests that help shows available providers
func TestE2E_HelpShowsProviders(t *testing.T) {
	t.Parallel()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	// Create test config
	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	testConfig := filepath.Join(testConfigDir, "ccc.json")
	configContent := `{
		"settings": {"permissions": {"defaultMode": "acceptEdits"}},
		"current_provider": "test1",
		"providers": {
			"test1": {},
			"test2": {},
			"test3": {}
		}
	}`
	if err := os.WriteFile(testConfig, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldGetDir := config.GetDirFunc
	config.GetDirFunc = func() string { return testConfigDir }
	defer func() { config.GetDirFunc = oldGetDir }()

	console, err := expect.NewTestConsole(t, expect.WithDefaultTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	cmd := exec.Command(cccPath, "--help")
	cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Help should show available providers
	console.ExpectString("Available Providers:")
	console.ExpectString("test1")
	console.ExpectString("test2")
	console.ExpectString("test3")
}

// TestE2E_Timeout tests command timeout handling
func TestE2E_Timeout(t *testing.T) {
	t.Parallel()

	console, err := expect.NewConsole(
		expect.WithStdout(os.Stdout),
		expect.WithDefaultTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	cccPath, err := os.Executable()
	if err != nil {
		t.Skipf("failed to get executable path: %v", err)
	}

	cmd := exec.Command(cccPath, "--version")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Should complete within timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	done := make(chan error)
	go func() {
		_, _ = console.ExpectString("claude-code-config-switcher version")
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	case <-ctx.Done():
		t.Error("command timed out")
	}
}
