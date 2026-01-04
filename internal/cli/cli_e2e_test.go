//go:build !windows && !ci
// +build !windows,!ci

// E2E tests require a PTY and may not work reliably in CI environments.
// Run locally with: go test -v ./internal/cli -run 'TestE2E_'
// Skip in CI by adding -tags=ci to the go test command.

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/guyskk/ccc/internal/config"
	"github.com/twpayne/go-expect"
)

// TestE2E_Help tests the --help flag
func TestE2E_Help(t *testing.T) {
	t.Parallel()

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
	if _, err := console.ExpectString("Usage: ccc [provider] [args...]"); err != nil {
		t.Errorf("expected usage string: %v", err)
	}
	if _, err := console.ExpectString("Claude Code Configuration Switcher"); err != nil {
		t.Errorf("expected title: %v", err)
	}
	if _, err := console.ExpectString("Commands:"); err != nil {
		t.Errorf("expected commands section: %v", err)
	}
	if _, err := console.ExpectString("Environment Variables:"); err != nil {
		t.Errorf("expected env vars section: %v", err)
	}
	if _, err := console.ExpectString("CCC_SUPERVISOR"); err != nil {
		t.Errorf("expected CCC_SUPERVISOR: %v", err)
	}
}

// TestE2E_Version tests the --version flag
func TestE2E_Version(t *testing.T) {
	t.Parallel()

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
	if _, err := console.ExpectString("claude-code-config-switcher version"); err != nil {
		t.Errorf("expected version string: %v", err)
	}
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

			console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
			if _, err := console.ExpectString("Launching with provider: " + tt.provider); err != nil {
				t.Errorf("expected launching message: %v", err)
			}
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

		console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
		if _, err := console.ExpectString("[Supervisor Mode enabled]"); err != nil {
			t.Errorf("expected supervisor mode message: %v", err)
		}
		if _, err := console.ExpectString("Launching with provider: test1"); err != nil {
			t.Errorf("expected launching message: %v", err)
		}
	})

	t.Run("supervisor mode disabled", func(t *testing.T) {
		t.Parallel()

		console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
		if _, err := console.ExpectString("Launching with provider: test1"); err != nil {
			t.Errorf("expected launching message: %v", err)
		}
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

			console, err := expect.NewConsole(expect.WithDefaultTimeout(10 * time.Second))
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
			if _, err := console.ExpectString(tt.expected); err != nil {
				t.Errorf("expected %q: %v", tt.expected, err)
			}
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

			console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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

			if _, err := console.ExpectString(tt.expected); err != nil {
				t.Errorf("expected %q: %v", tt.expected, err)
			}
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

			console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
			if _, err := console.ExpectString("Launching with provider: test1"); err != nil {
				t.Errorf("expected launching message: %v", err)
			}
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

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
	if _, err := console.SendLine(testInput); err != nil {
		t.Errorf("failed to send input: %v", err)
	}

	// Should see hook invocation logs
	if _, err := console.ExpectString("[ccc supervisor-hook]"); err != nil {
		t.Errorf("expected hook log: %v", err)
	}
	if _, err := console.ExpectString("session_id: test-session-123"); err != nil {
		t.Errorf("expected session_id: %v", err)
	}
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

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
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
	if _, err := console.ExpectString("Available Providers:"); err != nil {
		t.Errorf("expected Available Providers: %v", err)
	}
	if _, err := console.ExpectString("test1"); err != nil {
		t.Errorf("expected test1: %v", err)
	}
	if _, err := console.ExpectString("test2"); err != nil {
		t.Errorf("expected test2: %v", err)
	}
	if _, err := console.ExpectString("test3"); err != nil {
		t.Errorf("expected test3: %v", err)
	}
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
		if _, err := console.ExpectString("claude-code-config-switcher version"); err != nil {
			done <- err
			return
		}
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
