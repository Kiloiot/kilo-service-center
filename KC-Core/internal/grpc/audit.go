package grpc

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
)

// AuditRecorder records the audit trail of operator actions, naming the acting
// user, and reports an event it could not write; implemented by audit.Recorder.
type AuditRecorder interface {
	Record(ctx context.Context, ev audit.Event)
}

// RequiredAuditRecorder records the audit event an action cannot happen
// without, such as disclosing a secret, and returns the write failure so the
// action is refused; implemented by audit.Recorder.
type RequiredAuditRecorder interface {
	RecordRequired(ctx context.Context, ev audit.Event) error
}
