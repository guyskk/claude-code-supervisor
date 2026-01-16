// test-sdk-auto-compact tests Claude Agent SDK's auto-compact functionality
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guyskk/ccc/internal/config"
	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// CompactEvent records an auto-compact event
type CompactEvent struct {
	Timestamp             string  `json:"timestamp"`
	TriggerType           string  `json:"trigger_type"`
	QueryIndex            int     `json:"query_index"`
	EstimatedTokensBefore int     `json:"estimated_tokens_before"`
	CustomInstructions    *string `json:"custom_instructions,omitempty"`
	SessionID             string  `json:"session_id"`
}

// QueryRecord records a single query
type QueryRecord struct {
	Index          int            `json:"index"`
	Timestamp      string         `json:"timestamp"`
	QueryText      string         `json:"query_text"`
	QueryTokens    int            `json:"query_tokens"`
	ResponseText   string         `json:"response_text"`
	ResponseTokens int            `json:"response_tokens"`
	TotalTokens    int            `json:"total_tokens"`
	IsCompacted    bool           `json:"is_compacted"`
	CompactEvents  []CompactEvent `json:"compact_events,omitempty"`
}

// TestResult contains the complete test results
type TestResult struct {
	StartTime            string         `json:"start_time"`
	EndTime              string         `json:"end_time"`
	DurationSeconds      float64        `json:"duration_seconds"`
	TotalQueries         int            `json:"total_queries"`
	TotalTokensSent      int            `json:"total_tokens_sent"`
	TotalTokensReceived  int            `json:"total_tokens_received"`
	TotalTokens          int            `json:"total_tokens"`
	TargetTokens         int            `json:"target_tokens"`
	TargetReached        bool           `json:"target_reached"`
	AutoCompactTriggered bool           `json:"auto_compact_triggered"`
	CompactEvents        []CompactEvent `json:"compact_events"`
	Queries              []QueryRecord  `json:"queries"`
	Model                string         `json:"model"`
	SessionID            string         `json:"session_id"`
}

// TestConfig holds test configuration
type TestConfig struct {
	TargetTokens int
	QuerySize    int
	MaxQueries   int
	OutputDir    string
	WorkDir      string
	SessionID    string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Use current provider
	providerName := cfg.CurrentProvider
	if providerName == "" {
		// Default to first provider if none set
		for name := range cfg.Providers {
			providerName = name
			break
		}
	}

	provider, exists := cfg.Providers[providerName]
	if !exists {
		fmt.Printf("Provider %s not found in config\n", providerName)
		os.Exit(1)
	}

	rawEnv := provider["env"]
	if rawEnv == nil {
		fmt.Printf("No env configuration found for provider %s\n", providerName)
		os.Exit(1)
	}

	envMap := make(map[string]string)
	for k, v := range rawEnv.(map[string]interface{}) {
		envMap[k] = fmt.Sprintf("%v", v)
	}

	model := config.GetModel(provider)

	testConfig := TestConfig{
		TargetTokens: 200000, // Target 200k tokens to trigger auto-compact
		QuerySize:    15000,  // 15k tokens per query
		MaxQueries:   1000,
		OutputDir:    "./tmp/test-sdk-auto-compact",
		WorkDir:      "./tmp/agent-work-compact-test",
		SessionID:    "test-auto-compact",
	}

	// Create output directories
	os.MkdirAll(testConfig.OutputDir, 0755)
	os.MkdirAll(testConfig.WorkDir, 0755)

	// Run the test
	result, err := runTest(context.Background(), envMap, model, testConfig)
	if err != nil {
		fmt.Printf("Test failed: %v\n", err)
		os.Exit(1)
	}

	// Save results
	saveResults(result, testConfig.OutputDir)

	// Print final summary
	printFinalSummary(result)

	// Exit with appropriate code
	if result.AutoCompactTriggered {
		fmt.Println("\n✅ Auto-compact functionality is working!")
		os.Exit(0)
	} else {
		fmt.Println("\n❌ Auto-compact was NOT triggered!")
		os.Exit(1)
	}
}

// runTest executes the auto-compact verification test
func runTest(ctx context.Context, envMap map[string]string, model string, cfg TestConfig) (*TestResult, error) {
	fmt.Println("=== Claude Agent SDK Auto-Compact Verification Test ===")
	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Target Tokens: %d\n", cfg.TargetTokens)
	fmt.Printf("Query Size: %d tokens per query\n", cfg.QuerySize)
	fmt.Printf("Max Queries: %d\n", cfg.MaxQueries)
	fmt.Printf("Output Directory: %s\n", cfg.OutputDir)
	fmt.Println(strings.Repeat("=", 60))

	startTime := time.Now()

	result := &TestResult{
		StartTime:     startTime.Format(time.RFC3339),
		Model:         model,
		SessionID:     cfg.SessionID,
		Queries:       make([]QueryRecord, 0),
		CompactEvents: make([]CompactEvent, 0),
	}

	// Track state
	var totalTokensSent, totalTokensReceived int
	var actualSessionID *string
	paragraphIndex := 0
	compactEvents := make([]CompactEvent, 0)
	queries := make([]QueryRecord, 0)

	// Pre-compact hook callback
	preCompactHook := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("🔄 AUTO COMPACT TRIGGERED!\n")

		hookInput, ok := input.(*types.PreCompactHookInput)
		if !ok {
			return &types.SyncHookJSONOutput{}, nil
		}

		estimatedTokens := totalTokensSent + totalTokensReceived

		event := CompactEvent{
			Timestamp:             time.Now().Format(time.RFC3339),
			TriggerType:           hookInput.Trigger,
			QueryIndex:            len(queries) + 1,
			EstimatedTokensBefore: estimatedTokens,
			CustomInstructions:    hookInput.CustomInstructions,
			SessionID:             hookInput.SessionID,
		}
		compactEvents = append(compactEvents, event)

		fmt.Printf("  Trigger Type: %s\n", event.TriggerType)
		fmt.Printf("  Query Index: %d\n", event.QueryIndex)
		fmt.Printf("  Estimated Tokens Before: %d\n", event.EstimatedTokensBefore)
		fmt.Printf("  Timestamp: %s\n", event.Timestamp)
		fmt.Println(strings.Repeat("=", 60))

		return &types.SyncHookJSONOutput{Continue: boolPtr(true)}, nil
	}

	// System prompt - simple responses to minimize output tokens
	systemPrompt := `You are a code analyzer. When you receive code or text content:
1. Respond with ONLY the word ACKNOWLEDGED followed by a brief 1-2 sentence summary.
2. Do NOT explain the content in detail.
3. Keep your response under 50 words maximum.

Example response: ACKNOWLEDGED. Received technical documentation about software engineering.`

	for i := 0; i < cfg.MaxQueries; i++ {
		queryIndex := i + 1
		estimatedContext := totalTokensSent + totalTokensReceived

		// Generate query content
		queryText, queryTokens := generateQueryContent(cfg.QuerySize, paragraphIndex)
		paragraphIndex += (queryTokens / cfg.QuerySize) + 1

		fmt.Printf("\n%s\nQuery #%d\n%s\n", strings.Repeat("─", 60), queryIndex, strings.Repeat("─", 60))
		fmt.Printf("Query Tokens: %d\n", queryTokens)
		fmt.Printf("Estimated Context Before: %d tokens\n", estimatedContext)
		fmt.Printf("Progress: %d / %d (%.1f%%)\n", estimatedContext, cfg.TargetTokens, float64(estimatedContext)/float64(cfg.TargetTokens)*100)

		// Create options
		opts := types.NewClaudeAgentOptions().
			WithVerbose(true).
			WithEnv(envMap).
			WithSystemPrompt(systemPrompt).
			WithSettingSources(types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal).
			WithCWD(cfg.WorkDir).
			WithHook(types.HookEventPreCompact, types.HookMatcher{
				Matcher: nil,
				Hooks:   []types.HookCallbackFunc{preCompactHook},
			})

		// Add resume if we have a session ID
		if actualSessionID != nil {
			opts = opts.WithResume(*actualSessionID)
		}

		// Execute query
		messages, err := claude.Query(ctx, queryText, opts)
		if err != nil {
			return nil, fmt.Errorf("query failed: %w", err)
		}

		// Process response
		var responseText strings.Builder
		var responseTokens int

		for msg := range messages {
			switch m := msg.(type) {
			case *types.AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(*types.TextBlock); ok {
						responseText.WriteString(tb.Text)
					}
				}
			case *types.ResultMessage:
				// Capture session ID on first query
				if actualSessionID == nil && m.SessionID != "" {
					actualSessionID = &m.SessionID
					fmt.Printf("Got actual session_id from SDK: %s\n", m.SessionID)
				}

				responseTokens = estimateTokens(responseText.String())
				totalTokensSent += queryTokens
				totalTokensReceived += responseTokens

				// Check if this query triggered compact
				isCompacted := false
				queryCompactEvents := make([]CompactEvent, 0)
				for _, event := range compactEvents {
					if event.QueryIndex == queryIndex {
						isCompacted = true
						queryCompactEvents = append(queryCompactEvents, event)
					}
				}

				record := QueryRecord{
					Index:          queryIndex,
					Timestamp:      time.Now().Format(time.RFC3339),
					QueryText:      truncateString(queryText, 500),
					QueryTokens:    queryTokens,
					ResponseText:   responseText.String(),
					ResponseTokens: responseTokens,
					TotalTokens:    queryTokens + responseTokens,
					IsCompacted:    isCompacted,
					CompactEvents:  queryCompactEvents,
				}
				queries = append(queries, record)

				fmt.Printf("Response Tokens: %d\n", responseTokens)
				fmt.Printf("Total Tokens (cumulative): %d\n", totalTokensSent+totalTokensReceived)
				break
			case *types.SystemMessage:
				if m.IsError() {
					// Extract error message from data if available
					if errMsg, ok := m.Data["error"].(string); ok {
						return nil, fmt.Errorf("system error: %s", errMsg)
					}
					return nil, fmt.Errorf("system error occurred")
				}
			}
		}

		// Check if target reached
		if estimatedContext >= cfg.TargetTokens {
			fmt.Printf("\n%s\n🎯 Target tokens reached! %d >= %d\n%s\n",
				strings.Repeat("=", 60), estimatedContext, cfg.TargetTokens, strings.Repeat("=", 60))
			break
		}
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime).Seconds()

	result.EndTime = endTime.Format(time.RFC3339)
	result.DurationSeconds = duration
	result.TotalQueries = len(queries)
	result.TotalTokensSent = totalTokensSent
	result.TotalTokensReceived = totalTokensReceived
	result.TotalTokens = totalTokensSent + totalTokensReceived
	result.TargetTokens = cfg.TargetTokens
	result.TargetReached = result.TotalTokens >= cfg.TargetTokens
	result.AutoCompactTriggered = len(compactEvents) > 0
	result.CompactEvents = compactEvents
	result.Queries = queries

	return result, nil
}

// saveResults saves test results to files
func saveResults(result *TestResult, outputDir string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Saving test results...")

	// Save JSON result
	jsonFile := filepath.Join(outputDir, "test_result.json")
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
		fmt.Printf("Failed to save JSON result: %v\n", err)
	} else {
		fmt.Printf("JSON result saved to: %s\n", jsonFile)
	}

	// Save text report
	reportFile := filepath.Join(outputDir, "test_report.txt")
	saveTextReport(result, reportFile)
	fmt.Printf("Text report saved to: %s\n", reportFile)

	// Save queries
	queriesFile := filepath.Join(outputDir, "queries.json")
	queriesData, _ := json.MarshalIndent(result.Queries, "", "  ")
	if err := os.WriteFile(queriesFile, queriesData, 0644); err != nil {
		fmt.Printf("Failed to save queries: %v\n", err)
	} else {
		fmt.Printf("Queries saved to: %s\n", queriesFile)
	}

	// Save compact events
	compactFile := filepath.Join(outputDir, "compact_events.json")
	eventsData, _ := json.MarshalIndent(result.CompactEvents, "", "  ")
	if err := os.WriteFile(compactFile, eventsData, 0644); err != nil {
		fmt.Printf("Failed to save compact events: %v\n", err)
	} else {
		fmt.Printf("Compact events saved to: %s\n", compactFile)
	}
}

// saveTextReport saves a human-readable text report
func saveTextReport(result *TestResult, filepath string) {
	var sb strings.Builder

	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("Claude Agent SDK Auto-Compact Verification Report\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	// Summary
	sb.WriteString("## SUMMARY\n\n")
	sb.WriteString(fmt.Sprintf("Test Duration: %.2f seconds\n", result.DurationSeconds))
	sb.WriteString(fmt.Sprintf("Model: %s\n", result.Model))
	sb.WriteString(fmt.Sprintf("Session ID: %s\n", result.SessionID))
	sb.WriteString(fmt.Sprintf("Start Time: %s\n", result.StartTime))
	sb.WriteString(fmt.Sprintf("End Time: %s\n\n", result.EndTime))

	// Token Statistics
	sb.WriteString("## TOKEN STATISTICS\n\n")
	sb.WriteString(fmt.Sprintf("Target Tokens: %d\n", result.TargetTokens))
	sb.WriteString(fmt.Sprintf("Total Tokens Sent: %d\n", result.TotalTokensSent))
	sb.WriteString(fmt.Sprintf("Total Tokens Received: %d\n", result.TotalTokensReceived))
	sb.WriteString(fmt.Sprintf("Total Context Tokens: %d\n", result.TotalTokens))
	sb.WriteString(fmt.Sprintf("Target Reached: %s\n\n", boolToStr(result.TargetReached)))

	// Auto-Compact Result
	sb.WriteString("## AUTO-COMPACT RESULT\n\n")
	sb.WriteString(fmt.Sprintf("Auto-Compact Triggered: %s\n", passFail(result.AutoCompactTriggered)))
	sb.WriteString(fmt.Sprintf("Number of Compact Events: %d\n\n", len(result.CompactEvents)))

	// Compact Events
	if len(result.CompactEvents) > 0 {
		sb.WriteString("## COMPACT EVENTS\n\n")
		for i, event := range result.CompactEvents {
			sb.WriteString(fmt.Sprintf("### Event %d\n", i+1))
			sb.WriteString(fmt.Sprintf("  Timestamp: %s\n", event.Timestamp))
			sb.WriteString(fmt.Sprintf("  Trigger Type: %s\n", event.TriggerType))
			sb.WriteString(fmt.Sprintf("  Query Index: %d\n", event.QueryIndex))
			sb.WriteString(fmt.Sprintf("  Estimated Tokens Before: %d\n", event.EstimatedTokensBefore))
			if event.CustomInstructions != nil && *event.CustomInstructions != "" {
				sb.WriteString(fmt.Sprintf("  Custom Instructions: %s\n", *event.CustomInstructions))
			}
			sb.WriteString(fmt.Sprintf("  Session ID: %s\n\n", event.SessionID))
		}
	}

	// Query Statistics
	sb.WriteString("## QUERY STATISTICS\n\n")
	sb.WriteString(fmt.Sprintf("Total Queries: %d\n", result.TotalQueries))
	if len(result.Queries) > 0 {
		avgQueryTokens := 0
		avgResponseTokens := 0
		for _, q := range result.Queries {
			avgQueryTokens += q.QueryTokens
			avgResponseTokens += q.ResponseTokens
		}
		avgQueryTokens /= len(result.Queries)
		avgResponseTokens /= len(result.Queries)
		sb.WriteString(fmt.Sprintf("Average Query Tokens: %d\n", avgQueryTokens))
		sb.WriteString(fmt.Sprintf("Average Response Tokens: %d\n\n", avgResponseTokens))
	}

	// Query Details (sample)
	sb.WriteString("## QUERY DETAILS (Sample)\n\n")
	sampleQueries := result.Queries
	if len(result.Queries) > 15 {
		sampleQueries = append(result.Queries[:10], result.Queries[len(result.Queries)-5:]...)
	}
	for _, q := range sampleQueries {
		sb.WriteString(fmt.Sprintf("### Query #%d\n", q.Index))
		sb.WriteString(fmt.Sprintf("  Timestamp: %s\n", q.Timestamp))
		sb.WriteString(fmt.Sprintf("  Query Tokens: %d\n", q.QueryTokens))
		sb.WriteString(fmt.Sprintf("  Response Tokens: %d\n", q.ResponseTokens))
		sb.WriteString(fmt.Sprintf("  Total Tokens: %d\n", q.TotalTokens))
		sb.WriteString(fmt.Sprintf("  Is Compacted: %s\n", boolToStr(q.IsCompacted)))
		sb.WriteString(fmt.Sprintf("  Query Text: %s...\n", truncateString(q.QueryText, 200)))
		sb.WriteString(fmt.Sprintf("  Response: %s...\n\n", truncateString(q.ResponseText, 200)))
	}

	sb.WriteString(strings.Repeat("=", 80) + "\n")

	os.WriteFile(filepath, []byte(sb.String()), 0644)
}

// printFinalSummary prints the final result summary
func printFinalSummary(result *TestResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("FINAL RESULT")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Duration: %.2f seconds\n", result.DurationSeconds)
	fmt.Printf("Total Queries: %d\n", result.TotalQueries)
	fmt.Printf("Total Context Tokens: %d\n", result.TotalTokens)
	fmt.Printf("Target Reached: %s\n", passFail(result.TargetReached))
	fmt.Printf("Auto-Compact Triggered: %s\n", passFail(result.AutoCompactTriggered))
	fmt.Printf("Number of Compact Events: %d\n", len(result.CompactEvents))
	fmt.Println(strings.Repeat("=", 80))

	if len(result.CompactEvents) > 0 {
		fmt.Println("\nCompact Event Details:")
		for _, event := range result.CompactEvents {
			fmt.Printf("  - Query #%d: trigger=%s, tokens≈%d\n",
				event.QueryIndex, event.TriggerType, event.EstimatedTokensBefore)
		}
	}
}

// Helper functions

func estimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 4 characters for English text
	return len(text) / 4
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func boolToStr(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func passFail(b bool) string {
	if b {
		return "YES ✅"
	}
	return "NO ❌"
}

func boolPtr(b bool) *bool {
	return &b
}
