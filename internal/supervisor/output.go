// Package supervisor provides supervisor output functionality.
package supervisor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

// HookOutput represents the output to stdout.
// Reason is always set (provides context for the decision).
// Decision is "block" when not allowing stop, omitted when allowing stop.
type HookOutput struct {
	Decision *string `json:"decision,omitempty"` // "block" or omitted (allows stop)
	Reason   string  `json:"reason"`             // Always set
}

// OutputDecision outputs the supervisor's decision.
//
// Parameters:
//   - log: The logger to use
//   - allowStop: true to allow the agent to stop, false to block and require more work
//   - feedback: Feedback message explaining the decision (can be empty)
//
// The function:
// 1. Outputs JSON to stdout for Claude Code to parse
// 2. Logs the decision
func OutputDecision(log *slog.Logger, allowStop bool, feedback string) error {
	// Trim feedback
	feedback = strings.TrimSpace(feedback)

	// Build output
	output := HookOutput{Reason: feedback}
	if !allowStop {
		block := "block"
		output.Decision = &block
		if feedback == "" {
			output.Reason = "Please continue completing the task"
		}
	}

	outputJSON, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal hook output: %w", err)
	}

	// Log the decision
	if allowStop {
		log.Info("supervisor output: allow stop", "feedback", feedback)
	} else {
		log.Info("supervisor output: not allow stop", "feedback", output.Reason)
	}

	// Output JSON to stdout (for Claude Code to parse)
	fmt.Println(string(outputJSON))

	return nil
}
