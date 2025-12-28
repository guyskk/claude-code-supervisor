package validate

import (
	"fmt"
	"strings"
	"testing"
)

// mockConfig implements Config interface for testing.
type mockConfig struct {
	providers       map[string]map[string]interface{}
	currentProvider string
}

func (m *mockConfig) Providers() map[string]map[string]interface{} {
	return m.providers
}

func (m *mockConfig) CurrentProvider() string {
	return m.currentProvider
}

func TestValidateProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    *mockConfig
		provider  string
		testAPI   bool
		wantValid bool
		wantErrs  []string
	}{
		{
			name: "valid provider with all required fields",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"kimi": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
							"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
							"ANTHROPIC_MODEL":      "claude-3-5-sonnet-20241022",
						},
					},
				},
			},
			provider:  "kimi",
			testAPI:   false,
			wantValid: true,
			wantErrs:  nil,
		},
		{
			name: "provider missing ANTHROPIC_BASE_URL",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"glm": {
						"env": map[string]interface{}{
							"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
						},
					},
				},
			},
			provider:  "glm",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Missing required environment variable: ANTHROPIC_BASE_URL"},
		},
		{
			name: "provider missing ANTHROPIC_AUTH_TOKEN",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"m2": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
						},
					},
				},
			},
			provider:  "m2",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Missing required environment variable: ANTHROPIC_AUTH_TOKEN"},
		},
		{
			name: "provider with invalid URL format - no scheme",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"broken": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "not-a-valid-url",
							"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
						},
					},
				},
			},
			provider:  "broken",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Invalid Base URL format: must use http:// or https:// scheme"},
		},
		{
			name: "provider with invalid URL - missing host",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"broken": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://",
							"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
						},
					},
				},
			},
			provider:  "broken",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Invalid Base URL format: missing host"},
		},
		{
			name: "provider not found",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"kimi": {},
				},
			},
			provider:  "unknown",
			testAPI:   false,
			wantValid: false,
			wantErrs:  []string{"Provider 'unknown' not found in configuration"},
		},
		{
			name: "provider with minimal valid config",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"minimal": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.example.com",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
				},
			},
			provider:  "minimal",
			testAPI:   false,
			wantValid: true,
			wantErrs:  nil,
		},
		{
			name: "provider without env field",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"noenv": {},
				},
			},
			provider:  "noenv",
			testAPI:   false,
			wantValid: false,
			wantErrs: []string{
				"Missing required environment variable: ANTHROPIC_BASE_URL",
				"Missing required environment variable: ANTHROPIC_AUTH_TOKEN",
			},
		},
		{
			name: "provider with empty env values",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"empty": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "",
							"ANTHROPIC_AUTH_TOKEN": "",
						},
					},
				},
			},
			provider:  "empty",
			testAPI:   false,
			wantValid: false,
			wantErrs: []string{
				"Missing required environment variable: ANTHROPIC_BASE_URL",
				"Missing required environment variable: ANTHROPIC_AUTH_TOKEN",
			},
		},
		{
			name: "provider with http URL",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"http": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "http://api.example.com/anthropic",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
				},
			},
			provider:  "http",
			testAPI:   false,
			wantValid: true,
			wantErrs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateProvider(tt.config, tt.provider, tt.testAPI)

			if result.Valid != tt.wantValid {
				t.Errorf("ValidateProvider() Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if len(result.Errors) != len(tt.wantErrs) {
				t.Errorf("ValidateProvider() got %d errors, want %d", len(result.Errors), len(tt.wantErrs))
			}

			for i, wantErr := range tt.wantErrs {
				if i >= len(result.Errors) {
					t.Errorf("ValidateProvider() missing expected error %d: %q", i, wantErr)
					continue
				}
				if !strings.Contains(result.Errors[i], wantErr) {
					t.Errorf("ValidateProvider() error %d = %q, want to contain %q", i, result.Errors[i], wantErr)
				}
			}
		})
	}
}

func TestValidateAllProviders(t *testing.T) {
	tests := []struct {
		name        string
		config      *mockConfig
		testAPI     bool
		wantTotal   int
		wantValid   int
		wantInvalid int
	}{
		{
			name: "all providers valid",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"kimi": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
					"glm": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://open.bigmodel.cn/api/anthropic",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
				},
			},
			testAPI:     false,
			wantTotal:   2,
			wantValid:   2,
			wantInvalid: 0,
		},
		{
			name: "mixed valid and invalid providers",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"valid": {
						"env": map[string]interface{}{
							"ANTHROPIC_BASE_URL":   "https://api.example.com",
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
					"invalid": {
						"env": map[string]interface{}{
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
				},
			},
			testAPI:     false,
			wantTotal:   2,
			wantValid:   1,
			wantInvalid: 1,
		},
		{
			name: "all providers invalid",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{
					"broken1": {
						"env": map[string]interface{}{
							"ANTHROPIC_AUTH_TOKEN": "sk-test",
						},
					},
					"broken2": {},
				},
			},
			testAPI:     false,
			wantTotal:   2,
			wantValid:   0,
			wantInvalid: 2,
		},
		{
			name: "no providers configured",
			config: &mockConfig{
				providers: map[string]map[string]interface{}{},
			},
			testAPI:     false,
			wantTotal:   0,
			wantValid:   0,
			wantInvalid: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := ValidateAllProviders(tt.config, tt.testAPI)

			if summary.Total != tt.wantTotal {
				t.Errorf("ValidateAllProviders() Total = %v, want %v", summary.Total, tt.wantTotal)
			}
			if summary.Valid != tt.wantValid {
				t.Errorf("ValidateAllProviders() Valid = %v, want %v", summary.Valid, tt.wantValid)
			}
			if summary.Invalid != tt.wantInvalid {
				t.Errorf("ValidateAllProviders() Invalid = %v, want %v", summary.Invalid, tt.wantInvalid)
			}
		})
	}
}

func TestValidationResultFields(t *testing.T) {
	config := &mockConfig{
		providers: map[string]map[string]interface{}{
			"test": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://api.example.com",
					"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
					"ANTHROPIC_MODEL":      "claude-3-opus-20240229",
				},
			},
		},
	}

	result := ValidateProvider(config, "test", false)

	if result.Provider != "test" {
		t.Errorf("Result.Provider = %q, want %q", result.Provider, "test")
	}

	if result.BaseURL != "https://api.example.com" {
		t.Errorf("Result.BaseURL = %q, want %q", result.BaseURL, "https://api.example.com")
	}

	if result.Model != "claude-3-opus-20240229" {
		t.Errorf("Result.Model = %q, want %q", result.Model, "claude-3-opus-20240229")
	}

	if result.APIStatus != "skipped" {
		t.Errorf("Result.APIStatus = %q, want %q", result.APIStatus, "skipped")
	}
}

func TestRun(t *testing.T) {
	t.Run("validate all - all valid", func(t *testing.T) {
		config := &mockConfig{
			providers: map[string]map[string]interface{}{
				"kimi": {
					"env": map[string]interface{}{
						"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
						"ANTHROPIC_AUTH_TOKEN": "sk-test",
					},
				},
			},
		}

		opts := &RunOptions{
			ValidateAll: true,
			TestAPI:     false,
		}

		err := Run(config, opts)
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	})

	t.Run("validate all - has invalid", func(t *testing.T) {
		config := &mockConfig{
			providers: map[string]map[string]interface{}{
				"invalid": {
					"env": map[string]interface{}{},
				},
			},
		}

		opts := &RunOptions{
			ValidateAll: true,
			TestAPI:     false,
		}

		err := Run(config, opts)
		if err == nil {
			t.Error("Run() error = nil, want error")
		}
	})

	t.Run("validate specific provider - valid", func(t *testing.T) {
		config := &mockConfig{
			providers: map[string]map[string]interface{}{
				"kimi": {
					"env": map[string]interface{}{
						"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
						"ANTHROPIC_AUTH_TOKEN": "sk-test",
					},
				},
			},
		}

		opts := &RunOptions{
			Provider: "kimi",
			TestAPI:  false,
		}

		err := Run(config, opts)
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	})

	t.Run("validate specific provider - invalid", func(t *testing.T) {
		config := &mockConfig{
			providers: map[string]map[string]interface{}{
				"invalid": {
					"env": map[string]interface{}{},
				},
			},
		}

		opts := &RunOptions{
			Provider: "invalid",
			TestAPI:  false,
		}

		err := Run(config, opts)
		if err == nil {
			t.Error("Run() error = nil, want error")
		}
	})

	t.Run("validate current provider - not set", func(t *testing.T) {
		config := &mockConfig{
			providers: map[string]map[string]interface{}{
				"kimi": {
					"env": map[string]interface{}{
						"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
						"ANTHROPIC_AUTH_TOKEN": "sk-test",
					},
				},
			},
			currentProvider: "",
		}

		opts := &RunOptions{
			Provider: "",
			TestAPI:  false,
		}

		err := Run(config, opts)
		if err == nil {
			t.Error("Run() error = nil, want error")
		}
	})

	t.Run("validate current provider - set", func(t *testing.T) {
		config := &mockConfig{
			providers: map[string]map[string]interface{}{
				"kimi": {
					"env": map[string]interface{}{
						"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
						"ANTHROPIC_AUTH_TOKEN": "sk-test",
					},
				},
			},
			currentProvider: "kimi",
		}

		opts := &RunOptions{
			Provider: "",
			TestAPI:  false,
		}

		err := Run(config, opts)
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	})
}

func TestPrintResult(t *testing.T) {
	tests := []struct {
		name   string
		result *ValidationResult
	}{
		{
			name: "valid result",
			result: &ValidationResult{
				Provider:  "kimi",
				Valid:     true,
				BaseURL:   "https://api.moonshot.cn/anthropic",
				Model:     "claude-3-5-sonnet-20241022",
				APIStatus: "ok",
			},
		},
		{
			name: "invalid result",
			result: &ValidationResult{
				Provider: "broken",
				Valid:    false,
				Errors: []string{
					"Missing required environment variable: ANTHROPIC_BASE_URL",
				},
			},
		},
		{
			name: "warning result",
			result: &ValidationResult{
				Provider:  "kimi",
				Valid:     true,
				BaseURL:   "https://api.moonshot.cn/anthropic",
				Model:     "claude-3-5-sonnet-20241022",
				APIStatus: "failed: HTTP 503",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the function doesn't crash
			PrintResult(tt.result)
		})
	}
}

func TestPrintSummary(t *testing.T) {
	tests := []struct {
		name    string
		summary *ValidationSummary
	}{
		{
			name: "all valid",
			summary: &ValidationSummary{
				Total:   3,
				Valid:   3,
				Invalid: 0,
				Warning: 0,
			},
		},
		{
			name: "some invalid",
			summary: &ValidationSummary{
				Total:   3,
				Valid:   2,
				Invalid: 1,
				Warning: 0,
			},
		},
		{
			name: "some warnings",
			summary: &ValidationSummary{
				Total:   3,
				Valid:   3,
				Invalid: 0,
				Warning: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the function doesn't crash
			PrintSummary(tt.summary)
		})
	}
}

// Test testAPIConnection with various scenarios
func TestTestAPIConnection(t *testing.T) {
	t.Run("invalid URL format", func(t *testing.T) {
		status := testAPIConnection("://invalid-url", "sk-test")
		if !strings.Contains(status, "failed") {
			t.Errorf("testAPIConnection() = %q, want contains 'failed'", status)
		}
	})

	t.Run("unreachable URL", func(t *testing.T) {
		status := testAPIConnection("http://localhost:9999/anthropic", "sk-test")
		if !strings.Contains(status, "failed") {
			t.Errorf("testAPIConnection() = %q, want contains 'failed'", status)
		}
	})
}

// Example mockConfig helper function for tests
func newMockConfig(providers map[string]map[string]interface{}, current string) *mockConfig {
	return &mockConfig{
		providers:       providers,
		currentProvider: current,
	}
}

func ExampleValidateProvider() {
	config := newMockConfig(
		map[string]map[string]interface{}{
			"kimi": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
					"ANTHROPIC_AUTH_TOKEN": "sk-test-token",
					"ANTHROPIC_MODEL":      "claude-3-5-sonnet-20241022",
				},
			},
		},
		"",
	)

	result := ValidateProvider(config, "kimi", false)
	fmt.Printf("Provider: %s, Valid: %v\n", result.Provider, result.Valid)
	// Output: Provider: kimi, Valid: true
}

func ExampleValidateAllProviders() {
	config := newMockConfig(
		map[string]map[string]interface{}{
			"kimi": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://api.moonshot.cn/anthropic",
					"ANTHROPIC_AUTH_TOKEN": "sk-test",
				},
			},
			"glm": {
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL":   "https://open.bigmodel.cn/api/anthropic",
					"ANTHROPIC_AUTH_TOKEN": "sk-test",
				},
			},
		},
		"",
	)

	summary := ValidateAllProviders(config, false)
	fmt.Printf("Total: %d, Valid: %d, Invalid: %d\n", summary.Total, summary.Valid, summary.Invalid)
	// Output: Total: 2, Valid: 2, Invalid: 0
}
