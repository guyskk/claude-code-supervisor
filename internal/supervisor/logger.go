// Package supervisor provides supervisor-specific logging functionality.
package supervisor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SupervisorLogger is a handler that outputs to both stderr and a log file.
// If supervisorID is empty, it only outputs to stderr.
// If supervisorID is non-empty, it outputs to both stderr and a log file.
type SupervisorLogger struct {
	stderrHandler slog.Handler
	fileHandler   slog.Handler
	logFile       *os.File // Track the file for proper cleanup
	mu            sync.Mutex
	closed        bool
}

// NewSupervisorLogger creates a new slog.Logger with SupervisorHandler.
//
// If supervisorID is empty, only stderr output is enabled.
// If supervisorID is non-empty, both stderr and log file output are enabled.
// The log file is created at ~/.claude/ccc/supervisor-{supervisorID}.log
//
// Errors are logged to stderr and a fallback logger is returned.
func NewSupervisorLogger(supervisorID string) *slog.Logger {
	// Create custom stderr handler with friendly format
	stderrHandler := newFriendlyTextHandler(os.Stderr, slog.LevelDebug)

	if supervisorID == "" {
		return slog.New(stderrHandler)
	}

	// Create log file for supervisor session
	stateDir, err := GetStateDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get state directory: %v\n", err)
		return slog.New(stderrHandler)
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create state directory: %v\n", err)
		return slog.New(stderrHandler)
	}

	logFilePath := filepath.Join(stateDir, fmt.Sprintf("supervisor-%s.log", supervisorID))
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open supervisor log file: %v\n", err)
		return slog.New(stderrHandler)
	}

	// Create file handler with debug level
	fileHandler := newFriendlyTextHandler(logFile, slog.LevelDebug)

	// Create combined handler
	handler := &SupervisorLogger{
		stderrHandler: stderrHandler,
		fileHandler:   fileHandler,
		logFile:       logFile,
	}

	return slog.New(handler)
}

// Enabled reports whether l handles level.
func (l *SupervisorLogger) Enabled(ctx context.Context, level slog.Level) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stderrHandler.Enabled(ctx, level)
}

// Handle handles the Record by writing to both stderr and file.
func (l *SupervisorLogger) Handle(ctx context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Always log to stderr
	l.stderrHandler.Handle(ctx, r)

	// Log to file if enabled and not closed
	if l.fileHandler != nil && !l.closed {
		l.fileHandler.Handle(ctx, r)
	}
	return nil
}

// WithAttrs returns a new Handler with the given attributes.
func (l *SupervisorLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	l.mu.Lock()
	defer l.mu.Unlock()

	return &SupervisorLogger{
		stderrHandler: l.stderrHandler.WithAttrs(attrs),
		fileHandler:   l.withFileHandlerAttrs(attrs),
		logFile:       l.logFile,
		closed:        false, // New handler is not closed
	}
}

func (l *SupervisorLogger) withFileHandlerAttrs(attrs []slog.Attr) slog.Handler {
	if l.fileHandler == nil {
		return nil
	}
	return l.fileHandler.WithAttrs(attrs)
}

// WithGroup returns a new Handler with the given group name.
func (l *SupervisorLogger) WithGroup(name string) slog.Handler {
	l.mu.Lock()
	defer l.mu.Unlock()

	return &SupervisorLogger{
		stderrHandler: l.stderrHandler.WithGroup(name),
		fileHandler:   l.withFileHandlerGroup(name),
		logFile:       l.logFile,
		closed:        false, // New handler is not closed
	}
}

func (l *SupervisorLogger) withFileHandlerGroup(name string) slog.Handler {
	if l.fileHandler == nil {
		return nil
	}
	return l.fileHandler.WithGroup(name)
}

// Close closes the log file if it was opened.
// This is safe to call multiple times.
func (l *SupervisorLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// Close the file handle
	if l.logFile != nil {
		if err := l.logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
	}

	return nil
}

// friendlyTextHandler is a custom slog.Handler that outputs logs in a friendly format.
type friendlyTextHandler struct {
	writer io.Writer
	level  slog.Level
	attrs  []slog.Attr // Pre-attributes set via WithAttrs
	mu     sync.Mutex
}

// newFriendlyTextHandler creates a new friendly text handler.
func newFriendlyTextHandler(w io.Writer, level slog.Level) *friendlyTextHandler {
	return &friendlyTextHandler{
		writer: w,
		level:  level,
		attrs:  nil,
	}
}

// Enabled reports whether h handles level.
func (h *friendlyTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle handles the Record by writing to the writer in a friendly format.
func (h *friendlyTextHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Format: 2026-01-10T16:09:39.061+08:00 INFO message key=value key=value
	_, err := fmt.Fprintf(h.writer, "%s %s %s",
		r.Time.Format(time.RFC3339Nano),
		r.Level.String(),
		r.Message,
	)

	// Write pre-set attributes first
	if h.attrs != nil {
		for _, a := range h.attrs {
			fmt.Fprintf(h.writer, " %s=%s", a.Key, a.Value)
		}
	}

	// Write record attributes
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(h.writer, " %s=%s", a.Key, a.Value)
		return true
	})

	fmt.Fprintln(h.writer)
	return err
}

// WithAttrs returns a new Handler with the given attributes.
func (h *friendlyTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := append(h.attrs, attrs...)
	return &friendlyTextHandler{
		writer: h.writer,
		level:  h.level,
		attrs:  newAttrs,
	}
}

// WithGroup returns a new Handler with the given group name.
//
// Note: This is a simplified implementation that preserves existing attributes
// but does not actually apply the group prefix. This is sufficient for the
// current supervisor logging use case, which does not use groups.
// If group functionality is needed in the future, this should be updated to
// properly prefix grouped attributes (e.g., "group.key=value").
func (h *friendlyTextHandler) WithGroup(name string) slog.Handler {
	return &friendlyTextHandler{
		writer: h.writer,
		level:  h.level,
		attrs:  append([]slog.Attr{}, h.attrs...),
	}
}
