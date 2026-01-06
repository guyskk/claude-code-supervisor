// Package claude provides a Go SDK for interacting with the Claude Code CLI.
// It wraps the claude command-line tool with a clean Go API for managing sessions,
// executing prompts, and handling output streams.
package claude

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/guyskk/ccc/internal/supervisor"
)

// ErrorCode represents different types of errors that can occur.
type ErrorCode string

const (
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeStartFailed    ErrorCode = "START_FAILED"
	ErrCodeExecutionError ErrorCode = "EXECUTION_ERROR"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeCancelled      ErrorCode = "CANCELLED"
	ErrCodeParseError     ErrorCode = "PARSE_ERROR"
)

// Error represents an error from the Claude SDK.
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError creates a new Error with the given code and message.
func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// OutputFormat specifies how Claude should format output.
type OutputFormat string

const (
	// OutputFormatText is plain text output.
	OutputFormatText OutputFormat = "text"
	// OutputFormatStreamJSON is streaming JSON output.
	OutputFormatStreamJSON OutputFormat = "stream-json"
)

// SessionConfig holds configuration for a Claude session.
type SessionConfig struct {
	// Prompt is the user prompt to send to Claude.
	Prompt string
	// ResumeID is the session ID to resume from.
	ResumeID string
	// ForkSession creates a child session instead of a new one.
	ForkSession bool
	// OutputFormat specifies the output format.
	OutputFormat OutputFormat
	// JSONSchema specifies the JSON schema for structured output.
	JSONSchema string
	// Verbose enables verbose output.
	Verbose bool
	// ExtraArgs are additional arguments to pass to claude.
	ExtraArgs []string
	// Env are environment variables to set for the claude process.
	Env []string
	// Timeout is the maximum time to wait for completion (0 = no timeout).
	Timeout time.Duration
}

// StreamMessage represents a message from Claude's stream-json output.
type StreamMessage = supervisor.StreamMessage

// Client represents a Claude CLI client.
type Client struct {
	// ClaudePath is the path to the claude executable.
	ClaudePath string
}

// NewClient creates a new Claude client.
func NewClient() (*Client, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return nil, NewError(ErrCodeNotFound, "claude executable not found in PATH", err)
	}
	return &Client{
		ClaudePath: claudePath,
	}, nil
}

// NewClientWithPath creates a new Claude client with a specific path.
func NewClientWithPath(path string) *Client {
	return &Client{
		ClaudePath: path,
	}
}

// Execute runs a Claude session with the given configuration.
// It returns the combined stdout and stderr output.
func (c *Client) Execute(ctx context.Context, config *SessionConfig) (string, error) {
	var output strings.Builder
	var stderrOutput strings.Builder

	handler := &StreamHandler{
		OnText: func(msg *StreamMessage) error {
			if msg.Content != "" {
				output.WriteString(msg.Content)
				output.WriteString("\n")
			}
			return nil
		},
		OnResult: func(msg *StreamMessage) error {
			if msg.Result != "" {
				output.WriteString(msg.Result)
			}
			return nil
		},
		OnStderr: func(line string) error {
			stderrOutput.WriteString(line)
			return nil
		},
	}

	err := c.ExecuteStream(ctx, config, handler)
	if err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

// StreamHandler handles streaming output from Claude.
type StreamHandler struct {
	// OnText is called for each text message.
	OnText func(msg *StreamMessage) error
	// OnResult is called for each result message.
	OnResult func(msg *StreamMessage) error
	// OnStructuredOutput is called when structured output is received.
	OnStructuredOutput func(output interface{}) error
	// OnStderr is called for each line of stderr.
	OnStderr func(line string) error
	// OnRawLine is called for each raw line of stdout (before parsing).
	OnRawLine func(line string) error
}

// ExecuteStream runs a Claude session with streaming output.
func (c *Client) ExecuteStream(ctx context.Context, config *SessionConfig, handler *StreamHandler) error {
	args := c.buildArgs(config)

	cmd := exec.CommandContext(ctx, c.ClaudePath, args...)
	cmd.Env = append(os.Environ(), config.Env...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return NewError(ErrCodeStartFailed, "failed to create stdout pipe", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return NewError(ErrCodeStartFailed, "failed to create stderr pipe", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return NewError(ErrCodeStartFailed, "failed to start claude command", err)
	}

	// Read stdout and stderr concurrently
	var wg sync.WaitGroup
	var stdoutErr, stderrErr error

	// Goroutine to read stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		stdoutErr = c.readStdout(stdout, config.OutputFormat, handler)
	}()

	// Goroutine to read stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		stderrErr = c.readStderr(stderr, handler)
	}()

	// Wait for both readers to finish
	wg.Wait()

	// Wait for command to finish
	cmdErr := cmd.Wait()

	// Check for context cancellation
	if ctx.Err() != nil {
		return NewError(ErrCodeCancelled, "claude execution cancelled", ctx.Err())
	}

	// Handle errors
	if cmdErr != nil {
		// Check if it's a signal (usually from Kill)
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				// Killed by signal, usually expected
				if stdoutErr == nil && stderrErr == nil {
					return nil
				}
			}
		}
		return NewError(ErrCodeExecutionError, "claude command failed", cmdErr)
	}

	if stdoutErr != nil {
		return stdoutErr
	}
	if stderrErr != nil {
		return stderrErr
	}

	return nil
}

// buildArgs constructs the command line arguments for claude.
func (c *Client) buildArgs(config *SessionConfig) []string {
	args := []string{}

	// Add interactive flag for prompts
	if config.Prompt != "" {
		args = append(args, "-p")
	}

	// Add session flags
	if config.ForkSession {
		args = append(args, "--fork-session")
	}
	if config.ResumeID != "" {
		args = append(args, "--resume", config.ResumeID)
	}

	// Add output format
	if config.OutputFormat != "" {
		args = append(args, "--output-format", string(config.OutputFormat))
	}

	// Add verbose flag
	if config.Verbose {
		args = append(args, "--verbose")
	}

	// Add JSON schema for structured output
	if config.JSONSchema != "" {
		args = append(args, "--json-schema", config.JSONSchema)
	}

	// Add extra args
	args = append(args, config.ExtraArgs...)

	// Add prompt as last argument
	if config.Prompt != "" {
		args = append(args, config.Prompt)
	}

	return args
}

// readStdout reads and processes stdout line by line.
func (c *Client) readStdout(stdout io.Reader, format OutputFormat, handler *StreamHandler) error {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Call raw line handler
		if handler.OnRawLine != nil {
			if err := handler.OnRawLine(line); err != nil {
				return err
			}
		}

		// Parse based on format
		if format == OutputFormatStreamJSON {
			msg, err := supervisor.ParseStreamJSONLine(line)
			if err != nil {
				// Not valid JSON, might be non-JSON output
				continue
			}
			if msg == nil {
				continue
			}

			// Handle text messages
			if msg.Type == "text" && handler.OnText != nil {
				if err := handler.OnText(msg); err != nil {
					return err
				}
			}

			// Handle result messages
			if msg.Type == "result" {
				if handler.OnResult != nil {
					if err := handler.OnResult(msg); err != nil {
						return err
					}
				}
				// Handle structured output
				if msg.StructuredOutput != nil && handler.OnStructuredOutput != nil {
					if err := handler.OnStructuredOutput(msg.StructuredOutput); err != nil {
						return err
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return NewError(ErrCodeParseError, "error reading stdout", err)
	}

	return nil
}

// readStderr reads and processes stderr line by line.
func (c *Client) readStderr(stderr io.Reader, handler *StreamHandler) error {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if handler.OnStderr != nil {
			if err := handler.OnStderr(line); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return NewError(ErrCodeParseError, "error reading stderr", err)
	}

	return nil
}

// ExecuteWithTimeout runs a Claude session with a timeout.
func (c *Client) ExecuteWithTimeout(config *SessionConfig, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Execute(ctx, config)
}

// Session represents an active Claude session.
type Session struct {
	client  *Client
	config  *SessionConfig
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{}
	output  strings.Builder
	mu      sync.Mutex
	result  interface{}
	lastErr error
}

// NewSession creates a new Claude session.
func (c *Client) NewSession(config *SessionConfig) *Session {
	ctx, cancel := context.WithCancel(context.Background())

	return &Session{
		client: c,
		config: config,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// Start starts the session in the background.
func (s *Session) Start() {
	handler := &StreamHandler{
		OnText: func(msg *StreamMessage) error {
			s.mu.Lock()
			defer s.mu.Unlock()
			if msg.Content != "" {
				s.output.WriteString(msg.Content)
				s.output.WriteString("\n")
			}
			return nil
		},
		OnStructuredOutput: func(output interface{}) error {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.result = output
			// Cancel the context to stop the process
			s.cancel()
			return nil
		},
	}

	go func() {
		defer close(s.done)
		s.lastErr = s.client.ExecuteStream(context.Background(), s.config, handler)
	}()
}

// Wait waits for the session to complete.
func (s *Session) Wait() error {
	<-s.done
	return s.lastErr
}

// Cancel cancels the running session.
func (s *Session) Cancel() {
	s.cancel()
}

// Output returns the accumulated output so far.
func (s *Session) Output() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output.String()
}

// Result returns the structured output if available.
func (s *Session) Result() (interface{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.result, s.result != nil
}

// Done returns a channel that is closed when the session completes.
func (s *Session) Done() <-chan struct{} {
	return s.done
}
