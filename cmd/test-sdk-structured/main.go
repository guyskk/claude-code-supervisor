// test-sdk-structured tests Claude Agent SDK's StructuredOutput feature
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/guyskk/ccc/internal/config"
	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}
	rawEnv := cfg.Providers["glm"]["env"]
	fmt.Println("=== Claude Agent SDK StructuredOutput Test ===")
	fmt.Println(rawEnv)

	// Define JSON schema for structured output
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_result": map[string]interface{}{
				"type":        "string",
				"description": "Description of what was accomplished",
			},
			"file_count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of source code files in the project",
			},
		},
		"required": []string{"task_result", "file_count"},
	}

	// Test prompt
	testPrompt := `Please analyze the current project and provide a summary.
Return your response in the structured JSON format as specified.`

	fmt.Println("\nTest Prompt:")
	fmt.Println(testPrompt)
	fmt.Println("\nJSON Schema:")
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Println(string(schemaJSON))
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Convert map[string]interface{} to map[string]string
	envMap := make(map[string]string)
	for k, v := range rawEnv.(map[string]interface{}) {
		envMap[k] = fmt.Sprintf("%v", v)
	}

	// Create options with structured output format
	opts := types.NewClaudeAgentOptions().
		WithVerbose(true).
		WithEnv(envMap).
		WithSettingSources(types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal).
		WithOutputFormat(schema) // Enable structured output with JSON schema

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("Sending query to Claude with structured output...")
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
	var structuredOutput interface{}

	for msg := range messages {
		messageCount++
		content, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("Error marshaling message: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[%d] %s\n", messageCount, string(content))
		fmt.Println(strings.Repeat("-", 60))

		// Capture structured output from result message
		if resultMsg, ok := msg.(*types.ResultMessage); ok {
			if resultMsg.StructuredOutput != nil {
				structuredOutput = resultMsg.StructuredOutput
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Structured Output Result:")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	if structuredOutput != nil {
		outputJSON, _ := json.MarshalIndent(structuredOutput, "", "  ")
		fmt.Println(string(outputJSON))

		// Validate the output matches schema
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("Validation:")
		fmt.Println(strings.Repeat("=", 60))
		validateStructuredOutput(structuredOutput)
	} else {
		fmt.Println("No structured output found in result message!")
	}

	fmt.Printf("\n=== Test Complete. Total messages: %d ===\n", messageCount)
}

func validateStructuredOutput(output interface{}) {
	outputMap, ok := output.(map[string]interface{})
	if !ok {
		fmt.Println("❌ Output is not a JSON object")
		return
	}

	requiredFields := []string{"task_result", "file_count"}
	allValid := true

	for _, field := range requiredFields {
		if _, exists := outputMap[field]; !exists {
			fmt.Printf("❌ Missing required field: %s\n", field)
			allValid = false
		} else {
			fmt.Printf("✓ Found field: %s\n", field)
		}
	}

	// Type validation
	if v, ok := outputMap["file_count"].(float64); ok {
		fmt.Printf("✓ file_count is a number: %.0f\n", v)
	} else {
		fmt.Println("❌ file_count is not a number")
		allValid = false
	}

	if allValid {
		fmt.Println("\n✅ All validations passed!")
	} else {
		fmt.Println("\n❌ Some validations failed!")
	}
}
