package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var errEventStoreDown = errors.New("event store down")

// refusingEventWriter fails every lifecycle event write.
type refusingEventWriter struct{ attempts int }

func (w *refusingEventWriter) CreateEvent(context.Context, *models.SystemEvent) error {
	w.attempts++
	return errEventStoreDown
}

// warnRecorder records warning messages and discards everything else.
type warnRecorder struct {
	logger.Logger
	warnings []string
}

func (r *warnRecorder) WarnContext(_ context.Context, msg string, _ ...interface{}) {
	r.warnings = append(r.warnings, msg)
}

func TestLifecycleEvents_LogAFailedWrite(t *testing.T) {
	cfg := &config.Config{}
	events := &refusingEventWriter{}
	log := &warnRecorder{Logger: logger.NewNop()}

	emitStartedEvent(testutil.TestContext(), log, events, cfg, testLifecycleHost)
	emitStoppedEvent(log, events, cfg, testLifecycleHost)

	assert.Equal(t, 2, events.attempts)
	assert.Equal(t, []string{LogGatewayLifecycleEventFailed, LogGatewayLifecycleEventFailed}, log.warnings)
}

const testLifecycleHost = "gateway-host"
