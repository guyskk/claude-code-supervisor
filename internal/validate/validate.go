// Package validate provides configuration validation functionality for ccc.
package validate

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ValidationResult represents the result of validating a provider configuration.
type ValidationResult struct {
	Provider  string
	Valid     bool
	Warnings  []string
	Errors    []string
	BaseURL   string
	Model     string
	APIStatus string // "ok", "failed", "skipped"
	APIError  error
}

// ValidationSummary represents the summary of validating multiple providers.
type ValidationSummary struct {
	Total   int
	Valid   int
	Invalid int
	Warning int
	Results []*ValidationResult
}

// Provider represents a provider configuration for validation.
type Provider interface {
	// Name returns the provider name.
	Name() string
	// Env returns the environment variables map.
	Env() map[string]interface{}
}

// Config represents the configuration interface for validation.
type Config interface {
	// Providers returns all providers.
	Providers() map[string]map[string]interface{}
	// CurrentProvider returns the current provider name.
	CurrentProvider() string
}

// ValidateProvider validates a single provider configuration.
func ValidateProvider(cfg Config, providerName string) *ValidationResult {
	result := &ValidationResult{
		Provider:  providerName,
		Valid:     true,
		Warnings:  []string{},
		Errors:    []string{},
		APIStatus: "",
	}

	providers := cfg.Providers()

	// Check if provider exists
	provider, exists := providers[providerName]
	if !exists {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Provider '%s' not found in configuration", providerName))
		return result
	}

	// Extract env from provider config
	var env map[string]interface{}
	if envVal, ok := provider["env"]; ok {
		if envMap, ok := envVal.(map[string]interface{}); ok {
			env = envMap
		}
	}

	if env == nil {
		env = make(map[string]interface{})
	}

	// Check required environment variables
	baseURL, hasBaseURL := env["ANTHROPIC_BASE_URL"].(string)
	authToken, hasAuthToken := env["ANTHROPIC_AUTH_TOKEN"].(string)

	if !hasBaseURL || baseURL == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Missing required environment variable: ANTHROPIC_BASE_URL")
	} else {
		result.BaseURL = baseURL
		// Validate URL format - must be http or https
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid Base URL format: %v", err))
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			result.Valid = false
			result.Errors = append(result.Errors, "Invalid Base URL format: must use http:// or https:// scheme")
		} else if parsedURL.Host == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "Invalid Base URL format: missing host")
		}
	}

	if !hasAuthToken || authToken == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Missing required environment variable: ANTHROPIC_AUTH_TOKEN")
	}

	// Check model if present
	if model, ok := env["ANTHROPIC_MODEL"].(string); ok {
		result.Model = model
	}

	// Test API connection if config is valid so far
	if result.Valid && hasBaseURL && hasAuthToken {
		result.APIStatus = testAPIConnection(baseURL, authToken)
	}

	return result
}

// testAPIConnection tests if the API endpoint is reachable using a minimal request.
func testAPIConnection(baseURL, authToken string) string {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Construct the correct /v1/messages endpoint
	apiURL := strings.TrimSuffix(baseURL, "/") + "/v1/messages"

	// Minimal test request body - simple question for faster response
	body := `{"model":"test","max_tokens":10,"messages":[{"role":"user","content":"直接回答 1+1=?"}]}`

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(body))
	if err != nil {
		return fmt.Sprintf("failed: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response content (limited length)
	buf, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
	respStr := strings.TrimSpace(string(buf))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "ok"
	}

	// Error format: HTTP {code} {response}
	if respStr != "" {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, respStr)
	}
	return fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// ValidateAllProviders validates all configured providers.
func ValidateAllProviders(cfg Config) *ValidationSummary {
	providers := cfg.Providers()
	summary := &ValidationSummary{
		Total:   len(providers),
		Results: make([]*ValidationResult, 0, len(providers)),
	}

	for providerName := range providers {
		result := ValidateProvider(cfg, providerName)
		summary.Results = append(summary.Results, result)

		if result.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}

		// Check if API status indicates a failure (starts with "HTTP" or "failed")
		if result.APIStatus != "" && result.APIStatus != "ok" {
			summary.Warning++
		}
	}

	return summary
}

// PrintResult prints a single validation result with color coding.
func PrintResult(result *ValidationResult) {
	status := "Valid"
	statusColor := "\033[32m" // green
	if !result.Valid {
		status = "Invalid"
		statusColor = "\033[31m" // red
	} else if result.APIStatus != "" && result.APIStatus != "ok" {
		status = "Warning"
		statusColor = "\033[33m" // yellow
	}

	fmt.Printf("  %s%s\033[0m: %s\n", statusColor, status, result.Provider)

	if result.BaseURL != "" {
		fmt.Printf("    Base URL: %s\n", result.BaseURL)
	}
	if result.Model != "" {
		fmt.Printf("    Model: %s\n", result.Model)
	}
	if result.APIStatus != "" {
		apiStatus := "OK"
		apiColor := "\033[32m"
		if result.APIStatus != "ok" {
			apiStatus = result.APIStatus
			apiColor = "\033[33m"
		}
		fmt.Printf("    API connection: %s%s\033[0m\n", apiColor, apiStatus)
	}

	for _, warning := range result.Warnings {
		fmt.Printf("    Warning: %s\n", warning)
	}
	for _, err := range result.Errors {
		fmt.Printf("    Error: %s\n", err)
	}
}

// PrintSummary prints the validation summary for all providers.
func PrintSummary(summary *ValidationSummary) {
	fmt.Println()
	if summary.Invalid > 0 {
		fmt.Printf("\033[31m%d/%d\033[0m providers invalid\n", summary.Invalid, summary.Total)
	} else if summary.Warning > 0 {
		fmt.Printf("\033[33mAll providers valid (%d with API warnings)\033[0m\n", summary.Warning)
	} else {
		fmt.Println("\033[32mAll providers valid\033[0m")
	}
}

// RunOptions represents the options for running validation.
type RunOptions struct {
	Provider    string // Empty means current provider
	ValidateAll bool
}

// Run executes the validation command with the given options.
func Run(cfg Config, opts *RunOptions) error {
	// Handle validate all
	if opts.ValidateAll {
		if len(cfg.Providers()) == 0 {
			fmt.Println("No providers configured")
			return nil
		}

		fmt.Printf("Validating %d provider(s)...\n\n", len(cfg.Providers()))
		summary := ValidateAllProviders(cfg)

		for _, result := range summary.Results {
			PrintResult(result)
		}

		PrintSummary(summary)

		// Return error if any provider is invalid or API test failed
		if summary.Invalid > 0 {
			return fmt.Errorf("%d provider(s) invalid", summary.Invalid)
		}
		if summary.Warning > 0 {
			return fmt.Errorf("%d provider(s) with API failures", summary.Warning)
		}
		return nil
	}

	// Determine which provider to validate
	providerName := opts.Provider
	if providerName == "" {
		providerName = cfg.CurrentProvider()
	}

	if providerName == "" {
		fmt.Println("No current provider set")
		if len(cfg.Providers()) > 0 {
			fmt.Println("\nAvailable providers:")
			for name := range cfg.Providers() {
				fmt.Printf("  %s\n", name)
			}
		}
		return fmt.Errorf("no provider specified")
	}

	result := ValidateProvider(cfg, providerName)
	PrintResult(result)

	if !result.Valid {
		return fmt.Errorf("provider '%s' is invalid", providerName)
	}

	// Also return error if API connection failed
	if result.APIStatus != "" && result.APIStatus != "ok" {
		return fmt.Errorf("provider '%s' API test failed: %s", providerName, result.APIStatus)
	}

	return nil
}
