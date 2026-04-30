// Package logger testing helpers.
// FromZap is provided for tests that need to capture log output via zap's observer.
// Production code should use logger.Get() or logger.NewNop() instead.
package logger

import (
	"context"

	"go.uber.org/zap"
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

// zapAdapter wraps *zap.Logger to implement Logger interface.
// Used exclusively in tests that need to capture log output via zap's observer.
type zapAdapter struct {
	zap *zap.Logger
}

func (z *zapAdapter) Debug(msg string, fields ...interface{}) {
	z.zap.Sugar().Debugw(msg, expandZapFields(fields)...)
}

func (z *zapAdapter) Info(msg string, fields ...interface{}) {
	z.zap.Sugar().Infow(msg, expandZapFields(fields)...)
}

func (z *zapAdapter) Warn(msg string, fields ...interface{}) {
	z.zap.Sugar().Warnw(msg, expandZapFields(fields)...)
}

func (z *zapAdapter) Error(msg string, fields ...interface{}) {
	z.zap.Sugar().Errorw(msg, expandZapFields(fields)...)
}

func (z *zapAdapter) Fatal(msg string, fields ...interface{}) {
	z.zap.Sugar().Fatalw(msg, expandZapFields(fields)...)
}

func (z *zapAdapter) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	zapFields := mapToZapSlice(allFields)
	z.zap.Sugar().Debugw(msg, zapFields...)
}

func (z *zapAdapter) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	zapFields := mapToZapSlice(allFields)
	z.zap.Sugar().Infow(msg, zapFields...)
}

func (z *zapAdapter) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	zapFields := mapToZapSlice(allFields)
	z.zap.Sugar().Warnw(msg, zapFields...)
}

func (z *zapAdapter) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	zapFields := mapToZapSlice(allFields)
	z.zap.Sugar().Errorw(msg, zapFields...)
}

func (z *zapAdapter) FatalContext(ctx context.Context, msg string, fields ...interface{}) {
	contextFields := extractContextFields(ctx)
	allFields := mergeFields(contextFields, toMap(fields...))
	zapFields := mapToZapSlice(allFields)
	z.zap.Sugar().Fatalw(msg, zapFields...)
}

func (z *zapAdapter) WithField(key string, value interface{}) Logger {
	return &zapAdapter{zap: z.zap.With(zap.Any(key, value))}
}

func (z *zapAdapter) WithFields(fields map[string]interface{}) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &zapAdapter{zap: z.zap.With(zapFields...)}
}

// FromZap wraps a *zap.Logger to implement the Logger interface.
// This exists for tests that use zap's observer to capture and assert on log output.
// Production code should use logger.Get() instead.
func FromZap(zapLogger *zap.Logger) Logger {
	return &zapAdapter{zap: zapLogger}
}

// expandZapFields expands Field structs into key-value pairs for zap's SugaredLogger.
func expandZapFields(fields []interface{}) []interface{} {
	result := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		if field, ok := f.(Field); ok {
			result = append(result, field.Key, field.Value)
			continue
		}
		result = append(result, f)
	}
	return result
}

// mapToZapSlice converts a map to a slice of alternating keys and values for zap.
func mapToZapSlice(m map[string]interface{}) []interface{} {
	result := make([]interface{}, 0, len(m)*2)
	for k, v := range m {
		result = append(result, k, v)
	}
	return result
}
