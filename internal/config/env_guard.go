package config

import (
	"fmt"
	"sort"
	"strings"
)

// EnvConflict describes a single env key in settings.json that conflicts with ccc's
// provider/base env management. Conflicts must be resolved by the user (ccc never
// modifies settings.json on the user's behalf).
type EnvConflict struct {
	// Key is the env variable name found in settings.json's "env" field.
	Key string
	// Reason explains why the key is considered a conflict. The text is human-readable
	// and intended to appear in error messages presented to the user.
	Reason string
}

const (
	reasonAnthropicClaudePrefix = "anthropic/claude prefix"
	reasonManagedByProvider     = "managed by provider/base"
)

// DetectSettingsEnvConflicts inspects the env map inside the user's settings.json
// and returns every key that conflicts with ccc's env management.
//
// A key is considered conflicting if either:
//   - it starts with "ANTHROPIC_" or "CLAUDE_" (these are reserved by Claude Code
//     itself and will override env passed by ccc via syscall.Exec); or
//   - it is present in managedEnvKeys (i.e., it is also set in ccc.json's base/provider
//     env, where the settings.json value would silently override the ccc one).
//
// Returns nil when userSettings is nil, contains no "env" field, env is not a map,
// or no conflicts are found.
func DetectSettingsEnvConflicts(userSettings map[string]interface{}, managedEnvKeys map[string]bool) []EnvConflict {
	envMap := GetEnv(userSettings)
	if envMap == nil {
		return nil
	}

	var conflicts []EnvConflict
	for key := range envMap {
		if strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "CLAUDE_") {
			conflicts = append(conflicts, EnvConflict{Key: key, Reason: reasonAnthropicClaudePrefix})
			continue
		}
		if managedEnvKeys[key] {
			conflicts = append(conflicts, EnvConflict{Key: key, Reason: reasonManagedByProvider})
		}
	}

	if len(conflicts) == 0 {
		return nil
	}

	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].Key < conflicts[j].Key
	})
	return conflicts
}

// FormatEnvConflictError builds a multi-line error message that lists every conflicting
// key (without its value, to avoid leaking secrets like ANTHROPIC_AUTH_TOKEN) and tells
// the user exactly how to fix the conflict themselves. ccc deliberately does not auto-fix
// settings.json — configuration conflicts are the user's responsibility.
//
// Returns an empty string when conflicts is empty so callers can treat that as "no error".
func FormatEnvConflictError(settingsPath, configPath string, conflicts []EnvConflict) string {
	if len(conflicts) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("settings.json env conflicts with provider configuration:\n")
	b.WriteString(fmt.Sprintf("  file: %s\n", settingsPath))
	b.WriteString("  conflicting keys:\n")
	for _, c := range conflicts {
		b.WriteString(fmt.Sprintf("    - %s  (%s)\n", c.Key, c.Reason))
	}
	b.WriteString("\nWhy this is a problem:\n")
	b.WriteString("  Claude Code's settings.json \"env\" field overrides environment variables\n")
	b.WriteString("  passed by ccc when launching claude. The keys above would silently override\n")
	b.WriteString("  the provider env that ccc passes via command line, causing the provider\n")
	b.WriteString("  switch to use the wrong base_url / token / model.\n")
	b.WriteString("\nHow to fix:\n")
	b.WriteString(fmt.Sprintf("  1. Remove the keys above from the \"env\" field in %s.\n", settingsPath))
	b.WriteString(fmt.Sprintf("  2. Put provider-related config under providers.<name>.env in %s instead.\n", configPath))
	b.WriteString("  ccc will refuse to start claude until this conflict is resolved.\n")
	return b.String()
}
