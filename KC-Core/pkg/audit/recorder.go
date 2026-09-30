package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const errWrapRegisterDropCounter = "register audit drop counter"

// LogEventDropped is logged when an audit event could not be written.
const LogEventDropped = "audit event dropped: the action is committed but has no audit record"

// LogRequiredEventNotWritten is logged when the audit event an action needs could not be written.
const LogRequiredEventNotWritten = "audit event not written: the action it records was refused"

// Drop counter metric, exported on the service's metrics endpoint.
const (
	dropCounterName      = "kc_audit_events_dropped_total"
	dropCounterHelp      = "Audit events that could not be written, by event type."
	dropCounterEventType = "event_type"
)

// eventEmitter is the emission port a Recorder reports on; implemented by Emitter.
type eventEmitter interface {
	EmitAudit(ctx context.Context, ev Event) error
}

// DropCounter counts audit events that could not be written, by event type.
type DropCounter interface {
	Inc(eventType string)
}

// Recorder records audit events and logs and counts every one it could not
// write: Record for an action already committed, RecordRequired for one the
// caller still withholds on failure.
type Recorder struct {
	emitter eventEmitter
	log     logger.Logger
	dropped DropCounter
}

// NewRecorder returns a Recorder writing through emitter; every collaborator
// is required so a dropped event can never go unreported.
func NewRecorder(emitter eventEmitter, log logger.Logger, dropped DropCounter) (*Recorder, error) {
	switch {
	case emitter == nil:
		return nil, ErrNilEmitter
	case log == nil:
		return nil, ErrNilLogger
	case dropped == nil:
		return nil, ErrNilDropCounter
	}
	return &Recorder{emitter: emitter, log: log, dropped: dropped}, nil
}

// Record writes ev, naming the acting user of ctx unless ev names one (or,
// for a service-account key, the key in the details), and reports it as
// dropped when the write fails.
func (r *Recorder) Record(ctx context.Context, ev Event) {
	if err := r.write(ctx, ev); err != nil {
		r.reportDropped(ctx, LogEventDropped, ev, err)
	}
}

// RecordRequired writes ev like Record for an action that must not happen
// unrecorded, such as disclosing a secret, and returns the write failure so
// the caller withholds the action.
func (r *Recorder) RecordRequired(ctx context.Context, ev Event) error {
	if err := r.write(ctx, ev); err != nil {
		r.reportDropped(ctx, LogRequiredEventNotWritten, ev, err)
		return err
	}
	return nil
}

// write names the actor of ctx in ev and emits it.
func (r *Recorder) write(ctx context.Context, ev Event) error {
	if ev.UserID == "" {
		ev.UserID = ActingUser(ctx)
	}
	if ev.UserID == "" {
		ev.Details = withActingServiceAccount(ctx, ev.Details)
	}
	return r.emitter.EmitAudit(ctx, ev)
}

// reportDropped logs ev as unwritten under msg and counts it.
func (r *Recorder) reportDropped(ctx context.Context, msg string, ev Event, err error) {
	r.log.ErrorContext(ctx, msg,
		logger.FieldEventType, ev.EventType, logger.FieldTenantIDSnake, ev.TenantID, logger.FieldError, err)
	r.dropped.Inc(ev.EventType)
}

// ActingUser returns the authenticated user of ctx; empty for a call without one.
func ActingUser(ctx context.Context) string {
	userID, err := pkgcontext.GetUserID(ctx)
	if err != nil {
		return ""
	}
	if _, err := uuid.Parse(userID); err != nil {
		return ""
	}
	return userID
}

// withActingServiceAccount names the service-account key of ctx in a copy of details.
func withActingServiceAccount(ctx context.Context, details map[string]any) map[string]any {
	keyID, err := pkgcontext.GetServiceAccountID(ctx)
	if err != nil {
		return details
	}
	named := make(map[string]any, len(details)+1)
	for k, v := range details {
		named[k] = v
	}
	named[models.EventDetailKeyServiceAccount] = keyID.String()
	return named
}

// PrometheusDropCounter is the DropCounter exported as a Prometheus counter.
type PrometheusDropCounter struct {
	dropped *prometheus.CounterVec
}

// NewPrometheusDropCounter registers the drop counter with reg.
func NewPrometheusDropCounter(reg prometheus.Registerer) (*PrometheusDropCounter, error) {
	if reg == nil {
		return nil, ErrNilRegisterer
	}
	dropped := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: dropCounterName,
		Help: dropCounterHelp,
	}, []string{dropCounterEventType})
	if err := reg.Register(dropped); err != nil {
		return nil, fmt.Errorf("%s: %w", errWrapRegisterDropCounter, err)
	}
	return &PrometheusDropCounter{dropped: dropped}, nil
}

// Inc counts one dropped event of eventType.
func (c *PrometheusDropCounter) Inc(eventType string) {
	c.dropped.WithLabelValues(eventType).Inc()
}
