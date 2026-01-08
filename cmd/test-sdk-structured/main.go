// test-sdk-structured tests Claude Agent SDK's StructuredOutput tool behavior
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
	// Check for API key
	if os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" && os.Getenv("CLAUDE_API_KEY") == "" {
		fmt.Println("Error: ANTHROPIC_AUTH_TOKEN or CLAUDE_API_KEY environment variable required")
		os.Exit(1)
	}

	fmt.Println("=== Claude Agent SDK StructuredOutput Tool Test ===\n")

	// Test prompt that asks for StructuredOutput usage
	testPrompt := `You must use the StructuredOutput tool to provide your response.

The schema should be: {"test_field": string, "success": boolean}

Please call the StructuredOutput tool with: {"test_field": "hello world", "success": true}`

	fmt.Println("Test Prompt:")
	fmt.Println(testPrompt)
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Create options
	opts := types.NewClaudeAgentOptions().
		WithVerbose(true).
		WithSettingSources(types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println("Sending query to Claude...")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	// Execute query
	messages, err := claude.Query(ctx, testPrompt, opts)
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
		os.Exit(1)
	}

	// Process messages with detailed logging
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Message Stream Analysis:")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	messageCount := 0
	for msg := range messages {
		messageCount++

		switch m := msg.(type) {
		case *types.UserMessage:
			fmt.Printf("[%d] UserMessage\n", messageCount)
			if strContent, ok := m.Content.(string); ok {
				fmt.Printf("  Content: %s\n", truncate(strContent, 100))
			} else {
				contentJSON, _ := json.Marshal(m.Content)
				fmt.Printf("  Content (structured): %s\n", truncate(string(contentJSON), 200))
			}

		case *types.AssistantMessage:
			fmt.Printf("[%d] AssistantMessage\n", messageCount)
			for i, block := range m.Content {
				switch b := block.(type) {
				case *types.TextBlock:
					fmt.Printf("  Block[%d]: TextBlock\n", i)
					fmt.Printf("    Text: %s\n", truncate(b.Text, 200))

				case *types.ToolUseBlock:
					fmt.Printf("  Block[%d]: ToolUseBlock\n", i)
					fmt.Printf("    Name: %s\n", b.Name)
					fmt.Printf("    ID: %s\n", b.ID)
					inputJSON, _ := json.MarshalIndent(b.Input, "    ", "  ")
					fmt.Printf("    Input: %s\n", string(inputJSON))

					// Check if this is a StructuredOutput tool call
					if b.Name == "structured_output" || b.Name == "StructuredOutput" {
						fmt.Printf("    *** This is a StructuredOutput tool call! ***\n")

						// Extract the structured data
						if response, ok := b.Input["response"].(map[string]interface{}); ok {
							responseJSON, _ := json.MarshalIndent(response, "      ", "  ")
							fmt.Printf("    Response data: %s\n", string(responseJSON))
						}
					}

				case *types.ToolResultBlock:
					fmt.Printf("  Block[%d]: ToolResultBlock\n", i)
					fmt.Printf("    ToolUseID: %s\n", b.ToolUseID)
					fmt.Printf("    IsError: %v\n", b.IsError)
					contentJSON, _ := json.Marshal(b.Content)
					fmt.Printf("    Content: %s\n", truncate(string(contentJSON), 200))

				default:
					fmt.Printf("  Block[%d]: Unknown type %T\n", i, block)
				}
			}

		case *types.ResultMessage:
			fmt.Printf("[%d] ResultMessage\n", messageCount)
			fmt.Printf("  Subtype: %s\n", m.Subtype)
			fmt.Printf("  SessionID: %s\n", m.SessionID)
			fmt.Printf("  DurationMs: %d\n", m.DurationMs)
			if m.TotalCostUSD != nil {
				fmt.Printf("  TotalCostUSD: %.4f\n", *m.TotalCostUSD)
			}
			if m.Result != nil {
				resultJSON, _ := json.MarshalIndent(m.Result, "  ", "  ")
				fmt.Printf("  Result: %s\n", string(resultJSON))
				fmt.Printf("  *** ResultMessage.Result field contains data! ***\n")
			}

		case *types.SystemMessage:
			fmt.Printf("[%d] SystemMessage\n", messageCount)
			fmt.Printf("  Subtype: %s\n", m.Subtype)

		default:
			fmt.Printf("[%d] Unknown message type: %T\n", messageCount, msg)
			msgJSON, _ := json.Marshal(msg)
			fmt.Printf("  Raw: %s\n", truncate(string(msgJSON), 300))
		}
		fmt.Println()
	}

	fmt.Printf("\n=== Test Complete. Total messages: %d ===\n", messageCount)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
