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
	"sync"
	"testing"
	"time"

	"github.com/twpayne/go-expect"
)

var cccBinaryPath string

// TestMain builds the ccc binary before running tests
func TestMain(m *testing.M) {
	// Create a temporary directory for the test binary
	tmpDir, err := os.MkdirTemp("", "ccc-e2e-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Build the ccc binary
	cccBinaryPath = filepath.Join(tmpDir, "ccc")
	buildCmd := exec.Command("go", "build", "-o", cccBinaryPath, ".")
	// main.go is in the project root, which is ../.. from internal/cli
	buildCmd.Dir = "../.."
	buildCmd.Env = append(os.Environ(),
		"CGO_ENABLED=0", // Static binary
	)
	if output, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build ccc binary: %v\n%s\n", err, output)
		os.Exit(1)
	}

	// Run the tests
	os.Exit(m.Run())
}

// processManager manages subprocess lifecycle to prevent resource leaks
type processManager struct {
	mu        sync.Mutex
	processes []*exec.Cmd
	waited    map[*exec.Cmd]struct{} // Track which processes have been waited on
}

// add registers a process for cleanup
func (pm *processManager) add(cmd *exec.Cmd) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.processes = append(pm.processes, cmd)
	if pm.waited == nil {
		pm.waited = make(map[*exec.Cmd]struct{})
	}
}

// markWaited marks a process as already waited on (to avoid double-wait in cleanup)
func (pm *processManager) markWaited(cmd *exec.Cmd) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.waited == nil {
		pm.waited = make(map[*exec.Cmd]struct{})
	}
	pm.waited[cmd] = struct{}{}
}

// cleanup terminates all registered processes that haven't been waited on
func (pm *processManager) cleanup() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, cmd := range pm.processes {
		if cmd.Process != nil {
			// Only kill and wait if we haven't already waited on this process
			if _, alreadyWaited := pm.waited[cmd]; !alreadyWaited {
				cmd.Process.Kill()
				cmd.Wait()
			}
		}
	}
	pm.processes = nil
	pm.waited = nil
}

// start starts a command and registers it for cleanup
// The caller is responsible for waiting on the process (or letting cleanup handle it)
func (pm *processManager) start(cmd *exec.Cmd) error {
	pm.add(cmd)
	return cmd.Start()
}

// TestE2E_Help tests the --help flag
func TestE2E_Help(t *testing.T) {
	t.Parallel()

	pm := &processManager{}
	defer pm.cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

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

	cmd := exec.CommandContext(ctx, cccBinaryPath, "--help")
	cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	// Start the command
	if err := pm.start(cmd); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Check for help output while process is still running
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

	// Wait for command to complete
	pm.markWaited(cmd)
	cmd.Wait()
}

// TestE2E_Version tests the --version flag
func TestE2E_Version(t *testing.T) {
	t.Parallel()

	pm := &processManager{}
	defer pm.cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	cmd := exec.CommandContext(ctx, cccBinaryPath, "--version")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := pm.start(cmd); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Check for version output while process is still running
	if _, err := console.ExpectString("claude-code-config-switcher version"); err != nil {
		t.Errorf("expected version string: %v", err)
	}

	// Wait for command to complete
	pm.markWaited(cmd)
	cmd.Wait()
}

// TestE2E_ProviderSwitch tests provider switching
func TestE2E_ProviderSwitch(t *testing.T) {
	// Not parallel - share test config
	// t.Parallel()

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

			pm := &processManager{}
			defer pm.cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
			if err != nil {
				t.Fatal(err)
			}
			defer console.Close()

			cmd := exec.CommandContext(ctx, cccBinaryPath, tt.provider, "--help")
			cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
			cmd.Stdin = console.Tty()
			cmd.Stdout = console.Tty()
			cmd.Stderr = console.Tty()

			if err := pm.start(cmd); err != nil {
				t.Fatalf("failed to start command: %v", err)
			}

			// Should see launching message while process is running
			if _, err := console.ExpectString("Launching with provider: " + tt.provider); err != nil {
				t.Errorf("expected launching message: %v", err)
			}
		})
	}
}

// TestE2E_SupervisorMode tests the supervisor mode with CCC_SUPERVISOR env var
func TestE2E_SupervisorMode(t *testing.T) {
	// t.Parallel()

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

	t.Run("supervisor mode enabled", func(t *testing.T) {
		t.Parallel()

		pm := &processManager{}
		defer pm.cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
		if err != nil {
			t.Fatal(err)
		}
		defer console.Close()

		cmd := exec.CommandContext(ctx, cccBinaryPath, "test1")
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir),
			"CCC_SUPERVISOR=1")
		cmd.Stdin = console.Tty()
		cmd.Stdout = console.Tty()
		cmd.Stderr = console.Tty()

		if err := pm.start(cmd); err != nil {
			t.Fatalf("failed to start command: %v", err)
		}

		// Should see supervisor mode message while process is running
		// The actual output format is "Supervisor enabled: tail -f <logpath>"
		if _, err := console.ExpectString("Supervisor enabled: tail -f"); err != nil {
			t.Errorf("expected supervisor enabled message: %v", err)
		}
		if _, err := console.ExpectString("Launching with provider: test1"); err != nil {
			t.Errorf("expected launching message: %v", err)
		}
	})

	t.Run("supervisor mode disabled", func(t *testing.T) {
		t.Parallel()

		pm := &processManager{}
		defer pm.cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
		if err != nil {
			t.Fatal(err)
		}
		defer console.Close()

		cmd := exec.CommandContext(ctx, cccBinaryPath, "test1")
		cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
		cmd.Stdin = console.Tty()
		cmd.Stdout = console.Tty()
		cmd.Stderr = console.Tty()

		if err := pm.start(cmd); err != nil {
			t.Fatalf("failed to start command: %v", err)
		}

		// Should see launching message (without supervisor mode message)
		if _, err := console.ExpectString("Launching with provider: test1"); err != nil {
			t.Errorf("expected launching message: %v", err)
		}
	})
}

// TestE2E_ValidateCommand tests the validate command
func TestE2E_ValidateCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"validate current provider", []string{"validate"}},
		{"validate specific provider", []string{"validate", "test1"}},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			// Don't run in parallel to avoid PTY resource issues

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

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Run command and capture output
			cmd := exec.CommandContext(ctx, cccBinaryPath, tt.args...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
			output, err := cmd.CombinedOutput()
			if err != nil {
				// Validate command returns error when API test fails - that's expected
				t.Logf("Command completed with error (expected): %v", err)
			}

			// Check that we got some validation output
			outputStr := string(output)
			if !contains(outputStr, "test1") {
				t.Errorf("Expected provider name 'test1' in output, got: %s", outputStr)
			}
			if !contains(outputStr, "Base URL") {
				t.Errorf("Expected 'Base URL' in output, got: %s", outputStr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestE2E_Launching tests provider launching with various flags
func TestE2E_Launching(t *testing.T) {
	// t.Parallel()

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

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"with provider and --debug flag", []string{"test1", "--debug"}, "Launching with provider: test1"},
		{"with provider and --verbose flag", []string{"test1", "--verbose"}, "Launching with provider: test1"},
		{"with provider and project path", []string{"test1", "/path/to/project"}, "Launching with provider: test1"},
		{"with provider and multiple flags", []string{"test1", "--debug", "--verbose"}, "Launching with provider: test1"},
		{"without provider using current", []string{"--debug"}, "Launching with provider: test1"},
		{"without provider multiple flags", []string{"--debug", "--verbose"}, "Launching with provider: test1"},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pm := &processManager{}
			defer pm.cleanup()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
			if err != nil {
				t.Fatal(err)
			}
			defer console.Close()

			cmd := exec.CommandContext(ctx, cccBinaryPath, tt.args...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
			cmd.Stdin = console.Tty()
			cmd.Stdout = console.Tty()
			cmd.Stderr = console.Tty()

			if err := pm.start(cmd); err != nil {
				t.Fatalf("failed to start command: %v", err)
			}

			if _, err := console.ExpectString(tt.expected); err != nil {
				t.Errorf("expected %q: %v", tt.expected, err)
			}
		})
	}
}

// TestE2E_HookSubcommand tests the supervisor-hook subcommand
func TestE2E_HookSubcommand(t *testing.T) {
	t.Parallel()

	pm := &processManager{}
	defer pm.cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	// Create test config directory
	testConfigDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test config (required by hook)
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

	// Create ccc state directory
	cccStateDir := filepath.Join(testConfigDir, "ccc")
	if err := os.MkdirAll(cccStateDir, 0755); err != nil {
		t.Fatal(err)
	}

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	// Set CCC_SUPERVISOR_HOOK=1 to bypass the external claude command call
	// This tests the hook's early return path without depending on claude availability
	cmd := exec.CommandContext(ctx, cccBinaryPath, "supervisor-hook")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir),
		"CCC_SUPERVISOR_HOOK=1",
	)
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	// Start the command
	if err := pm.start(cmd); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Send test input
	testInput := `{"session_id":"test-session-123","stop_hook_active":true}`
	if _, err := console.SendLine(testInput); err != nil {
		t.Errorf("failed to send input: %v", err)
	}

	// Should see the bypass output (empty object with decision omitted to allow stop)
	if _, err := console.ExpectString(`{}`); err != nil {
		t.Errorf("expected bypass output (empty object): %v", err)
	}

	// Wait for hook to complete
	pm.markWaited(cmd)
	if err := cmd.Wait(); err != nil {
		t.Logf("Command completed (may have exited): %v", err)
	}
}

// TestE2E_HelpShowsProviders tests that help shows available providers
func TestE2E_HelpShowsProviders(t *testing.T) {
	// Don't run in parallel - PTY tests can have resource conflicts
	// t.Parallel()

	pm := &processManager{}
	defer pm.cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	console, err := expect.NewConsole(expect.WithDefaultTimeout(5 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	cmd := exec.CommandContext(ctx, cccBinaryPath, "--help")
	cmd.Env = append(os.Environ(), fmt.Sprintf("CCC_CONFIG_DIR=%s", testConfigDir))
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := pm.start(cmd); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Help should show available providers while process is still running
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

	// Wait for help command to complete
	pm.markWaited(cmd)
	cmd.Wait()
}

// TestE2E_Timeout tests command timeout handling
func TestE2E_Timeout(t *testing.T) {
	t.Parallel()

	pm := &processManager{}
	defer pm.cleanup()

	// Use a shorter timeout for this specific test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	console, err := expect.NewConsole(
		expect.WithStdout(os.Stdout),
		expect.WithDefaultTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	cmd := exec.CommandContext(ctx, cccBinaryPath, "--version")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	if err := pm.start(cmd); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Check we got the expected output while process is running
	if _, err := console.ExpectString("claude-code-config-switcher version"); err != nil {
		t.Errorf("expected version string: %v", err)
	}

	// Wait for version command to complete
	pm.markWaited(cmd)
	cmd.Wait()
}
