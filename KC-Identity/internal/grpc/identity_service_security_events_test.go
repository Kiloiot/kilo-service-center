package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

var errEventStoreDown = errors.New("event store down")

// refusingEventWriter fails every event write.
type refusingEventWriter struct{}

func (refusingEventWriter) CreateEvent(context.Context, *models.SystemEvent) error {
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

func TestEmitSecurityEvent_LogsAFailedWrite(t *testing.T) {
	log := &warnRecorder{Logger: logger.NewNop()}
	svc := &IdentityService{eventWriter: refusingEventWriter{}, log: log}

	svc.emitSecurityEvent(testutil.TestContext(), models.EventTypeAuthPermissionDenied,
		models.EventTitleAuthPermissionDenied, opNameRequireAdmin, detailNonAdminAttemptedAdminOp)

	assert.Equal(t, []string{LogSecurityEventWriteFailed}, log.warnings)
}
