package bssci

import (
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// The session mutation semantics of an attach live here, next to the record
// types, so every persister - whatever transaction machinery drives it -
// applies the same field mapping.

// ApplyToSession refreshes an existing active session from an attach:
// the session key, attach counter, short address and primary base station are
// replaced, and the session is marked active now.
func (rec AttachSessionRecord) ApplyToSession(s *models.EndPointSession, now time.Time, primaryBsID *int64) {
	shAddr := int32(rec.ShAddr)
	s.SessionKey = rec.EncryptedKey
	//nolint:gosec // G115: AttachCnt is handler-validated to fit 24 bits
	s.AttachCnt = int32(rec.AttachCnt)
	s.LastActivityAt = now
	s.ShAddr = &shAddr
	s.PrimaryBaseStationID = primaryBsID
}

// NewSession builds the initial session row for a first attach.
func (rec AttachSessionRecord) NewSession(now time.Time, primaryBsID *int64) *models.EndPointSession {
	shAddr := int32(rec.ShAddr)
	return &models.EndPointSession{
		TenantID:   rec.TenantID,
		EndPointID: rec.EndpointID,
		SessionID:  uuid.New().String(),
		SessionKey: rec.EncryptedKey,
		//nolint:gosec // G115: AttachCnt is handler-validated to fit 24 bits
		AttachCnt:            int32(rec.AttachCnt),
		Status:               string(models.SessionStatusActive),
		UplinkMode:           models.UplinkModeStandard,
		StartedAt:            now,
		LastActivityAt:       now,
		ShAddr:               &shAddr,
		PrimaryBaseStationID: primaryBsID,
	}
}

// ApplyToSession refreshes an existing active session from an attach
// propagate; the attach counter is owned by the origin service center and is
// not touched.
func (rec AttachPropagateSessionRecord) ApplyToSession(s *models.EndPointSession, now time.Time, primaryBsID *int64) {
	shAddr := int32(rec.ShAddr)
	s.SessionKey = rec.EncryptedKey
	s.LastActivityAt = now
	s.ShAddr = &shAddr
	s.PrimaryBaseStationID = primaryBsID
}

// NewSession builds the initial session row for a propagated attach.
func (rec AttachPropagateSessionRecord) NewSession(now time.Time, primaryBsID *int64) *models.EndPointSession {
	shAddr := int32(rec.ShAddr)
	return &models.EndPointSession{
		TenantID:             rec.TenantID,
		EndPointID:           rec.EndpointID,
		SessionID:            uuid.New().String(),
		SessionKey:           rec.EncryptedKey,
		Status:               string(models.SessionStatusActive),
		UplinkMode:           models.UplinkModeStandard,
		StartedAt:            now,
		LastActivityAt:       now,
		ShAddr:               &shAddr,
		PrimaryBaseStationID: primaryBsID,
	}
}
