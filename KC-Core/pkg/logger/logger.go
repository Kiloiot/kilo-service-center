// Package logger provides structured logging for KiloCenter
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	pkgcontext "github.com/kilocenter/pkg/context"
)

// Level represents the logging level
type Level int

// Logging levels from most to least verbose
const (
	// DebugLevel enables debug and all higher severity logs
	DebugLevel Level = iota
	// InfoLevel enables info and all higher severity logs
	InfoLevel
	// WarnLevel enables warning and all higher severity logs
	WarnLevel
	// ErrorLevel enables error and fatal logs
	ErrorLevel
	// FatalLevel enables only fatal logs
	FatalLevel
)

// Format constants
const (
	// FormatJSON specifies JSON output format
	FormatJSON = "json"
)

var (
	levelNames = map[Level]string{
		DebugLevel: "DEBUG",
		InfoLevel:  "INFO",
		WarnLevel:  "WARN",
		ErrorLevel: "ERROR",
		FatalLevel: "FATAL",
	}

	levelColors = map[Level]string{
		DebugLevel: "\033[36m", // Cyan
		InfoLevel:  "\033[32m", // Green
		WarnLevel:  "\033[33m", // Yellow
		ErrorLevel: "\033[31m", // Red
		FatalLevel: "\033[35m", // Magenta
	}

	resetColor = "\033[0m"
)

// extractContextFields extracts tenant/org/user metadata from context for automatic log enrichment.
// This enables context-aware logging where tenant, organization, and user IDs are automatically
// included in log entries without explicit field passing.
//
// Returns a map with available context fields. Missing fields are omitted (not set to empty/zero values).
func extractContextFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	// Extract tenant ID (int64)
	if tenantID, err := pkgcontext.GetTenantID(ctx); err == nil {
		fields["tenant_id"] = tenantID
	}

	// Extract organization ID (UUID)
	if orgID, err := pkgcontext.GetOrganizationID(ctx); err == nil {
		fields["organization_id"] = orgID.String()
	}

	// Extract user ID (string)
	if userID, err := pkgcontext.GetUserID(ctx); err == nil {
		fields["user_id"] = userID
	}

	return fields
}

// Logger defines the interface for logging in KiloCenter
type Logger interface {
	// Basic logging methods
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})

	// Context-aware logging methods.
	// These methods automatically extract and inject tenant/org/user metadata from context.
	DebugContext(ctx context.Context, msg string, fields ...interface{})
	InfoContext(ctx context.Context, msg string, fields ...interface{})
	WarnContext(ctx context.Context, msg string, fields ...interface{})
	ErrorContext(ctx context.Context, msg string, fields ...interface{})
	FatalContext(ctx context.Context, msg string, fields ...interface{})

	// WithField returns a new logger with the given field
	WithField(key string, value interface{}) Logger

	// WithFields returns a new logger with the given fields
	WithFields(fields map[string]interface{}) Logger
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// logger is the default implementation
type logger struct {
	mu       sync.Mutex
	level    Level
	format   string
	output   io.Writer
	fields   map[string]interface{}
	useColor bool
}

var (
	defaultLogger *logger
	once          sync.Once
)

// Initialize sets up the default logger
func Initialize(level, format string) {
	once.Do(func() {
		lvl := parseLevel(level)
		useColor := format != FormatJSON && isTerminal()

		defaultLogger = &logger{
			level:    lvl,
			format:   format,
			output:   os.Stdout,
			fields:   make(map[string]interface{}),
			useColor: useColor,
		}
	})
}

// Get returns the default logger instance
func Get() Logger {
	if defaultLogger == nil {
		Initialize("info", FormatJSON)
	}
	return defaultLogger
}

// WithField creates a new logger with the given field
func (l *logger) WithField(key string, value interface{}) Logger {
	newLogger := &logger{
		level:    l.level,
		format:   l.format,
		output:   l.output,
		fields:   copyFields(l.fields),
		useColor: l.useColor,
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields creates a new logger with the given fields
func (l *logger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &logger{
		level:    l.level,
		format:   l.format,
		output:   l.output,
		fields:   mergeFields(l.fields, fields),
		useColor: l.useColor,
	}
	return newLogger
}

// Log methods
func (l *logger) Debug(msg string, fields ...interface{}) {
	l.log(DebugLevel, msg, fields...)
}

func (l *logger) Info(msg string, fields ...interface{}) {
	l.log(InfoLevel, msg, fields...)
}

func (l *logger) Warn(msg string, fields ...interface{}) {
	l.log(WarnLevel, msg, fields...)
}

func (l *logger) Error(msg string, fields ...interface{}) {
	l.log(ErrorLevel, msg, fields...)
}

func (l *logger) Fatal(msg string, fields ...interface{}) {
	l.log(FatalLevel, msg, fields...)
	os.Exit(1)
}

// Context-aware logging methods.
// These methods automatically extract tenant/org/user from context and merge with provided fields.

func (l *logger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	l.logWithFields(DebugLevel, msg, allFields)
}

func (l *logger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	l.logWithFields(InfoLevel, msg, allFields)
}

func (l *logger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	l.logWithFields(WarnLevel, msg, allFields)
}

func (l *logger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	l.logWithFields(ErrorLevel, msg, allFields)
}

func (l *logger) FatalContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	l.logWithFields(FatalLevel, msg, allFields)
	os.Exit(1)
}

// log is the core logging function
func (l *logger) log(level Level, msg string, fields ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Combine logger fields with call fields
	allFields := mergeFields(l.fields, toMap(fields...))

	// Add standard fields
	allFields["timestamp"] = time.Now().Format(time.RFC3339)
	allFields["level"] = levelNames[level]
	allFields["message"] = msg

	// Add caller information for errors
	if level >= ErrorLevel {
		if file, line := getCaller(); file != "" {
			allFields["file"] = file
			allFields["line"] = line
		}
	}

	// Format and output
	if l.format == FormatJSON {
		l.outputJSON(allFields)
	} else {
		l.outputText(level, msg, allFields)
	}
}

// logWithFields is a helper for context-aware logging that accepts a map of fields
// instead of variadic interface{} pairs. Used by *Context methods.
func (l *logger) logWithFields(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Combine logger fields with provided fields
	allFields := mergeFields(l.fields, fields)

	// Add standard fields
	allFields["timestamp"] = time.Now().Format(time.RFC3339)
	allFields["level"] = levelNames[level]
	allFields["message"] = msg

	// Add caller information for errors
	if level >= ErrorLevel {
		if file, line := getCaller(); file != "" {
			allFields["file"] = file
			allFields["line"] = line
		}
	}

	// Format and output
	if l.format == FormatJSON {
		l.outputJSON(allFields)
	} else {
		l.outputText(level, msg, allFields)
	}
}

// outputJSON formats the log entry as JSON
func (l *logger) outputJSON(fields map[string]interface{}) {
	if data, err := json.Marshal(fields); err == nil {
		_, _ = fmt.Fprintln(l.output, string(data)) //nolint:errcheck,gosec // Logging should not crash the application
	}
}

// outputText formats the log entry as human-readable text
func (l *logger) outputText(level Level, msg string, fields map[string]interface{}) {
	// Build the log line
	var sb strings.Builder

	// Timestamp
	sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString(" ")

	// Level with color
	if l.useColor {
		sb.WriteString(levelColors[level])
	}
	sb.WriteString(fmt.Sprintf("[%-5s]", levelNames[level]))
	if l.useColor {
		sb.WriteString(resetColor)
	}
	sb.WriteString(" ")

	// Message
	sb.WriteString(msg)

	// Additional fields
	for key, value := range fields {
		if key == "timestamp" || key == "level" || key == "message" {
			continue
		}
		sb.WriteString(fmt.Sprintf(" %s=%v", key, value))
	}

	_, _ = fmt.Fprintln(l.output, sb.String()) //nolint:errcheck,gosec // Logging should not crash the application
}

// Helper functions

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

func getCaller() (string, int) {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "", 0
	}

	// Extract just the filename
	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return file, line
}

func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func copyFields(fields map[string]interface{}) map[string]interface{} {
	copied := make(map[string]interface{})
	for key, value := range fields {
		copied[key] = value
	}
	return copied
}

func mergeFields(base map[string]interface{}, override map[string]interface{}) map[string]interface{} {
	merged := copyFields(base)
	for key, value := range override {
		merged[key] = value
	}
	return merged
}

func toMap(fields ...interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	i := 0
	for i < len(fields) {
		if f, ok := fields[i].(Field); ok {
			m[f.Key] = f.Value
			i++
			continue
		}
		if i+1 >= len(fields) {
			break
		}
		key, ok := fields[i].(string)
		if !ok {
			i += 2
			continue
		}
		val := fields[i+1]
		if e, ok := val.(error); ok {
			m[key] = e.Error()
		} else {
			m[key] = val
		}
		i += 2
	}
	return m
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}
