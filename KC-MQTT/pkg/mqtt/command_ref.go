package mqtt

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// admit resolves the tenant of a command and tells whether it may be
// answered: a ref the organization already queued for the endpoint is not
// answered again, whatever else is wrong with the repeat, so a body that did
// not decode is refused only after its ref proved new.
func (h *CommandHandler) admit(ctx context.Context, target commandTarget, ref string, decodeErr error) (int64, bool) {
	tenantID, err := h.lookup.LookupTenant(ctx, target.org)
	if err != nil {
		h.unresolvedOrganization(ctx, target, ref, err)
		return 0, false
	}
	if !h.firstReception(ctx, target, tenantID, ref) {
		return 0, false
	}
	if decodeErr != nil {
		h.reject(ctx, target, ref, decodeErr)
		return 0, false
	}
	return tenantID, true
}

// unresolvedOrganization refuses a command whose organization does not
// exist. A command with a ref whose organization could not be looked up is
// not answered, since a refusal could contradict an earlier acceptance of it.
func (h *CommandHandler) unresolvedOrganization(ctx context.Context, target commandTarget, ref string, err error) {
	if ref != "" && !errors.Is(err, storage.ErrNotFound) {
		h.logger.WarnContext(ctx, LogCommandOrgLookupFailed,
			logger.FieldOrgUUID, target.org.String(), logger.FieldEpEui, target.epEUIHex,
			logger.FieldRef, logger.UntrustedValue(ref, storage.MaxDownlinkRefBytes), logger.FieldError, err)
		return
	}
	h.reject(ctx, target, ref, fmt.Errorf("%w: %w", refusalOrgUnresolved, err))
}

// firstReception tells whether a command may be answered: one without a ref,
// or whose ref the organization has not queued for the endpoint yet. A ref
// that cannot be looked up is not answered either, since a refusal could
// contradict an earlier acceptance of the same command.
func (h *CommandHandler) firstReception(ctx context.Context, target commandTarget, tenantID int64, ref string) bool {
	if ref == "" {
		return true
	}
	queued, err := h.enqueuer.CommandQueued(ctx, storage.DownlinkCommandRef{
		TenantID: tenantID, OrganizationID: target.org, EpEUI: target.epEUI, Ref: ref,
	})
	if err != nil {
		h.logger.WarnContext(ctx, LogCommandRefLookupFailed,
			logger.FieldEpEui, target.epEUIHex, logger.FieldRef, logger.UntrustedValue(ref, storage.MaxDownlinkRefBytes), logger.FieldError, err)
		return false
	}
	if queued {
		h.logRepeat(ctx, target, ref)
	}
	return !queued
}

// logRepeat logs a command that repeats a ref already queued; it is not answered.
func (h *CommandHandler) logRepeat(ctx context.Context, target commandTarget, ref string) {
	h.logger.InfoContext(ctx, LogCommandAlreadyQueued,
		logger.FieldEpEui, target.epEUIHex, logger.FieldRef, logger.UntrustedValue(ref, storage.MaxDownlinkRefBytes))
}
