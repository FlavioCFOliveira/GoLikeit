// Package logging provides structured logging functionality.
package logging

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry.
type LogLevel int

const (
	// DEBUG level for detailed debugging information.
	DEBUG LogLevel = iota
	// INFO level for informational messages.
	INFO
	// WARN level for warning messages.
	WARN
	// ERROR level for error messages.
	ERROR
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Fields represents structured logging fields.
type Fields map[string]interface{}

// Logger is the interface for structured logging.
type Logger interface {
	// Debug logs a debug message with the given fields.
	Debug(msg string, fields Fields)
	// Info logs an info message with the given fields.
	Info(msg string, fields Fields)
	// Warn logs a warning message with the given fields.
	Warn(msg string, fields Fields)
	// Error logs an error message with the given fields.
	Error(msg string, fields Fields)
	// WithFields returns a new Logger with the given fields added to each log entry.
	WithFields(fields Fields) Logger
	// WithContext returns a new Logger with correlation_id from context.
	WithContext(ctx context.Context) Logger
}

// contextKey is the type for context keys.
type contextKey string

// correlationIDKey is the context key for correlation ID.
const correlationIDKey contextKey = "correlation_id"

// ExtractCorrelationID extracts the correlation ID from the context.
func ExtractCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithCorrelationID returns a new context with the correlation ID set.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, correlationIDKey, id)
}

// LogConfig configures the StandardLogger.
type LogConfig struct {
	// Level is the minimum log level to output.
	Level LogLevel
	// Format is the output format: "json" or "text".
	Format string
	// Output is the writer for log output. Defaults to os.Stderr.
	Output io.Writer
}

// DefaultLogConfig returns a LogConfig with sensible defaults.
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:  INFO,
		Format: "text",
		Output: os.Stderr,
	}
}

// StandardLogger implements Logger using the standard library log package.
type StandardLogger struct {
	config   LogConfig
	fields   Fields
	mu       sync.RWMutex
	encoder  *json.Encoder
	once     sync.Once
}

// NewStandardLogger creates a new StandardLogger with the given configuration.
func NewStandardLogger(config LogConfig) *StandardLogger {
	if config.Output == nil {
		config.Output = os.Stderr
	}
	return &StandardLogger{
		config:  config,
		fields:  make(Fields),
		encoder: json.NewEncoder(config.Output),
	}
}

// logEntry represents a single log entry for JSON output.
type logEntry struct {
	Timestamp      string                 `json:"timestamp"`
	Level          string                 `json:"level"`
	Message        string                 `json:"message"`
	CorrelationID  string                 `json:"correlation_id,omitempty"`
	Fields         map[string]interface{} `json:"fields,omitempty"`
}

// log writes a log entry at the given level.
func (l *StandardLogger) log(level LogLevel, msg string, fields Fields) {
	if level < l.config.Level {
		return
	}

	l.mu.RLock()
	allFields := make(Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		allFields[k] = v
	}
	l.mu.RUnlock()

	for k, v := range fields {
		allFields[k] = v
	}

	correlationID := ""
	if id, ok := allFields["correlation_id"].(string); ok {
		correlationID = id
		delete(allFields, "correlation_id")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	if l.config.Format == "json" {
		entry := logEntry{
			Timestamp:     timestamp,
			Level:         level.String(),
			Message:       msg,
			CorrelationID: correlationID,
			Fields:        allFields,
		}
		l.encoder.Encode(entry)
	} else {
		// Text format
		var fieldsStr string
		if len(allFields) > 0 {
			fieldsJSON, _ := json.Marshal(allFields)
			fieldsStr = string(fieldsJSON)
		}

		if correlationID != "" {
			log.Printf("[%s] %s %s correlation_id=%s %s", timestamp, level.String(), msg, correlationID, fieldsStr)
		} else {
			log.Printf("[%s] %s %s %s", timestamp, level.String(), msg, fieldsStr)
		}
	}
}

// Debug implements Logger.
func (l *StandardLogger) Debug(msg string, fields Fields) {
	l.log(DEBUG, msg, fields)
}

// Info implements Logger.
func (l *StandardLogger) Info(msg string, fields Fields) {
	l.log(INFO, msg, fields)
}

// Warn implements Logger.
func (l *StandardLogger) Warn(msg string, fields Fields) {
	l.log(WARN, msg, fields)
}

// Error implements Logger.
func (l *StandardLogger) Error(msg string, fields Fields) {
	l.log(ERROR, msg, fields)
}

// WithFields implements Logger.
func (l *StandardLogger) WithFields(fields Fields) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := make(Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &StandardLogger{
		config:  l.config,
		fields:  newFields,
		encoder: l.encoder,
	}
}

// WithContext implements Logger.
func (l *StandardLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}
	correlationID := ExtractCorrelationID(ctx)
	if correlationID == "" {
		return l
	}
	return l.WithFields(Fields{"correlation_id": correlationID})
}

// NoopLogger is a Logger implementation that does nothing.
type NoopLogger struct{}

// Debug implements Logger.
func (n *NoopLogger) Debug(msg string, fields Fields) {}

// Info implements Logger.
func (n *NoopLogger) Info(msg string, fields Fields) {}

// Warn implements Logger.
func (n *NoopLogger) Warn(msg string, fields Fields) {}

// Error implements Logger.
func (n *NoopLogger) Error(msg string, fields Fields) {}

// WithFields implements Logger.
func (n *NoopLogger) WithFields(fields Fields) Logger {
	return n
}

// WithContext implements Logger.
func (n *NoopLogger) WithContext(ctx context.Context) Logger {
	return n
}

// DefaultLogger returns a Logger that discards all log output.
// This is the default when no logging is configured.
func DefaultLogger() Logger {
	return &NoopLogger{}
}

// logWriter wraps an io.Writer to implement io.Writer for log package.
type logWriter struct {
	writer io.Writer
}

// Write implements io.Writer.
func (w *logWriter) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}

// init initializes the standard log package output.
func init() {
	log.SetFlags(0) // Remove default flags since we handle formatting
}
