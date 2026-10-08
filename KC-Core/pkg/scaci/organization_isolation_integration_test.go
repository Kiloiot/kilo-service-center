package scaci_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Client certificate common names of the two organizations of harnessTenant.
const (
	ownerOrgCN    = "owner-organization"
	claimantOrgCN = "claimant-organization"
	isolationPing = int64(7)
)

// organizationResolver resolves each client certificate to its organization
// of harnessTenant; any other certificate falls back like fallbackResolver.
type organizationResolver struct {
	fallbackResolver
	byCN map[string]uuid.UUID
}

func (r organizationResolver) ResolveCert(ctx context.Context, cert *x509.Certificate) (uuid.UUID, int64, error) {
	if orgID, ok := r.byCN[cert.Subject.CommonName]; ok {
		return orgID, harnessTenant, nil
	}
	return r.fallbackResolver.ResolveCert(ctx, cert)
}

// organizationHarness serves two organizations of harnessTenant, each with
// its own client certificate.
type organizationHarness struct {
	*scaciHarness
	owner, claimant      uuid.UUID
	ownerCert, claimCert tls.Certificate
}

func newOrganizationHarness(t *testing.T) *organizationHarness {
	t.Helper()
	owner, claimant := uuid.New(), uuid.New()
	h := newSCACIHarnessWith(t, scaci.Config{}, organizationResolver{byCN: map[string]uuid.UUID{ownerOrgCN: owner, claimantOrgCN: claimant}})
	for _, orgID := range []uuid.UUID{owner, claimant} {
		_, err := h.db.Exec(testutil.TestContext(), `INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`,
			orgID, harnessTenant, orgID.String())
		require.NoError(t, err)
	}
	return &organizationHarness{scaciHarness: h, owner: owner, claimant: claimant,
		ownerCert: h.pki.issueClient(ownerOrgCN), claimCert: h.pki.issueClient(claimantOrgCN)}
}

// statusOf reads the status of the organization's harness application
// center's session of snAcUUID.
func (h *organizationHarness) statusOf(t *testing.T, orgID uuid.UUID, snAcUUID scaci.UUID16) string {
	t.Helper()
	ac := scaci.ApplicationCenter{TenantID: harnessTenant, OrganizationID: orgID, AcEui: harnessAcEui}
	session, err := h.sessions.GetSessionByAcUUID(testutil.TestContext(), ac.Key(), snAcUUID)
	require.NoError(t, err)
	return session.Status
}

func (h *organizationHarness) waitForStatusOf(t *testing.T, orgID uuid.UUID, snAcUUID scaci.UUID16, status string) {
	t.Helper()
	require.Eventually(t, func() bool { return h.statusOf(t, orgID, snAcUUID) == status },
		harnessWaitTimeout, harnessWaitStep, "the session of %s never became %s", orgID, status)
}

// Two organizations of one tenant each keep a live session of an Application
// Center with the same acEui (SCACI §1: an Application Center is an acEui of
// one organization). The other organization connecting neither fails nor
// takes the owner's connection, and the owner resumes its session while the
// other organization's is live, even when that one presented the owner's
// snAcUuid.
func TestSCACIServer_OrganizationsKeepTheirOwnSessionOfAnAcEui(t *testing.T) {
	h := newOrganizationHarness(t)
	snAcUUID := harnessUUID(harnessUUIDFirst)
	owner := h.dialAs(t, h.ownerCert, harnessAcEui)
	owner.connect(snAcUUID)
	h.waitForStatusOf(t, h.owner, snAcUUID, models.SCACISessionStatusActive)

	claimant := h.dialAs(t, h.claimCert, harnessAcEui)
	require.Equal(t, false, claimant.connect(snAcUUID)["snResume"], "the claimant starts a session of its own")
	h.waitForStatusOf(t, h.claimant, snAcUUID, models.SCACISessionStatusActive)
	assert.Equal(t, scaci.CmdPingResponse, owner.ping(isolationPing)["command"], "the owner keeps its connection")
	assert.Equal(t, models.SCACISessionStatusActive, h.statusOf(t, h.owner, snAcUUID))

	require.NoError(t, owner.conn.Close())
	h.waitForStatusOf(t, h.owner, snAcUUID, models.SCACISessionStatusDisconnected)
	resumed := h.dialAs(t, h.ownerCert, harnessAcEui)
	require.Equal(t, true, resumed.resume(snAcUUID, 0, 0)["snResume"], "the owner resumes its own session")
	h.waitForStatusOf(t, h.owner, snAcUUID, models.SCACISessionStatusActive)
	assert.Equal(t, scaci.CmdPingResponse, resumed.ping(isolationPing + 1)["command"])
	assert.Equal(t, scaci.CmdPingResponse, claimant.ping(isolationPing + 1)["command"], "the claimant keeps its own session")
	assert.Equal(t, models.SCACISessionStatusActive, h.statusOf(t, h.claimant, snAcUUID))
}
