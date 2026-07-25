package testutil

import (
	"context"
	"sync"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// RecordedEntry is one captured log call.
type RecordedEntry struct {
	Level   string
	Message string
	Fields  []interface{}
}

// levelRank orders levels for AllAtLeast filtering.
var levelRank = map[string]int{
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
	"FATAL": 4,
}

// RecordingLogger is a logger.Logger implementation that captures every call
// so tests can assert on emitted log messages without importing zap.
type RecordingLogger struct {
	mu      sync.Mutex
	entries []RecordedEntry
}

// NewRecordingLogger creates an empty recorder.
func NewRecordingLogger() *RecordingLogger {
	return &RecordingLogger{}
}

func (r *RecordingLogger) record(level, msg string, fields []interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, RecordedEntry{Level: level, Message: msg, Fields: fields})
}

// Debug records a DEBUG entry.
func (r *RecordingLogger) Debug(msg string, fields ...interface{}) { r.record("DEBUG", msg, fields) }

// Info records an INFO entry.
func (r *RecordingLogger) Info(msg string, fields ...interface{}) { r.record("INFO", msg, fields) }

// Warn records a WARN entry.
func (r *RecordingLogger) Warn(msg string, fields ...interface{}) { r.record("WARN", msg, fields) }

// Error records an ERROR entry.
func (r *RecordingLogger) Error(msg string, fields ...interface{}) { r.record("ERROR", msg, fields) }

// Fatal records a FATAL entry (never exits).
func (r *RecordingLogger) Fatal(msg string, fields ...interface{}) { r.record("FATAL", msg, fields) }

// DebugContext records a DEBUG entry.
func (r *RecordingLogger) DebugContext(_ context.Context, msg string, fields ...interface{}) {
	r.record("DEBUG", msg, fields)
}

// InfoContext records an INFO entry.
func (r *RecordingLogger) InfoContext(_ context.Context, msg string, fields ...interface{}) {
	r.record("INFO", msg, fields)
}

// WarnContext records a WARN entry.
func (r *RecordingLogger) WarnContext(_ context.Context, msg string, fields ...interface{}) {
	r.record("WARN", msg, fields)
}

// ErrorContext records an ERROR entry.
func (r *RecordingLogger) ErrorContext(_ context.Context, msg string, fields ...interface{}) {
	r.record("ERROR", msg, fields)
}

// FatalContext records a FATAL entry (never exits).
func (r *RecordingLogger) FatalContext(_ context.Context, msg string, fields ...interface{}) {
	r.record("FATAL", msg, fields)
}

// WithField returns the same recorder (field scoping is not recorded).
func (r *RecordingLogger) WithField(_ string, _ interface{}) logger.Logger { return r }

// WithFields returns the same recorder (field scoping is not recorded).
func (r *RecordingLogger) WithFields(_ map[string]interface{}) logger.Logger { return r }

// All returns every captured entry.
func (r *RecordingLogger) All() []RecordedEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RecordedEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

// AllAtLeast returns entries at or above the given level (DEBUG < INFO <
// WARN < ERROR < FATAL), mirroring a level-filtered observer.
func (r *RecordingLogger) AllAtLeast(level string) []RecordedEntry {
	minRank := levelRank[level]
	out := []RecordedEntry{}
	for _, e := range r.All() {
		if levelRank[e.Level] >= minRank {
			out = append(out, e)
		}
	}
	return out
}

// FilterMessage returns entries whose message equals msg.
func (r *RecordingLogger) FilterMessage(msg string) []RecordedEntry {
	out := []RecordedEntry{}
	for _, e := range r.All() {
		if e.Message == msg {
			out = append(out, e)
		}
	}
	return out
}

// interface guard
var _ logger.Logger = (*RecordingLogger)(nil)

// FieldMap converts the entry's variadic key-value fields into a map
// (odd-positioned strings are keys). Non-string keys are skipped.
func (e RecordedEntry) FieldMap() map[string]interface{} {
	out := make(map[string]interface{})
	for i := 0; i+1 < len(e.Fields); i += 2 {
		if key, ok := e.Fields[i].(string); ok {
			out[key] = e.Fields[i+1]
		}
	}
	return out
}
