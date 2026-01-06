// Package supervisor implements stream-json parsing for supervisor mode.
package supervisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StreamMessage represents a message from claude stream-json output.
type StreamMessage struct {
	Type             string            `json:"type"`
	SessionID        string            `json:"session_id,omitempty"`
	Content          string            `json:"content,omitempty"`
	Result           string            `json:"result,omitempty"`            // Text result for type="result" messages
	StructuredOutput *SupervisorResult `json:"structured_output,omitempty"` // Structured output for type="result" messages
}

// ParseStreamJSONLine parses a single line of stream-json output.
func ParseStreamJSONLine(line string) (*StreamMessage, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	var msg StreamMessage
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil, fmt.Errorf("failed to parse stream-json: %w", err)
	}

	return &msg, nil
}

// SupervisorResult represents the structured output from Supervisor.
type SupervisorResult struct {
	Completed bool   `json:"completed"`
	Feedback  string `json:"feedback"`
}

// ParseSupervisorResult parses the result field from a stream message.
func ParseSupervisorResult(result string) (*SupervisorResult, error) {
	var sr SupervisorResult
	if err := json.Unmarshal([]byte(result), &sr); err != nil {
		return nil, fmt.Errorf("failed to parse supervisor result: %w", err)
	}
	return &sr, nil
}
