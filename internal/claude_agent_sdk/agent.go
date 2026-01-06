// Package claude_agent_sdk provides a Go SDK for interacting with Claude Code CLI.
package claude_agent_sdk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/guyskk/ccc/internal/logger"
)

// Config holds the configuration for the Agent.
type Config struct {
	// ClaudePath is the path to the claude executable. Defaults to "claude".
	ClaudePath string

	// Timeout is the default timeout for claude operations.
	Timeout time.Duration

	// Logger is the logger to use. If nil, uses a no-op logger.
	Logger logger.Logger
}

// Agent represents a Claude Code agent that can execute prompts.
type Agent struct {
	config *Config
	logger logger.Logger
	mu     sync.RWMutex
}

// NewAgent creates a new Agent with the given configuration.
func NewAgent(config *Config) (*Agent, error) {
	if config == nil {
		config = &Config{}
	}

	// Set defaults
	if config.ClaudePath == "" {
		config.ClaudePath = "claude"
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}
	if config.Logger == nil {
		config.Logger = logger.NewNopLogger()
	}

	// Verify claude is available
	if _, err := exec.LookPath(config.ClaudePath); err != nil {
		return nil, fmt.Errorf("claude executable not found: %w", err)
	}

	return &Agent{
		config: config,
		logger: config.Logger,
	}, nil
}

// RunOptions specifies how to run the claude command.
type RunOptions struct {
	// SessionID is the session to resume (if any).
	SessionID string

	// ForkSession creates a child session instead of using --print.
	ForkSession bool

	// Prompt is the user prompt to send.
	Prompt string

	// OutputFormat specifies the output format ("stream-json", "json", "text").
	OutputFormat string

	// JSONSchema specifies the JSON schema for structured output.
	JSONSchema string

	// Env are additional environment variables to set.
	Env []string

	// Timeout overrides the default timeout. If zero, uses default.
	Timeout time.Duration
}

// RunResult contains the result of a Run call.
type RunResult struct {
	// Output is the combined text output.
	Output string

	// StructuredOutput is the parsed structured output from stream-json mode.
	StructuredOutput map[string]interface{}

	// Duration is how long the operation took.
	Duration time.Duration

	// Success indicates if the operation succeeded.
	Success bool

	// Error contains any error that occurred.
	Error error
}

// Run executes a claude command with the given options and returns the result.
func (a *Agent) Run(ctx context.Context, opts RunOptions) (*RunResult, error) {
	startTime := time.Now()

	// Set defaults
	if opts.OutputFormat == "" {
		opts.OutputFormat = "text"
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = a.config.Timeout
	}

	// Build context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create process
	process, err := a.newProcess(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	// Start process
	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Collect output
	var output strings.Builder
	var structuredOutput map[string]interface{}

	// Read stdout line by line
	for line := range process.StdoutLine() {
		output.WriteString(line)
		output.WriteString("\n")

		// Try to parse as stream-json
		if opts.OutputFormat == "stream-json" {
			msg, parseErr := ParseStreamJSONLine(line)
			if parseErr == nil && msg != nil {
				if msg.Type == "result" && msg.StructuredOutput != nil {
					structuredOutput = msg.StructuredOutput
				}
			}
		}
	}

	// Read stderr
	var stderr strings.Builder
	for line := range process.StderrLine() {
		stderr.WriteString(line)
		stderr.WriteString("\n")
	}

	// Wait for process to complete
	waitErr := process.Wait()

	result := &RunResult{
		Output:           output.String(),
		StructuredOutput: structuredOutput,
		Duration:         time.Since(startTime),
		Success:          waitErr == nil,
		Error:            waitErr,
	}

	// Log stderr if present
	if stderr.Len() > 0 {
		a.logger.Debug("claude stderr", logger.StringField("stderr", stderr.String()))
	}

	// Handle timeout
	ctxErr := ctx.Err()
	if ctxErr == context.DeadlineExceeded {
		result.Success = false
		result.Error = fmt.Errorf("claude operation timed out after %v", timeout)
		a.logger.Error("claude timeout", logger.StringField("duration", result.Duration.String()))
		return result, nil
	}

	return result, nil
}

// newProcess creates a new Process for the given options.
func (a *Agent) newProcess(ctx context.Context, opts RunOptions) (*Process, error) {
	args := a.buildArgs(opts)

	cmd := exec.CommandContext(ctx, a.config.ClaudePath, args...)
	cmd.Env = append(os.Environ(), opts.Env...)

	return NewProcess(cmd, a.logger), nil
}

// buildArgs constructs the command line arguments for claude.
func (a *Agent) buildArgs(opts RunOptions) []string {
	args := []string{"-p"}

	if opts.ForkSession && opts.SessionID != "" {
		args = append(args,
			"--fork-session",
			"--resume", opts.SessionID,
		)
	} else if opts.SessionID != "" {
		args = append(args, "--print", "--resume", opts.SessionID)
	}

	args = append(args, "--verbose")

	if opts.OutputFormat != "" {
		args = append(args, "--output-format", opts.OutputFormat)
	}

	if opts.JSONSchema != "" {
		args = append(args, "--json-schema", opts.JSONSchema)
	}

	if opts.Prompt != "" {
		args = append(args, opts.Prompt)
	}

	return args
}

// EventType represents the type of a StreamEvent.
type EventType int

const (
	// EventStart is emitted when the process starts.
	EventStart EventType = iota
	// EventMessage is emitted for each stream-json text message.
	EventMessage
	// EventToolUse is emitted when a tool is called.
	EventToolUse
	// EventToolResult is emitted when a tool result is received.
	EventToolResult
	// EventEnd is emitted when the process ends.
	EventEnd
)

// StreamEvent represents an event from RunStream.
type StreamEvent struct {
	Type    EventType
	Content string
	Error   error
	Meta    map[string]interface{}
}

// RunStream executes a claude command and returns a channel of events.
func (a *Agent) RunStream(ctx context.Context, opts RunOptions) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		startTime := time.Now()
		ch <- StreamEvent{Type: EventStart, Meta: map[string]interface{}{"timestamp": startTime}}

		// Set defaults
		if opts.OutputFormat == "" {
			opts.OutputFormat = "stream-json"
		}
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = a.config.Timeout
		}

		// Build context with timeout
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Create and start process
		process, err := a.newProcess(ctx, opts)
		if err != nil {
			ch <- StreamEvent{Type: EventEnd, Error: err}
			return
		}

		if err := process.Start(); err != nil {
			ch <- StreamEvent{Type: EventEnd, Error: err}
			return
		}

		// Read stdout line by line
		for line := range process.StdoutLine() {
			// Try to parse as stream-json
			msg, parseErr := ParseStreamJSONLine(line)
			if parseErr == nil && msg != nil {
				switch msg.Type {
				case "text":
					ch <- StreamEvent{Type: EventMessage, Content: msg.Content}
				case "tool_use":
					ch <- StreamEvent{Type: EventToolUse, Content: msg.Content, Meta: msg.Meta}
				case "tool_result":
					ch <- StreamEvent{Type: EventToolResult, Content: msg.Content, Meta: msg.Meta}
				case "result":
					// Stream result completed
					ch <- StreamEvent{Type: EventEnd, Meta: map[string]interface{}{
						"structured_output": msg.StructuredOutput,
						"duration":          time.Since(startTime),
					}}
					return
				}
			}
		}

		// Wait for process
		waitErr := process.Wait()

		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			ch <- StreamEvent{Type: EventEnd, Error: fmt.Errorf("timeout after %v", timeout)}
			return
		}

		ch <- StreamEvent{Type: EventEnd, Error: waitErr, Meta: map[string]interface{}{"duration": time.Since(startTime)}}
	}()

	return ch
}
