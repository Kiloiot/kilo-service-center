package scaci

import (
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	ownerScopeTenant    = int64(4)
	ownerScopeEndpoint  = uint64(0x1234567890ABCDEF)
	ownerScopeQueueID   = uint64(4200042)
	ownerScopePacketCnt = uint32(100)
)

// organizationSession is an active session of an Application Center of one
// organization in a tenant with several.
func organizationSession(orgID uuid.UUID) *Session {
	return &Session{TenantID: ownerScopeTenant, OrganizationID: orgID, State: StateActive}
}

// organizationRef is the revoke reference an organization's Application
// Center may issue: its tenant, organization and endpoint.
func organizationRef(orgID uuid.UUID, epEUI, queID uint64) scheduler.DownlinkRef {
	return scheduler.DownlinkRef{TenantID: ownerScopeTenant, QueID: queID, OrganizationID: &orgID, EpEUI: &epEUI}
}

// TestHandleDLDataRevoke_ScopedToTheSessionOrganization: a dlDataRev of an
// organization's Application Center looks up and revokes only that
// organization's downlinks, so another organization of the tenant never
// reaches them.
func TestHandleDLDataRevoke_ScopedToTheSessionOrganization(t *testing.T) {
	orgID := uuid.New()
	downlinks := new(MockDLService)
	downlinks.On("GetDownlinksByPacketCnt", mock.Anything, storage.PacketCounterDownlinks{
		TenantID: ownerScopeTenant, OrganizationID: &orgID, EpEUI: ownerScopeEndpoint, PacketCnt: ownerScopePacketCnt,
	}).Return([]*storage.DownlinkMessage{{QueID: int64(ownerScopeQueueID)}}, nil)
	downlinks.On("RevokeDownlink", mock.Anything, organizationRef(orgID, ownerScopeEndpoint, ownerScopeQueueID)).Return(uint64(0), "")
	conn := &mockConn{}

	require.NoError(t, revokeHandlerServer(downlinks, nil, nil).handleDLDataRevoke(conn, organizationSession(orgID), 14, revokePayload(t, 14, ownerScopePacketCnt)))

	var resp DLDataRevokeResponse
	require.NoError(t, decodeResponse(conn.written, &resp))
	assert.Equal(t, CmdDLDataRevokeResponse, resp.Command)
	downlinks.AssertExpectations(t)
}

// TestHandleDLDataRevoke_ForeignDownlinkReadsAsNotFound: a downlink the
// organization-scoped revoke does not reach answers the Application Center
// with the unknown-downlink error (SCACI §3.14).
func TestHandleDLDataRevoke_ForeignDownlinkReadsAsNotFound(t *testing.T) {
	orgID := uuid.New()
	downlinks := new(MockDLService)
	downlinks.On("GetDownlinksByPacketCnt", mock.Anything, mock.Anything).
		Return([]*storage.DownlinkMessage{{QueID: int64(ownerScopeQueueID)}}, nil)
	downlinks.On("RevokeDownlink", mock.Anything, organizationRef(orgID, ownerScopeEndpoint, ownerScopeQueueID)).Return(uint64(0), ErrDownlinkNotFound)
	conn := &mockConn{}

	require.NoError(t, revokeHandlerServer(downlinks, nil, nil).handleDLDataRevoke(conn, organizationSession(orgID), 15, revokePayload(t, 15, ownerScopePacketCnt)))

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))
	assert.Equal(t, POSIX_ENOENT, errorResp.Code)
	assertErrorToken(t, errorResp, errDownlinkNotFound)
}

// TestRevokeEndpointDownlinks_ScopedToTheSessionOrganization: completing a
// deregistration revokes only the downlinks the Application Center's
// organization queued for the endpoint.
func TestRevokeEndpointDownlinks_ScopedToTheSessionOrganization(t *testing.T) {
	orgID := uuid.New()
	endpoint := [8]byte(mioty.EUI64Bytes(ownerScopeEndpoint))
	downlinks := new(MockDLService)
	downlinks.On("GetDownlinkQueue", mock.Anything, ownerScopeTenant, storage.DownlinkQueueFilter{EpEUI: &endpoint, OrganizationID: &orgID}).
		Return([]*storage.DownlinkMessage{{QueID: int64(ownerScopeQueueID)}}, nil)
	downlinks.On("RevokeDownlink", mock.Anything, organizationRef(orgID, ownerScopeEndpoint, ownerScopeQueueID)).Return(uint64(0), "")
	server := &Server{
		registry: newTestRegistry(make(map[net.Conn]*Session), nil),
		clock:    clock.SystemClock{},
		logger:   testLogger(),
		dlSvc:    downlinks,
	}

	revoked, err := server.revokeEndpointDownlinks(t.Context(), organizationSession(orgID).ApplicationCenter(), ownerScopeEndpoint)

	require.NoError(t, err)
	assert.Equal(t, 1, revoked)
	downlinks.AssertExpectations(t)
}
