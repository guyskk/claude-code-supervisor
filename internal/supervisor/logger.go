// Package supervisor provides supervisor-specific logging functionality.
package supervisor

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/guyskk/ccc/internal/logger"
)

// SupervisorLogger is a logger that outputs to both stderr and a log file.
// If supervisorID is empty, it only outputs to stderr.
// If supervisorID is non-empty, it outputs to both stderr and a log file.
type SupervisorLogger struct {
	stderrLogger logger.Logger
	fileLogger   logger.Logger
	mu           sync.Mutex
	closed       bool
}

// NewSupervisorLogger creates a new SupervisorLogger.
//
// If supervisorID is empty, only stderr output is enabled.
// If supervisorID is non-empty, both stderr and log file output are enabled.
// The log file is created at ~/.claude/ccc/supervisor-{supervisorID}.log
//
// Errors are logged to stderr and a fallback logger is returned.
func NewSupervisorLogger(supervisorID string) logger.Logger {
	// Create stderr logger (always enabled)
	stderrLogger := logger.NewLogger(os.Stderr, logger.LevelDebug)

	if supervisorID == "" {
		return stderrLogger
	}

	// Create log file for supervisor session
	stateDir, err := GetStateDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get state directory: %v\n", err)
		return stderrLogger
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create state directory: %v\n", err)
		return stderrLogger
	}

	logFilePath := filepath.Join(stateDir, fmt.Sprintf("supervisor-%s.log", supervisorID))
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open supervisor log file: %v\n", err)
		return stderrLogger
	}

	// Create file logger with debug level
	fileLogger := logger.NewLogger(logFile, logger.LevelDebug).With(
		logger.StringField("supervisor_id", supervisorID),
	)

	supervisorLogger := &SupervisorLogger{
		stderrLogger: stderrLogger,
		fileLogger:   fileLogger,
	}

	// Wrap with supervisor_id field for all log entries
	resultLogger := logger.Logger(supervisorLogger)
	resultLogger = resultLogger.With(logger.StringField("supervisor_id", supervisorID))

	return resultLogger
}

// Debug logs a debug message to both stderr and file (if enabled).
func (l *SupervisorLogger) Debug(msg string, fields ...logger.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stderrLogger.Debug(msg, fields...)
	if l.fileLogger != nil && !l.closed {
		l.fileLogger.Debug(msg, fields...)
	}
}

// Info logs an info message to both stderr and file (if enabled).
func (l *SupervisorLogger) Info(msg string, fields ...logger.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stderrLogger.Info(msg, fields...)
	if l.fileLogger != nil && !l.closed {
		l.fileLogger.Info(msg, fields...)
	}
}

// Warn logs a warning message to both stderr and file (if enabled).
func (l *SupervisorLogger) Warn(msg string, fields ...logger.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stderrLogger.Warn(msg, fields...)
	if l.fileLogger != nil && !l.closed {
		l.fileLogger.Warn(msg, fields...)
	}
}

// Error logs an error message to both stderr and file (if enabled).
func (l *SupervisorLogger) Error(msg string, fields ...logger.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stderrLogger.Error(msg, fields...)
	if l.fileLogger != nil && !l.closed {
		l.fileLogger.Error(msg, fields...)
	}
}

// With returns a new logger with additional fields.
func (l *SupervisorLogger) With(fields ...logger.Field) logger.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	return &SupervisorLogger{
		stderrLogger: l.stderrLogger.With(fields...),
		fileLogger:   l.withFileLogger(fields),
	}
}

// withFileLogger creates a new file logger with additional fields, if file logger is enabled.
func (l *SupervisorLogger) withFileLogger(fields []logger.Field) logger.Logger {
	if l.fileLogger == nil {
		return nil
	}
	return l.fileLogger.With(fields...)
}

// WithError returns a new logger with an error field.
func (l *SupervisorLogger) WithError(err error) logger.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	return &SupervisorLogger{
		stderrLogger: l.stderrLogger.WithError(err),
		fileLogger:   l.withFileLoggerError(err),
	}
}

// withFileLoggerError creates a new file logger with error field, if file logger is enabled.
func (l *SupervisorLogger) withFileLoggerError(err error) logger.Logger {
	if l.fileLogger == nil {
		return nil
	}
	return l.fileLogger.WithError(err)
}

// Close closes the log file if it was opened.
func (l *SupervisorLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true

	// Close the file logger if it exists
	if closer, ok := l.fileLogger.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

// IsFileLoggingEnabled returns true if file logging is enabled.
func (l *SupervisorLogger) IsFileLoggingEnabled() bool {
	return l.fileLogger != nil
}
