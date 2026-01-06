// Package log provides structured logging for supervisor operations.
package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Global logger for simple debug messages (used in hook.go before supervisor logger is created)
var globalLogger *Logger
var globalLoggerOnce sync.Once

func initGlobalLogger() {
	// Create a logger that only writes to stderr (no file)
	globalLogger = &Logger{
		file:     nil,
		filePath: "",
		writer:   os.Stderr,
		minLevel: DebugLevel,
	}
}

// Debug logs a debug message to stderr only.
func Debug(format string, args ...interface{}) {
	globalLoggerOnce.Do(initGlobalLogger)
	globalLogger.log(DebugLevel, format, args...)
}

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production.
	DebugLevel LogLevel = iota
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly, it shouldn't generate any error-level logs.
	ErrorLevel
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Color returns the ANSI color code for the log level.
func (l LogLevel) Color() string {
	switch l {
	case DebugLevel:
		return "\033[36m" // cyan
	case InfoLevel:
		return "\033[37m" // white
	case WarnLevel:
		return "\033[33m" // yellow
	case ErrorLevel:
		return "\033[31m" // red
	default:
		return "\033[0m" // reset
	}
}

// Logger is a structured logger that writes to both file and stderr.
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	filePath string
	writer   io.Writer
	minLevel LogLevel
}

// LoggerOption is a function that configures a Logger.
type LoggerOption func(*Logger)

// WithMinLevel sets the minimum log level.
func WithMinLevel(level LogLevel) LoggerOption {
	return func(l *Logger) {
		l.minLevel = level
	}
}

// NewLogger creates a new logger that writes to the specified file.
func NewLogger(filePath string, opts ...LoggerOption) (*Logger, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := &Logger{
		file:     file,
		filePath: filePath,
		writer:   io.MultiWriter(os.Stderr, file),
		minLevel: InfoLevel,
	}

	for _, opt := range opts {
		opt(logger)
	}

	return logger, nil
}

// Close closes the log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// log writes a log message at the specified level.
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.minLevel {
		return
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
	message := fmt.Sprintf(format, args...)

	// Write to file (always with full detail)
	fileLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)
	if l.file != nil {
		l.file.WriteString(fileLine)
		l.file.Sync()
	}

	// Write to stderr (with colors for terminal)
	color := level.Color()
	reset := "\033[0m"
	stderrLine := fmt.Sprintf("%s[%s] [%s]%s %s\n", color, timestamp, level, reset, message)
	fmt.Fprint(os.Stderr, stderrLine)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Section writes a section header to the log.
func (l *Logger) Section(title string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	separator := "======================================================================"
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")

	// Write to file
	if l.file != nil {
		l.file.WriteString("\n" + separator + "\n")
		l.file.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, title))
		l.file.WriteString(separator + "\n\n")
		l.file.Sync()
	}

	// Write to stderr
	fmt.Fprintln(os.Stderr, "\n"+separator)
	fmt.Fprintf(os.Stderr, "[%s] %s\n", timestamp, title)
	fmt.Fprintln(os.Stderr, separator+"\n")
}

// Subsection writes a subsection header to the log.
func (l *Logger) Subsection(title string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	separator := "----------------------------------------------------------------------"
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")

	// Write to file
	if l.file != nil {
		l.file.WriteString(separator + "\n")
		l.file.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, title))
		l.file.WriteString(separator + "\n")
		l.file.Sync()
	}

	// Write to stderr
	fmt.Fprintln(os.Stderr, separator)
	fmt.Fprintf(os.Stderr, "[%s] %s\n", timestamp, title)
	fmt.Fprintln(os.Stderr, separator)
}

// KeyValue writes a key-value pair to the log.
func (l *Logger) KeyValue(key string, value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
	line := fmt.Sprintf("[%s] %s: %v\n", timestamp, key, value)

	// Write to file
	if l.file != nil {
		l.file.WriteString(line)
		l.file.Sync()
	}

	// Write to stderr
	fmt.Fprint(os.Stderr, line)
}

// FilePath returns the path to the log file.
func (l *Logger) FilePath() string {
	return l.filePath
}

// SupervisorLogger is a specialized logger for supervisor operations.
type SupervisorLogger struct {
	*Logger
	sessionID string
}

// NewSupervisorLogger creates a new supervisor logger for the given session.
func NewSupervisorLogger(sessionID string) (*SupervisorLogger, error) {
	stateDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	stateDir = filepath.Join(stateDir, ".claude", "ccc")

	logPath := filepath.Join(stateDir, fmt.Sprintf("supervisor-%s.log", sessionID))
	logger, err := NewLogger(logPath)
	if err != nil {
		return nil, err
	}

	return &SupervisorLogger{
		Logger:    logger,
		sessionID: sessionID,
	}, nil
}

// SessionID returns the session ID for this logger.
func (l *SupervisorLogger) SessionID() string {
	return l.sessionID
}

// LogHookStart logs the start of a hook execution.
func (l *SupervisorLogger) LogHookStart(sessionID string, stopHookActive bool, args []string) {
	l.Section("SUPERVISOR HOOK STARTED")
	l.KeyValue("Session ID", sessionID)
	l.KeyValue("Stop Hook Active", stopHookActive)
	l.KeyValue("Args", args)
}

// LogInput logs the hook input JSON.
func (l *SupervisorLogger) LogInput(inputJSON string) {
	l.Subsection("HOOK INPUT")
	l.Info("%s", inputJSON)
}

// LogIterationCheck logs the iteration count check.
func (l *SupervisorLogger) LogIterationCheck(count, maxIterations int, shouldContinue bool) {
	l.Subsection("ITERATION CHECK")
	l.KeyValue("Current Count", count)
	l.KeyValue("Max Iterations", maxIterations)
	if shouldContinue {
		l.Info("Continuing (count < max)")
	} else {
		l.Warn("Stopping (max iterations reached)")
	}
}

// LogSupervisorCommand logs the command being executed for the supervisor.
func (l *SupervisorLogger) LogSupervisorCommand(command string, argsCount int) {
	l.Subsection("SUPERVISOR COMMAND")
	l.KeyValue("Command", command)
	l.KeyValue("Args Count", argsCount)
}

// LogStreamMessage logs a stream message from Claude.
func (l *SupervisorLogger) LogStreamMessage(line string) {
	l.Debug("Stream: %s", line)
}

// LogTextContent logs text content from Claude.
func (l *SupervisorLogger) LogTextContent(content string) {
	l.Info("Content: %s", content)
}

// LogSupervisorResult logs the supervisor's structured result.
func (l *SupervisorLogger) LogSupervisorResult(resultJSON string, completed bool, feedback string) {
	l.Section("SUPERVISOR RESULT")
	l.Subsection("STRUCTURED OUTPUT")
	l.Info("%s", resultJSON)
	l.Subsection("ANALYSIS")
	if completed {
		l.Info("Task completed - allowing stop")
	} else {
		l.Warn("Task not completed - blocking with feedback")
		l.KeyValue("Feedback", feedback)
	}
}

// LogHookDecision logs the final hook decision.
func (l *SupervisorLogger) LogHookDecision(decision string, reason string) {
	l.Section("HOOK DECISION")
	l.KeyValue("Decision", decision)
	if reason != "" {
		l.KeyValue("Reason", reason)
	}
}

// LogError logs an error with context.
func (l *SupervisorLogger) LogError(context string, err error) {
	l.Error("%s: %v", context, err)
}

// LogCommandComplete logs the completion of the supervisor command.
func (l *SupervisorLogger) LogCommandComplete(err error) {
	l.Subsection("COMMAND COMPLETION")
	if err != nil {
		l.Error("Command finished with error: %v", err)
	} else {
		l.Info("Command completed successfully")
	}
}
