// Package logger provides structured logging for KiloCenter
package logger

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// Level represents the logging level
type Level int

// Logging levels from most to least verbose
const (
	// debugLevel enables debug and all higher severity logs
	debugLevel Level = iota
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
	// formatJSON specifies JSON output format
	formatJSON = "json"
)

// Level selector names accepted by Initialize/parseLevel.
const (
	levelNameDebug   = "debug"
	levelNameInfo    = "info"
	levelNameWarn    = "warn"
	levelNameWarning = "warning"
	levelNameFatal   = "fatal"
)

// Reserved JSON entry keys that field maps must not override.
const (
	entryKeyTimestamp = "timestamp"
	entryKeyLevel     = "level"
	entryKeyMessage   = "message"
)

// callerSkipFrames skips getCaller, the level method and the public wrapper
// so the caller of the logging API is reported.
const callerSkipFrames = 3

// fieldPairStride walks key/value field slices two entries at a time.
const fieldPairStride = 2

var (
	levelNames = map[Level]string{
		debugLevel: "DEBUG",
		InfoLevel:  "INFO",
		WarnLevel:  "WARN",
		ErrorLevel: "ERROR",
		FatalLevel: "FATAL",
	}

	levelColors = map[Level]string{
		debugLevel: "\033[36m", // Cyan
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
// errorFieldKey is the structured log field name for error values.
const errorFieldKey = "error"

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
		useColor := format != formatJSON && isTerminal()

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
		Initialize(levelNameInfo, formatJSON)
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
	l.log(debugLevel, msg, fields...)
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
	l.logWithFields(debugLevel, msg, allFields)
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
	if l.format == formatJSON {
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
	if l.format == formatJSON {
		l.outputJSON(allFields)
	} else {
		l.outputText(level, msg, allFields)
	}
}

// Helper functions

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case levelNameDebug:
		return debugLevel
	case levelNameInfo:
		return InfoLevel
	case levelNameWarn, levelNameWarning:
		return WarnLevel
	case errorFieldKey:
		return ErrorLevel
	case levelNameFatal:
		return FatalLevel
	default:
		return InfoLevel
	}
}

func getCaller() (string, int) {
	_, file, line, ok := runtime.Caller(callerSkipFrames)
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

// isTerminal reports whether stdout is a character device; an unreadable
// stdout is treated as not a terminal, so no color codes are written.
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
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
			i += fieldPairStride
			continue
		}
		val := fields[i+1] //nolint:gosec // G602: bounds guarded above (i+1 >= len(fields) breaks)
		if e, ok := val.(error); ok {
			m[key] = e.Error()
		} else {
			m[key] = val
		}
		i += fieldPairStride
	}
	return m
}

// Err creates an error field
func Err(err error) Field {
	if err == nil {
		return Field{Key: errorFieldKey, Value: nil}
	}
	return Field{Key: errorFieldKey, Value: err.Error()}
}
