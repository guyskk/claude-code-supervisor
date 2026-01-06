// Package claude_agent_sdk provides stream-json parsing for Claude Code CLI.
package claude_agent_sdk

import (
	"encoding/json"
	"fmt"
)

// StreamMessage represents a message in the stream-json format.
type StreamMessage struct {
	Type             string                 `json:"type"`
	Content          string                 `json:"content,omitempty"`
	StructuredOutput map[string]interface{} `json:"structured_output,omitempty"`
	Meta             map[string]interface{} `json:"meta,omitempty"`
}

// ParseStreamJSONLine parses a single line of stream-json output.
// Returns nil if the line is not valid JSON or is not a stream message.
func ParseStreamJSONLine(line string) (*StreamMessage, error) {
	if line == "" {
		return nil, nil
	}

	var msg StreamMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		// Not a valid JSON line, return nil - this is expected for non-JSON output
		return nil, nil
	}

	return &msg, nil
}

// ParseSupervisorResult extracts the supervisor result from a stream message.
func ParseSupervisorResult(structuredOutput map[string]interface{}) (*SupervisorResult, error) {
	if structuredOutput == nil {
		return nil, fmt.Errorf("no structured output")
	}

	result := &SupervisorResult{}

	// Extract completed field
	if completed, ok := structuredOutput["completed"].(bool); ok {
		result.Completed = completed
	} else {
		return nil, fmt.Errorf("missing or invalid 'completed' field")
	}

	// Extract feedback field
	if feedback, ok := structuredOutput["feedback"].(string); ok {
		result.Feedback = feedback
	}

	return result, nil
}

// SupervisorResult represents the result from a supervisor review.
type SupervisorResult struct {
	Completed bool
	Feedback  string
}
