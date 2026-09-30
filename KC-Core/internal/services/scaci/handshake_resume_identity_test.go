package scaciservices

import (
	"context"
	"crypto/x509"
	"encoding/binary"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	resumeIdentityTenant    = int64(7)
	resumeIdentitySessionID = int64(99)
	resumeIdentityAcEui     = uint64(0xAABBCCDDEEFF1122)
	resumeIdentityOtherAc   = uint64(0x0102030405060708)
	resumeIdentityAcOpID    = int64(500)
	resumeIdentityScOpID    = int64(-900)
	// resumeIdentityNotFound is why the store resumes no session of another
	// application center.
	resumeIdentityNotFound = "session not found"
)

var (
	resumeIdentityOrg      = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	resumeIdentityOtherOrg = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	resumeIdentitySnAcUUID = [16]byte{0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B, 0x3C, 0x3D, 0x3E, 0x3F, 0x40}
)

// connectPresentingStoredSession connects with a certificate of certOrg in
// the stored session's tenant, as the application center acEui, presenting
// the snAcUuid of a disconnected session of resumeIdentityOrg's application
// center resumeIdentityAcEui.
func connectPresentingStoredSession(t *testing.T, certOrg uuid.UUID, acEui uint64) (*scaci.Session, *scaci.ConnectResponse) {
	t.Helper()
	var storedAcEUI [8]byte
	binary.BigEndian.PutUint64(storedAcEUI[:], resumeIdentityAcEui)
	storedOrg := resumeIdentityOrg
	owns := func(ac models.SCACIApplicationCenter) bool {
		return ac.TenantID == resumeIdentityTenant && ac.OrganizationID != nil && *ac.OrganizationID == storedOrg && ac.AcEUI == storedAcEUI
	}
	sessions := &mockSCACISessionRepository{
		checkSessionResumableFunc: func(_ context.Context, ac models.SCACIApplicationCenter, _ [16]byte) (*models.SCACISessionResumptionInfo, error) {
			if !owns(ac) {
				return &models.SCACISessionResumptionInfo{ReasonIfNotResumable: resumeIdentityNotFound}, nil
			}
			return &models.SCACISessionResumptionInfo{
				SessionID: resumeIdentitySessionID, CanResume: true,
				LastKnownAcOpId: resumeIdentityAcOpID, LastKnownScOpId: resumeIdentityScOpID,
				NegotiatedVersion: scaci.ProtocolVersionString,
			}, nil
		},
		getSessionByAcUUIDFunc: func(_ context.Context, ac models.SCACIApplicationCenter, _ [16]byte) (*models.SCACISession, error) {
			if !owns(ac) {
				return nil, storage.ErrNotFound
			}
			return &models.SCACISession{
				ID: resumeIdentitySessionID, TenantID: ac.TenantID, AcEUI: storedAcEUI, SnAcUUID: resumeIdentitySnAcUUID,
				LastOpIDAc: resumeIdentityAcOpID, LastOpIDSc: resumeIdentityScOpID, Status: "disconnected",
				OrganizationID: &storedOrg, ConnectedAt: time.Now().Add(-time.Hour),
			}, nil
		},
	}
	certificates := &mockOrgResolver{
		resolveCertFunc: func(context.Context, *x509.Certificate) (uuid.UUID, int64, error) {
			return certOrg, resumeIdentityTenant, nil
		},
	}
	svc := NewHandshakeService(sessions, logger.NewNop(), certificates, resumeIdentityTenant, true, &mockCertificateVerifier{},
		0x1122334455667788, "KiloCenter", "KC-1000", "test-sc", scaci.ProtocolVersionString, scaci.NewSessionFactory(clock.SystemClock{}))

	acOpID, scOpID := resumeIdentityAcOpID, resumeIdentityScOpID
	session, resp, errToken := svc.ValidateConnect(testutil.TestContext(), &scaci.Connect{
		Version:  scaci.ProtocolVersionString,
		AcEui:    acEui,
		SnAcUUID: resumeIdentitySnAcUUID,
		SnAcOpId: &acOpID,
		SnScOpId: &scOpID,
	}, createValidTestCert("org-"+certOrg.String()))
	require.Empty(t, errToken)
	require.NotNil(t, session)
	require.NotNil(t, resp)
	return session, resp
}

// SCACI §1, §3.3.1: a session is resumed only by the application center it
// belongs to. An application center of another organization of the same
// tenant that presents the session's snAcUuid starts a new session of its
// own organization; it neither takes the session over nor evicts it.
func TestResumeFromAnotherOrganizationOfTheTenantStartsANewSession(t *testing.T) {
	session, resp := connectPresentingStoredSession(t, resumeIdentityOtherOrg, resumeIdentityAcEui)

	assert.False(t, resp.SnResume, "the stored session is not resumed")
	assert.False(t, session.Resumed)
	assert.NotEqual(t, resumeIdentitySessionID, session.ID, "the new session is not the stored one")
	assert.Equal(t, resumeIdentityOtherOrg, session.OrganizationID, "the new session runs under the certificate's organization")
}

func TestResumeByAnotherApplicationCenterStartsANewSession(t *testing.T) {
	session, resp := connectPresentingStoredSession(t, resumeIdentityOrg, resumeIdentityOtherAc)

	assert.False(t, resp.SnResume, "the stored session is not resumed")
	assert.False(t, session.Resumed)
	assert.NotEqual(t, resumeIdentitySessionID, session.ID, "the new session is not the stored one")
	assert.Equal(t, resumeIdentityOtherAc, session.AcEui)
}

func TestResumeByTheSessionsApplicationCenterResumesIt(t *testing.T) {
	session, resp := connectPresentingStoredSession(t, resumeIdentityOrg, resumeIdentityAcEui)

	assert.True(t, resp.SnResume)
	assert.True(t, session.Resumed)
	assert.Equal(t, resumeIdentitySessionID, session.ID)
	assert.Equal(t, resumeIdentityOrg, session.OrganizationID)
}
