package blueprint

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// warnCapturingLogger records WarnContext messages and swallows everything else.
type warnCapturingLogger struct {
	mu    sync.Mutex
	warns []string
}

func (l *warnCapturingLogger) WarnContext(_ context.Context, msg string, _ ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warns = append(l.warns, msg)
}

func (l *warnCapturingLogger) warnings() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.warns...)
}

func (l *warnCapturingLogger) Debug(_ string, _ ...interface{})                           {}
func (l *warnCapturingLogger) Info(_ string, _ ...interface{})                            {}
func (l *warnCapturingLogger) Warn(_ string, _ ...interface{})                            {}
func (l *warnCapturingLogger) Error(_ string, _ ...interface{})                           {}
func (l *warnCapturingLogger) Fatal(_ string, _ ...interface{})                           {}
func (l *warnCapturingLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (l *warnCapturingLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (l *warnCapturingLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (l *warnCapturingLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (l *warnCapturingLogger) WithField(_ string, _ interface{}) logger.Logger            { return l }

func (l *warnCapturingLogger) WithFields(_ map[string]interface{}) logger.Logger { return l }

type typeEUIStubRepo struct {
	stubBlueprintRepo
	err error
}

func (r *typeEUIStubRepo) GetByTypeEUI(_ context.Context, _ int64, _ []byte) (*models.Blueprint, error) {
	return nil, r.err
}

func TestResolveBlueprint_NotFoundClassification(t *testing.T) {
	typeEUI := []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x00, 0x94}
	tests := []struct {
		name     string
		err      error
		wantWarn bool
	}{
		{"storage not-found sentinel is silent", storage.ErrNotFound, false},
		{"record not-found sentinel is silent", storage.ErrRecordNotFound, false},
		{"wrapped not-found is still silent", fmt.Errorf("blueprint lookup: %w", storage.ErrRecordNotFound), false},
		{"any other failure is warned about", errors.New("connection reset"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &warnCapturingLogger{}
			svc := NewResolverService(log, &typeEUIStubRepo{err: tt.err})

			bp, err := svc.ResolveBlueprint(testutil.TestContext(), 1, typeEUI, nil)

			require.NoError(t, err)
			require.Nil(t, bp)
			if tt.wantWarn {
				require.Contains(t, log.warnings(), LogBlueprintTypeEUILookupError)
				return
			}
			require.Empty(t, log.warnings(), "a not-found result must not be logged as a lookup error")
		})
	}
}
