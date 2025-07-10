package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
}

// New creates a new logger with the specified level and format
func New(level, format string) (*Logger, error) {
	// Parse log level
	var logLevel slog.Level
	switch strings.ToUpper(level) {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}

	// Create handler based on format
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		return nil, fmt.Errorf("invalid log format: %s (must be 'json' or 'text')", format)
	}

	// Create logger
	logger := slog.New(handler)

	return &Logger{Logger: logger}, nil
}

// Debug logs a debug message with optional key-value pairs
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.Debug(msg, keysAndValues...)
}

// Info logs an info message with optional key-value pairs
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, keysAndValues...)
}

// Warn logs a warning message with optional key-value pairs
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.Logger.Warn(msg, keysAndValues...)
}

// Error logs an error message with optional key-value pairs
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, keysAndValues...)
}

// With returns a new logger with the given key-value pairs added to the context
func (l *Logger) With(keysAndValues ...interface{}) *Logger {
	return &Logger{Logger: l.Logger.With(keysAndValues...)}
}

// WithGroup returns a new logger with the given group name
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{Logger: l.Logger.WithGroup(name)}
}

// LogRequest logs an HTTP request with structured fields
func (l *Logger) LogRequest(method, path, userAgent, remoteAddr string, statusCode int, duration string) {
	l.Info("HTTP request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration", duration,
		"user_agent", userAgent,
		"remote_addr", remoteAddr,
	)
}

// LogGitHubAPICall logs a GitHub API call with structured fields
func (l *Logger) LogGitHubAPICall(method, endpoint string, statusCode int, duration string, rateLimitRemaining int) {
	l.Info("GitHub API call",
		"method", method,
		"endpoint", endpoint,
		"status_code", statusCode,
		"duration", duration,
		"rate_limit_remaining", rateLimitRemaining,
	)
}

// LogError logs an error with additional context
func (l *Logger) LogError(err error, msg string, keysAndValues ...interface{}) {
	args := append([]interface{}{"error", err}, keysAndValues...)
	l.Error(msg, args...)
}
