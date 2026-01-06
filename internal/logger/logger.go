// Package logger provides structured logging for ccc.
package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Level represents the logging level.
type Level int

const (
	// LevelDebug is the debug logging level.
	LevelDebug Level = iota
	// LevelInfo is the info logging level.
	LevelInfo
	// LevelWarn is the warning logging level.
	LevelWarn
	// LevelError is the error logging level.
	LevelError
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string to a Level.
func ParseLevel(s string) (Level, error) {
	switch s {
	case "debug", "DEBUG":
		return LevelDebug, nil
	case "info", "INFO":
		return LevelInfo, nil
	case "warn", "WARN", "warning", "WARNING":
		return LevelWarn, nil
	case "error", "ERROR":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level: %s", s)
	}
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// Logger is the interface for structured logging.
type Logger interface {
	// Debug logs a debug message.
	Debug(msg string, fields ...Field)
	// Info logs an info message.
	Info(msg string, fields ...Field)
	// Warn logs a warning message.
	Warn(msg string, fields ...Field)
	// Error logs an error message.
	Error(msg string, fields ...Field)
	// With returns a new logger with additional fields.
	With(fields ...Field) Logger
	// WithError returns a new logger with an error field.
	WithError(err error) Logger
}

// slogLogger implements Logger using the standard library's log/slog.
type slogLogger struct {
	logger *slog.Logger
	level  Level
	mu     sync.Mutex
}

// NewLogger creates a new Logger that writes to w.
func NewLogger(w io.Writer, level Level) Logger {
	// Create a custom text handler
	handler := newCustomTextHandler(w, nil, level)

	return &slogLogger{
		logger: slog.New(handler),
		level:  level,
	}
}

// NewNopLogger returns a no-op logger that discards all log output.
func NewNopLogger() Logger {
	return &nopLogger{}
}

// NewDefaultLogger creates a new Logger that writes to stderr with INFO level.
func NewDefaultLogger() Logger {
	return NewLogger(os.Stderr, LevelInfo)
}

// Debug logs a debug message.
func (l *slogLogger) Debug(msg string, fields ...Field) {
	if l.level > LevelDebug {
		return
	}
	l.log(LevelDebug, msg, fields...)
}

// Info logs an info message.
func (l *slogLogger) Info(msg string, fields ...Field) {
	if l.level > LevelInfo {
		return
	}
	l.log(LevelInfo, msg, fields...)
}

// Warn logs a warning message.
func (l *slogLogger) Warn(msg string, fields ...Field) {
	if l.level > LevelWarn {
		return
	}
	l.log(LevelWarn, msg, fields...)
}

// Error logs an error message.
func (l *slogLogger) Error(msg string, fields ...Field) {
	if l.level > LevelError {
		return
	}
	l.log(LevelError, msg, fields...)
}

// With returns a new logger with additional fields.
func (l *slogLogger) With(fields ...Field) Logger {
	// Convert fields to any slice for slog
	args := make([]any, len(fields)*2)
	for i, f := range fields {
		args[i*2] = f.Key
		args[i*2+1] = f.Value
	}
	return &slogLogger{
		logger: l.logger.With(args...),
		level:  l.level,
	}
}

// WithError returns a new logger with an error field.
func (l *slogLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.With(Field{Key: "error", Value: err.Error()})
}

// log is the internal logging method.
func (l *slogLogger) log(level Level, msg string, fields ...Field) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Convert our Level to slog.Level
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	// Build the log message
	var attrs []slog.Attr
	attrs = append(attrs, slog.String("level", level.String()))
	attrs = append(attrs, fieldsToAttrs(fields)...)

	l.logger.LogAttrs(context.Background(), slogLevel, msg, attrs...)
}

// fieldsToAttrs converts Field slice to slog.Attr slice.
func fieldsToAttrs(fields []Field) []slog.Attr {
	attrs := make([]slog.Attr, len(fields))
	for i, f := range fields {
		attrs[i] = slog.Any(f.Key, f.Value)
	}
	return attrs
}

// textHandler is a custom handler that formats log entries.
type textHandler struct {
	w    io.Writer
	mu   sync.Mutex
	opts *slog.HandlerOptions
	level Level
}

func newCustomTextHandler(w io.Writer, opts *slog.HandlerOptions, level Level) *textHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &textHandler{w: w, opts: opts, level: level}
}

func (h *textHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Check if level is enabled
	minLevel := slog.LevelInfo
	switch h.level {
	case LevelDebug:
		minLevel = slog.LevelDebug
	case LevelInfo:
		minLevel = slog.LevelInfo
	case LevelWarn:
		minLevel = slog.LevelWarn
	case LevelError:
		minLevel = slog.LevelError
	}
	return level >= minLevel
}

func (h *textHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Build the log line: [timestamp] [level] message key=value...
	var buf bytes.Buffer

	// Timestamp
	timestamp := r.Time.Format("2006-01-02T15:04:05.000Z")
	buf.WriteString("[")
	buf.WriteString(timestamp)
	buf.WriteString("] ")

	// Level (convert slog.Level to our Level string)
	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelInfo:
		levelStr = "INFO"
	case slog.LevelWarn:
		levelStr = "WARN"
	case slog.LevelError:
		levelStr = "ERROR"
	default:
		levelStr = "INFO"
	}
	buf.WriteString("[")
	buf.WriteString(levelStr)
	buf.WriteString("] ")

	// Message
	buf.WriteString(r.Message)

	// Attributes
	r.Attrs(func(a slog.Attr) bool {
		// Skip the level attribute we added
		if a.Key == "level" {
			return true
		}
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(formatAttrValue(a.Value))
		return true
	})

	buf.WriteString("\n")

	// Write to output
	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Return a new handler with additional attributes
	// For simplicity, we'll just store them and add them during Handle
	return &textHandlerWithAttrs{
		textHandler: h,
		attrs:       attrs,
	}
}

func (h *textHandler) WithGroup(name string) slog.Handler {
	// For simplicity, just return self (groups not fully implemented)
	return h
}

// formatAttrValue formats an attribute value as a string
func formatAttrValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return fmt.Sprintf("%d", v.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", v.Uint64())
	case slog.KindFloat64:
		return fmt.Sprintf("%f", v.Float64())
	case slog.KindBool:
		return fmt.Sprintf("%t", v.Bool())
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	default:
		// For other types, use Any() representation
		return fmt.Sprintf("%v", v.Any())
	}
}

// textHandlerWithAttrs is a handler with pre-set attributes
type textHandlerWithAttrs struct {
	textHandler *textHandler
	attrs       []slog.Attr
}

func (h *textHandlerWithAttrs) Enabled(ctx context.Context, level slog.Level) bool {
	return h.textHandler.Enabled(ctx, level)
}

func (h *textHandlerWithAttrs) Handle(ctx context.Context, r slog.Record) error {
	// Add the pre-set attributes to the record
	// We need to create a new record that includes these attributes
	// For simplicity, we'll modify the handler to add them during formatting
	h.textHandler.mu.Lock()
	defer h.textHandler.mu.Unlock()

	var buf bytes.Buffer

	// Timestamp
	timestamp := r.Time.Format("2006-01-02T15:04:05.000Z")
	buf.WriteString("[")
	buf.WriteString(timestamp)
	buf.WriteString("] ")

	// Level
	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelInfo:
		levelStr = "INFO"
	case slog.LevelWarn:
		levelStr = "WARN"
	case slog.LevelError:
		levelStr = "ERROR"
	default:
		levelStr = "INFO"
	}
	buf.WriteString("[")
	buf.WriteString(levelStr)
	buf.WriteString("] ")

	// Message
	buf.WriteString(r.Message)

	// Pre-set attributes
	for _, a := range h.attrs {
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(formatAttrValue(a.Value))
	}

	// Record attributes
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "level" {
			return true
		}
		buf.WriteString(" ")
		buf.WriteString(a.Key)
		buf.WriteString("=")
		buf.WriteString(formatAttrValue(a.Value))
		return true
	})

	buf.WriteString("\n")

	_, err := h.textHandler.w.Write(buf.Bytes())
	return err
}

func (h *textHandlerWithAttrs) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &textHandlerWithAttrs{
		textHandler: h.textHandler,
		attrs:       append(h.attrs, attrs...),
	}
}

func (h *textHandlerWithAttrs) WithGroup(name string) slog.Handler {
	return h
}

// nopLogger is a logger that discards all output.
type nopLogger struct{}

func (n *nopLogger) Debug(msg string, fields ...Field) {}
func (n *nopLogger) Info(msg string, fields ...Field)  {}
func (n *nopLogger) Warn(msg string, fields ...Field)  {}
func (n *nopLogger) Error(msg string, fields ...Field) {}
func (n *nopLogger) With(fields ...Field) Logger      { return n }
func (n *nopLogger) WithError(err error) Logger        { return n }

// Global default logger
var defaultLogger Logger = NewDefaultLogger()

// SetDefault sets the default logger.
func SetDefault(l Logger) {
	defaultLogger = l
}

// Default returns the default logger.
func Default() Logger {
	return defaultLogger
}

// Convenience functions using the default logger

// Debug logs a debug message using the default logger.
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

// Info logs an info message using the default logger.
func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

// Warn logs a warning message using the default logger.
func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

// Error logs an error message using the default logger.
func Error(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}

// With returns a new logger with additional fields using the default logger.
func With(fields ...Field) Logger {
	return defaultLogger.With(fields...)
}

// WithError returns a new logger with an error field using the default logger.
func WithError(err error) Logger {
	return defaultLogger.WithError(err)
}

// StringField creates a string field.
func StringField(key, value string) Field {
	return Field{Key: key, Value: value}
}

// IntField creates an int field.
func IntField(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// DurationField creates a duration field.
func DurationField(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// ErrField creates an error field.
func ErrField(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: "<nil>"}
	}
	return Field{Key: "error", Value: err.Error()}
}
