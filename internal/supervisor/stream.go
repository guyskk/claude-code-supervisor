// Package supervisor implements Agent-Supervisor automatic loop.
package supervisor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StreamMessage represents a message from claude stream-json output.
type StreamMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
	// Add more fields as needed based on actual stream-json structure
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

// ExtractSessionID extracts the session ID from a stream message.
func ExtractSessionID(msg *StreamMessage) string {
	if msg == nil {
		return ""
	}
	return msg.SessionID
}

// DetectAgentWaiting detects if Agent is waiting for user input.
// Based on stream message type and content.
func DetectAgentWaiting(msg *StreamMessage) bool {
	if msg == nil {
		return false
	}

	// Agent is waiting when we see a "result" type message
	// This indicates Agent has completed its response and is waiting
	return msg.Type == "result"
}

// IsTaskCompleted checks if the output contains the task completion marker.
func IsTaskCompleted(output, marker string) bool {
	return strings.Contains(output, marker)
}
