package logging

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, got, tt.expected)
		}
	}
}

func TestExtractCorrelationID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "nil context",
			ctx:      nil,
			expected: "",
		},
		{
			name:     "empty context",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "with correlation ID",
			ctx:      WithCorrelationID(context.Background(), "abc-123"),
			expected: "abc-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCorrelationID(tt.ctx)
			if got != tt.expected {
				t.Errorf("ExtractCorrelationID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWithCorrelationID(t *testing.T) {
	ctx := WithCorrelationID(nil, "test-id")
	if ctx == nil {
		t.Error("WithCorrelationID(nil) should return non-nil context")
	}

	if id := ExtractCorrelationID(ctx); id != "test-id" {
		t.Errorf("expected correlation ID 'test-id', got %q", id)
	}
}

func TestNoopLogger(t *testing.T) {
	logger := &NoopLogger{}

	// These should not panic
	logger.Debug("debug message", Fields{"key": "value"})
	logger.Info("info message", Fields{"key": "value"})
	logger.Warn("warn message", Fields{"key": "value"})
	logger.Error("error message", Fields{"key": "value"})

	// WithFields should return the same logger
	newLogger := logger.WithFields(Fields{"new": "field"})
	if newLogger != logger {
		t.Error("NoopLogger.WithFields() should return the same logger")
	}

	// WithContext should return the same logger
	ctxLogger := logger.WithContext(context.Background())
	if ctxLogger != logger {
		t.Error("NoopLogger.WithContext() should return the same logger")
	}
}

func TestStandardLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	// Set the global log output to our buffer for testing
	oldOutput := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(oldOutput)

	config := LogConfig{
		Level:  WARN,
		Format: "text",
		Output: &buf,
	}
	logger := NewStandardLogger(config)

	logger.Info("should not appear", nil)
	logger.Debug("should not appear", nil)
	logger.Warn("should appear warn", nil)
	logger.Error("should appear error", nil)

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("Log entries below WARN level should not appear")
	}
	if !strings.Contains(output, "should appear warn") {
		t.Errorf("Log entries at WARN level should appear, got: %s", output)
	}
}

func TestStandardLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	config := LogConfig{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}
	logger := NewStandardLogger(config)
	loggerWithFields := logger.WithFields(Fields{"service": "test"})

	loggerWithFields.Info("test message", Fields{"key": "value"})

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Error("Expected output to contain message")
	}
	if !strings.Contains(output, "service") {
		t.Error("Expected output to contain persistent field")
	}
	if !strings.Contains(output, "key") {
		t.Error("Expected output to contain log-specific field")
	}
}

func TestStandardLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	config := LogConfig{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}
	logger := NewStandardLogger(config)

	ctx := WithCorrelationID(context.Background(), "corr-123")
	ctxLogger := logger.WithContext(ctx)

	ctxLogger.Info("test message", nil)

	output := buf.String()
	if !strings.Contains(output, "corr-123") {
		t.Errorf("Expected output to contain correlation_id 'corr-123', got: %s", output)
	}
}

func TestStandardLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	config := LogConfig{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}
	logger := NewStandardLogger(config)

	logger.Info("test message", Fields{"key": "value", "number": 42})

	output := buf.String()
	// Check for JSON structure
	if !strings.Contains(output, `"timestamp"`) {
		t.Error("Expected JSON output to contain timestamp")
	}
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Error("Expected JSON output to contain level")
	}
	if !strings.Contains(output, `"message":"test message"`) {
		t.Error("Expected JSON output to contain message")
	}
	if !strings.Contains(output, `"fields"`) {
		t.Error("Expected JSON output to contain fields")
	}
}

func TestStandardLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	// Set the global log output to our buffer for testing
	oldOutput := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(oldOutput)

	config := LogConfig{
		Level:  INFO,
		Format: "text",
		Output: &buf,
	}
	logger := NewStandardLogger(config)

	logger.Info("test message", Fields{"key": "value"})

	output := buf.String()
	// Text format uses standard log package
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected text output to contain level, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected text output to contain message, got: %s", output)
	}
}

func TestStandardLogger_AllLevels(t *testing.T) {
	tests := []struct {
		level   LogLevel
		logFunc func(l *StandardLogger, msg string, fields Fields)
	}{
		{DEBUG, func(l *StandardLogger, msg string, fields Fields) { l.Debug(msg, fields) }},
		{INFO, func(l *StandardLogger, msg string, fields Fields) { l.Info(msg, fields) }},
		{WARN, func(l *StandardLogger, msg string, fields Fields) { l.Warn(msg, fields) }},
		{ERROR, func(l *StandardLogger, msg string, fields Fields) { l.Error(msg, fields) }},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		config := LogConfig{
			Level:  DEBUG, // Allow all levels
			Format: "json",
			Output: &buf,
		}
		logger := NewStandardLogger(config)

		tt.logFunc(logger, "test", nil)

		output := buf.String()
		if !strings.Contains(output, tt.level.String()) {
			t.Errorf("Expected output to contain level %s, got: %s", tt.level.String(), output)
		}
	}
}

func TestDefaultLogConfig(t *testing.T) {
	config := DefaultLogConfig()

	if config.Level != INFO {
		t.Errorf("expected Level=INFO, got %v", config.Level)
	}
	if config.Format != "text" {
		t.Errorf("expected Format='text', got %q", config.Format)
	}
	if config.Output == nil {
		t.Error("expected non-nil Output")
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := DefaultLogger()
	if logger == nil {
		t.Error("expected non-nil Logger from DefaultLogger()")
	}

	// Should be NoopLogger
	_, ok := logger.(*NoopLogger)
	if !ok {
		t.Error("expected DefaultLogger() to return NoopLogger")
	}
}

func TestNewStandardLogger_DefaultOutput(t *testing.T) {
	var buf bytes.Buffer
	// Set the global log output to our buffer for testing
	oldOutput := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(oldOutput)

	config := LogConfig{
		Level:  ERROR,
		Format: "text",
		// Output is nil, should default to os.Stderr
	}
	logger := NewStandardLogger(config)
	if logger == nil {
		t.Error("expected non-nil logger")
	}

	// Should not panic
	logger.Error("test", nil)
}

func BenchmarkStandardLogger_Info(b *testing.B) {
	var buf bytes.Buffer
	config := LogConfig{
		Level:  INFO,
		Format: "json",
		Output: &buf,
	}
	logger := NewStandardLogger(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", Fields{"iteration": i})
	}
}

func BenchmarkNoopLogger_Info(b *testing.B) {
	logger := &NoopLogger{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", Fields{"iteration": i})
	}
}
