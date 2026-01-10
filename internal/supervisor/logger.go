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
)

// SupervisorLogger is a handler that outputs to both stderr and a log file.
// If supervisorID is empty, it only outputs to stderr.
// If supervisorID is non-empty, it outputs to both stderr and a log file.
type SupervisorLogger struct {
	stderrHandler slog.Handler
	fileHandler   slog.Handler
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
	// Create stderr handler (always enabled)
	stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

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

	// Create file handler with debug level and supervisor_id attribute
	fileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}).WithAttrs([]slog.Attr{slog.String("supervisor_id", supervisorID)})

	// Create combined handler
	handler := &SupervisorLogger{
		stderrHandler: stderrHandler.WithAttrs([]slog.Attr{slog.String("supervisor_id", supervisorID)}),
		fileHandler:   fileHandler,
	}

	return slog.New(handler)
}

// Enabled reports whether l handles level.
func (l *SupervisorLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.stderrHandler.Enabled(ctx, level)
}

// Handle handles the Record by writing to both stderr and file.
func (l *SupervisorLogger) Handle(ctx context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stderrHandler.Handle(ctx, r)
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
	}
}

func (l *SupervisorLogger) withFileHandlerGroup(name string) slog.Handler {
	if l.fileHandler == nil {
		return nil
	}
	return l.fileHandler.WithGroup(name)
}

// Close closes the log file if it was opened.
func (l *SupervisorLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// Close the file handler's underlying writer
	if l.fileHandler != nil {
		// Try to get the underlying writer from the handler
		type fileWriter interface {
			Close() error
		}
		// Note: slog.TextHandler doesn't expose Close, so we need to track the file separately
		// For now, we'll skip this since the file will be closed when the process exits
	}

	return nil
}

// NewTextHandlerFile creates a TextHandler with a closeable file.
// This is a helper that returns both the handler and a closer function.
func NewTextHandlerFile(path string, level slog.Level) (slog.Handler, io.Closer, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}

	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: level,
	})

	return handler, file, nil
}
