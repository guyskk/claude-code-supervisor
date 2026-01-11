package cli

import (
	"os"
	"testing"

	"github.com/guyskk/ccc/internal/config"
)

func setupTestDir(t *testing.T) func() {
	t.Helper()

	// Save original function
	originalFunc := config.GetDirFunc

	// Create temp directory
	tmpDir := t.TempDir()

	// Override GetDirFunc
	config.GetDirFunc = func() string {
		return tmpDir
	}

	// Return cleanup function
	cleanup := func() {
		config.GetDirFunc = originalFunc
	}

	return cleanup
}

func TestParse(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		wantVersion      bool
		wantHelp         bool
		wantValidate     bool
		wantValidateAll  bool
		wantValidateProv string
		wantProvider     string
		wantClaudeArgs   []string
	}{
		{
			name:        "--version flag",
			args:        []string{"--version"},
			wantVersion: true,
		},
		{
			name:        "-v flag",
			args:        []string{"-v"},
			wantVersion: true,
		},
		{
			name:     "--help flag",
			args:     []string{"--help"},
			wantHelp: true,
		},
		{
			name:     "-h flag",
			args:     []string{"-h"},
			wantHelp: true,
		},
		{
			name:           "provider specified",
			args:           []string{"kimi", "/path/to/project"},
			wantProvider:   "kimi",
			wantClaudeArgs: []string{"/path/to/project"},
		},
		{
			name:           "provider with no args",
			args:           []string{"glm"},
			wantProvider:   "glm",
			wantClaudeArgs: []string{},
		},
		{
			name:           "no args - use current provider",
			args:           []string{},
			wantProvider:   "",
			wantClaudeArgs: []string{},
		},
		{
			name:           "flags only - all passed to claude",
			args:           []string{"--debug", "--verbose"},
			wantProvider:   "",
			wantClaudeArgs: []string{"--debug", "--verbose"},
		},
		{
			name:           "single flag - passed to claude",
			args:           []string{"--debug"},
			wantProvider:   "",
			wantClaudeArgs: []string{"--debug"},
		},
		{
			name:         "validate command",
			args:         []string{"validate"},
			wantValidate: true,
		},
		{
			name:             "validate with provider",
			args:             []string{"validate", "kimi"},
			wantValidate:     true,
			wantValidateProv: "kimi",
		},
		{
			name:            "validate with --all",
			args:            []string{"validate", "--all"},
			wantValidate:    true,
			wantValidateAll: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Parse(tt.args)

			if cmd.Version != tt.wantVersion {
				t.Errorf("Version = %v, want %v", cmd.Version, tt.wantVersion)
			}
			if cmd.Help != tt.wantHelp {
				t.Errorf("Help = %v, want %v", cmd.Help, tt.wantHelp)
			}
			if cmd.Validate != tt.wantValidate {
				t.Errorf("Validate = %v, want %v", cmd.Validate, tt.wantValidate)
			}
			if cmd.ValidateOpts != nil {
				if cmd.ValidateOpts.Provider != tt.wantValidateProv {
					t.Errorf("ValidateOpts.Provider = %q, want %q", cmd.ValidateOpts.Provider, tt.wantValidateProv)
				}
				if cmd.ValidateOpts.ValidateAll != tt.wantValidateAll {
					t.Errorf("ValidateOpts.ValidateAll = %v, want %v", cmd.ValidateOpts.ValidateAll, tt.wantValidateAll)
				}
			}
			if cmd.Provider != tt.wantProvider {
				t.Errorf("Provider = %q, want %q", cmd.Provider, tt.wantProvider)
			}
			if len(cmd.ClaudeArgs) != len(tt.wantClaudeArgs) {
				t.Errorf("ClaudeArgs length = %d, want %d", len(cmd.ClaudeArgs), len(tt.wantClaudeArgs))
			}
		})
	}
}

func TestShowVersion(t *testing.T) {
	// Capture stdout
	old := Version
	Version = "test-v1.0.0"
	defer func() { Version = old }()

	// This just verifies the function doesn't crash
	ShowVersion()
}

func TestShowHelp(t *testing.T) {
	t.Run("with valid config", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		cfg := &config.Config{
			CurrentProvider: "kimi",
			Providers: map[string]map[string]interface{}{
				"kimi": {},
				"glm":  {},
			},
		}

		// This just verifies the function doesn't crash
		ShowHelp(cfg, nil)
	})

	t.Run("with config error", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		err := os.ErrNotExist
		ShowHelp(nil, err)
	})

	t.Run("with nil config", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		ShowHelp(nil, nil)
	})
}

func TestDetermineProvider(t *testing.T) {
	cfg := &config.Config{
		CurrentProvider: "kimi",
		Providers: map[string]map[string]interface{}{
			"kimi": {},
			"glm":  {},
		},
	}

	tests := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "valid provider specified",
			cmd: &Command{
				Provider: "glm",
			},
			want: "glm",
		},
		{
			name: "invalid provider, use current",
			cmd: &Command{
				Provider: "unknown",
			},
			want: "kimi",
		},
		{
			name: "no provider specified, use current",
			cmd: &Command{
				Provider: "",
			},
			want: "kimi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineProvider(tt.cmd, cfg)
			if got != tt.want {
				t.Errorf("determineProvider() = %q, want %q", got, tt.want)
			}
		})
	}

	// Separate test for "no current" case with different cfg
	t.Run("no provider specified, no current, use first", func(t *testing.T) {
		cfg := &config.Config{
			CurrentProvider: "",
			Providers: map[string]map[string]interface{}{
				"kimi": {},
				"glm":  {},
			},
		}
		cmd := &Command{Provider: ""}
		got := determineProvider(cmd, cfg)
		// Since map iteration order is random, just check it's one of the valid providers
		if got != "kimi" && got != "glm" {
			t.Errorf("determineProvider() = %q, want kimi or glm", got)
		}
	})

	t.Run("invalid provider and no current", func(t *testing.T) {
		cfg := &config.Config{
			CurrentProvider: "",
			Providers: map[string]map[string]interface{}{
				"kimi": {},
			},
		}

		cmd := &Command{Provider: "unknown"}
		got := determineProvider(cmd, cfg)
		if got != "" {
			t.Errorf("determineProvider() = %q, want empty string", got)
		}
	})

	t.Run("no providers configured", func(t *testing.T) {
		cfg := &config.Config{
			CurrentProvider: "",
			Providers:       map[string]map[string]interface{}{},
		}

		cmd := &Command{Provider: ""}
		got := determineProvider(cmd, cfg)
		if got != "" {
			t.Errorf("determineProvider() = %q, want empty string", got)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("--version", func(t *testing.T) {
		cmd := &Command{Version: true}
		if err := Run(cmd); err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})

	t.Run("--help", func(t *testing.T) {
		cleanup := setupTestDir(t)
		defer cleanup()

		cmd := &Command{Help: true}
		if err := Run(cmd); err != nil {
			t.Errorf("Run() error = %v", err)
		}
	})
}

func TestConstants(t *testing.T) {
	if Name != "claude-code-supervisor" {
		t.Errorf("Name = %s, want claude-code-supervisor", Name)
	}
}
