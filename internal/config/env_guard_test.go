package config

import (
	"sort"
	"strings"
	"testing"
)

// sortConflicts returns the conflict keys sorted, used for deterministic comparison.
func sortConflicts(conflicts []EnvConflict) []EnvConflict {
	sorted := make([]EnvConflict, len(conflicts))
	copy(sorted, conflicts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	return sorted
}

func TestDetectSettingsEnvConflicts(t *testing.T) {
	tests := []struct {
		name           string
		userSettings   map[string]interface{}
		managedEnvKeys map[string]bool
		wantKeys       []string          // expected keys (sorted)
		wantReasons    map[string]string // expected reasons keyed by conflict key (substring match)
	}{
		{
			name:           "nil userSettings returns nil",
			userSettings:   nil,
			managedEnvKeys: map[string]bool{"X": true},
			wantKeys:       nil,
		},
		{
			name:           "no env field returns nil",
			userSettings:   map[string]interface{}{"permissions": map[string]interface{}{}},
			managedEnvKeys: map[string]bool{"X": true},
			wantKeys:       nil,
		},
		{
			name: "non-conflicting custom var returns nil",
			userSettings: map[string]interface{}{
				"env": map[string]interface{}{
					"MY_CUSTOM_VAR": "value",
				},
			},
			managedEnvKeys: map[string]bool{},
			wantKeys:       nil,
		},
		{
			name: "ANTHROPIC_BASE_URL detected as prefix",
			userSettings: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL": "https://example.com",
				},
			},
			managedEnvKeys: map[string]bool{},
			wantKeys:       []string{"ANTHROPIC_BASE_URL"},
			wantReasons: map[string]string{
				"ANTHROPIC_BASE_URL": "prefix",
			},
		},
		{
			name: "CLAUDE_CODE_MAX_OUTPUT_TOKENS detected as prefix",
			userSettings: map[string]interface{}{
				"env": map[string]interface{}{
					"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "8192",
				},
			},
			managedEnvKeys: map[string]bool{},
			wantKeys:       []string{"CLAUDE_CODE_MAX_OUTPUT_TOKENS"},
			wantReasons: map[string]string{
				"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "prefix",
			},
		},
		{
			name: "non-prefix managed key hits managed reason",
			userSettings: map[string]interface{}{
				"env": map[string]interface{}{
					"API_TIMEOUT": "30000",
				},
			},
			managedEnvKeys: map[string]bool{"API_TIMEOUT": true},
			wantKeys:       []string{"API_TIMEOUT"},
			wantReasons: map[string]string{
				"API_TIMEOUT": "managed",
			},
		},
		{
			name: "mixed: prefix + custom (allowed) + managed",
			userSettings: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_MODEL": "claude-3",
					"MY_CUSTOM_VAR":   "value",
					"API_TIMEOUT":     "30000",
				},
			},
			managedEnvKeys: map[string]bool{"API_TIMEOUT": true},
			wantKeys:       []string{"ANTHROPIC_MODEL", "API_TIMEOUT"},
			wantReasons: map[string]string{
				"ANTHROPIC_MODEL": "prefix",
				"API_TIMEOUT":     "managed",
			},
		},
		{
			name: "prefix is strict: MYANTHROPIC_X not a hit",
			userSettings: map[string]interface{}{
				"env": map[string]interface{}{
					"MYANTHROPIC_X": "x",
					"ANTHROPIC_":    "edge",
					"CLAUDE_":       "edge",
				},
			},
			managedEnvKeys: map[string]bool{},
			wantKeys:       []string{"ANTHROPIC_", "CLAUDE_"},
			wantReasons: map[string]string{
				"ANTHROPIC_": "prefix",
				"CLAUDE_":    "prefix",
			},
		},
		{
			name: "env is not a map returns nil",
			userSettings: map[string]interface{}{
				"env": "not a map",
			},
			managedEnvKeys: map[string]bool{},
			wantKeys:       nil,
		},
		{
			name: "ANTHROPIC_BASE_URL also in managed: still reports as prefix",
			userSettings: map[string]interface{}{
				"env": map[string]interface{}{
					"ANTHROPIC_BASE_URL": "x",
				},
			},
			managedEnvKeys: map[string]bool{"ANTHROPIC_BASE_URL": true},
			wantKeys:       []string{"ANTHROPIC_BASE_URL"},
			wantReasons: map[string]string{
				"ANTHROPIC_BASE_URL": "prefix",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSettingsEnvConflicts(tt.userSettings, tt.managedEnvKeys)
			if len(tt.wantKeys) == 0 {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if len(got) != len(tt.wantKeys) {
				t.Fatalf("conflict count = %d, want %d (got: %+v)", len(got), len(tt.wantKeys), got)
			}

			sortedGot := sortConflicts(got)
			for i, key := range tt.wantKeys {
				if sortedGot[i].Key != key {
					t.Errorf("conflict[%d].Key = %q, want %q", i, sortedGot[i].Key, key)
				}
				if want, ok := tt.wantReasons[key]; ok {
					if !strings.Contains(strings.ToLower(sortedGot[i].Reason), want) {
						t.Errorf("conflict[%s].Reason = %q, want it to contain %q", key, sortedGot[i].Reason, want)
					}
				}
			}
		})
	}
}

func TestFormatEnvConflictError(t *testing.T) {
	settingsPath := "/home/user/.claude/settings.json"
	configPath := "/home/user/.claude/ccc.json"

	t.Run("error contains all conflict keys", func(t *testing.T) {
		conflicts := []EnvConflict{
			{Key: "ANTHROPIC_BASE_URL", Reason: "anthropic/claude prefix"},
			{Key: "API_TIMEOUT", Reason: "managed by provider/base"},
		}
		msg := FormatEnvConflictError(settingsPath, configPath, conflicts)

		for _, key := range []string{"ANTHROPIC_BASE_URL", "API_TIMEOUT"} {
			if !strings.Contains(msg, key) {
				t.Errorf("error message should contain key %q, got:\n%s", key, msg)
			}
		}
	})

	t.Run("error does not contain any value (secret redaction)", func(t *testing.T) {
		conflicts := []EnvConflict{
			{Key: "ANTHROPIC_AUTH_TOKEN", Reason: "anthropic/claude prefix"},
		}
		msg := FormatEnvConflictError(settingsPath, configPath, conflicts)

		// Common patterns that would indicate value leak
		forbiddenSubstrings := []string{
			"sk-",      // typical token prefix
			"Bearer ",  // header form
			"https://", // URL form (would leak base url)
		}
		for _, bad := range forbiddenSubstrings {
			if strings.Contains(msg, bad) {
				t.Errorf("error message must not contain %q (value leak risk), got:\n%s", bad, msg)
			}
		}
	})

	t.Run("error contains both paths and remediation guidance", func(t *testing.T) {
		conflicts := []EnvConflict{
			{Key: "ANTHROPIC_MODEL", Reason: "anthropic/claude prefix"},
		}
		msg := FormatEnvConflictError(settingsPath, configPath, conflicts)

		// Must include settings.json path
		if !strings.Contains(msg, settingsPath) {
			t.Errorf("error message should reference settings path %q, got:\n%s", settingsPath, msg)
		}
		// Must reference ccc.json (so users know where to put provider config)
		if !strings.Contains(msg, configPath) {
			t.Errorf("error message should reference config path %q, got:\n%s", configPath, msg)
		}
		// Must explain WHY (provider switch failure)
		lowered := strings.ToLower(msg)
		if !strings.Contains(lowered, "provider") {
			t.Errorf("error message should explain why (mention provider), got:\n%s", msg)
		}
	})

	t.Run("empty conflicts returns empty string", func(t *testing.T) {
		msg := FormatEnvConflictError(settingsPath, configPath, nil)
		if msg != "" {
			t.Errorf("expected empty string for nil conflicts, got: %q", msg)
		}
	})
}
