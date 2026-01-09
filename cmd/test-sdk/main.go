// test-sdk is a diagnostic tool to verify Claude Agent SDK configuration.
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
	fmt.Println("=== Claude Agent SDK Configuration Test ===")

	// Test 1: Verify options structure
	fmt.Println("Test 1: SDK Options Structure")
	opts := types.NewClaudeAgentOptions().
		WithForkSession(true).
		WithResume("test-session-123").
		WithSettingSources(types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal)

	// Add environment variables
	opts.Env = map[string]string{
		"CCC_SUPERVISOR_HOOK": "1",
	}

	// Print options as JSON for verification
	optsJSON, _ := json.MarshalIndent(opts, "", "  ")
	fmt.Printf("Options:\n%s\n\n", string(optsJSON))

	// Test 2: Check if setting sources are correctly set
	fmt.Println("Test 2: Setting Sources")
	if len(opts.SettingSources) == 3 {
		fmt.Printf("✓ SettingSources set correctly: %v\n\n", opts.SettingSources)
	} else {
		fmt.Printf("✗ SettingSources incorrect: got %d, want 3\n\n", len(opts.SettingSources))
	}

	// Test 3: Verify environment variables
	fmt.Println("Test 3: Environment Variables")
	if opts.Env != nil {
		if val, ok := opts.Env["CCC_SUPERVISOR_HOOK"]; ok && val == "1" {
			fmt.Printf("✓ CCC_SUPERVISOR_HOOK = 1\n\n")
		} else {
			fmt.Printf("✗ CCC_SUPERVISOR_HOOK not set correctly\n\n")
		}
	} else {
		fmt.Printf("✗ Env map is nil\n\n")
	}

	// Test 4: Try to create a query (will fail without real claude, but we can catch the error)
	fmt.Println("Test 4: SDK Query Creation (Expected to fail without real session)")
	ctx := context.Background()
	testPrompt := "Hello, this is a test."

	// This will fail because there's no real session, but we can see if the SDK
	// properly constructs the command
	messages, err := claude.Query(ctx, testPrompt, opts)
	if err != nil {
		fmt.Printf("✓ Query creation failed as expected (no real session): %v\n\n", err)
	} else {
		// If somehow it succeeds, drain the channel
		for range messages {
		}
		fmt.Printf("✗ Query succeeded unexpectedly\n\n")
	}

	// Test 5: Show what command would be executed
	fmt.Println("Test 5: Simulated Command Line")
	fmt.Printf("If SDK were to execute, it would run something like:\n")
	fmt.Printf("  claude --print --setting-sources user,project,local --resume test-session-123 --fork-session\n\n")

	// Test 6: Verify SettingSource constants
	fmt.Println("Test 6: SettingSource Constants")
	fmt.Printf("  SettingSourceUser    = %q\n", types.SettingSourceUser)
	fmt.Printf("  SettingSourceProject = %q\n", types.SettingSourceProject)
	fmt.Printf("  SettingSourceLocal   = %q\n\n", types.SettingSourceLocal)

	fmt.Println("=== Test Complete ===")
	fmt.Println("\nNote: Full integration test requires:")
	fmt.Println("  1. A running claude CLI in PATH")
	fmt.Println("  2. A valid session ID")
	fmt.Println("  3. Valid API credentials")
}
