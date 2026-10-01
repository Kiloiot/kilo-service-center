// Package logger testing helpers.
// Production code should use logger.Get(); tests that do not assert on log
// output should use logger.NewNop().
package logger

import (
	"context"
)

// nopLogger is a logger that discards all output. Useful in tests.
type nopLogger struct{}

func (n *nopLogger) Debug(_ string, _ ...interface{})                           {}
func (n *nopLogger) Info(_ string, _ ...interface{})                            {}
func (n *nopLogger) Warn(_ string, _ ...interface{})                            {}
func (n *nopLogger) Error(_ string, _ ...interface{})                           {}
func (n *nopLogger) Fatal(_ string, _ ...interface{})                           {}
func (n *nopLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (n *nopLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (n *nopLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (n *nopLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (n *nopLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (n *nopLogger) WithField(_ string, _ interface{}) Logger                   { return n }
func (n *nopLogger) WithFields(_ map[string]interface{}) Logger                 { return n }

// NewNop returns a logger that silently discards all output.
// Intended for tests where log output is irrelevant.
func NewNop() Logger {
	return &nopLogger{}
}
