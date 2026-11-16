package bssciservices

import (
	"encoding/binary"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	tenantScopeTestTenant   = int64(9)
	tenantScopeTestEUI      = "70B3D56770111509"
	tenantScopeTestEUIValue = uint64(0x70B3D56770111509)
)

// tenantRecordingStations records the tenant each propagation and station
// listing was scoped to.
type tenantRecordingStations struct {
	mu       sync.Mutex
	attaches []int64
	detaches []int64
	listings []int64
}

func (s *tenantRecordingStations) SendAttachPropagateToAll(tenantID int64, _ uint64, _ []byte, _ uint16, _ bool, _ uint32, _ bool, _ uint8, _ bool, _ bool) []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attaches = append(s.attaches, tenantID)
	return nil
}

func (s *tenantRecordingStations) SendDetachPropagateToAll(tenantID int64, _ uint64) []error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.detaches = append(s.detaches, tenantID)
	return nil
}

func (s *tenantRecordingStations) GetConnectedSessionEUIs(tenantID int64) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listings = append(s.listings, tenantID)
	return nil
}

func newTenantScopeFixture(t *testing.T) (*attachmentFixture, *tenantRecordingStations) {
	t.Helper()
	endpoint := &models.EndPoint{
		ID:       91,
		TenantID: tenantScopeTestTenant,
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}
	binary.BigEndian.PutUint64(endpoint.EUI[:], tenantScopeTestEUIValue)
	stations := &tenantRecordingStations{}
	f := newAttachmentFixture(t, sessionKeyEndpointLookup{endpoint: endpoint}, noEndpointCreation{}, stations, fixedSessionKeys{}, discardedEvents{})
	return f, stations
}

// An attach from the API is propagated, and its events list stations, only
// for the tenant that owns the endpoint.
func TestAttachEndPointPropagatesToTheEndpointTenantOnly(t *testing.T) {
	f, stations := newTenantScopeFixture(t)

	_, err := f.svc.AttachEndPoint(testutil.TestContext(), tenantScopeTestEUI, tenantScopeTestTenant)
	require.NoError(t, err)
	f.wait(t)

	assert.Equal(t, []int64{tenantScopeTestTenant}, stations.attaches)
	assert.Equal(t, []int64{tenantScopeTestTenant}, stations.listings)
}

func TestDetachEndPointPropagatesToTheEndpointTenantOnly(t *testing.T) {
	f, stations := newTenantScopeFixture(t)

	_, err := f.svc.DetachEndPoint(testutil.TestContext(), tenantScopeTestEUI, tenantScopeTestTenant)
	require.NoError(t, err)
	f.wait(t)

	assert.Equal(t, []int64{tenantScopeTestTenant}, stations.detaches)
	assert.Equal(t, []int64{tenantScopeTestTenant}, stations.listings)
}
