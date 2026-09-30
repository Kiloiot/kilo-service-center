package bssci

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	completionStationEUI  = uint64(0x70B3D59CD00009E6)
	completionEndpointEUI = uint64(0x70B3D56770111505)
	completionOpID        = int64(-1770)
	completionOwnTenant   = int64(1)
	completionOtherTenant = int64(4)
)

func completionSession() *Session {
	return &Session{ProtocolSessionState: ProtocolSessionState{BaseStationEUI: completionStationEUI, ResolvedTenantID: completionOwnTenant}}
}

func completionOp(owner int64) *PendingOperation {
	return &PendingOperation{Metadata: map[string]interface{}{
		models.EventDetailKeyEpEui: completionEndpointEUI,
		metadataKeyTenantID:        owner,
	}}
}

// The station's tenant learns which of its endpoints an attach propagate
// completed for; a roaming endpoint of another tenant stays unnamed.
func TestAttachPropagateCompletion_NamesTheStationTenantsEndpoint(t *testing.T) {
	s := &Server{tenantID: completionOwnTenant}

	assert.Equal(t, "Attach propagate of endpoint 70B3D56770111505 to base station 70B3D59CD00009E6 completed (opId -1770)",
		s.attachPropagateCompletion(completionSession(), completionOpID, completionOp(completionOwnTenant)))
	assert.Equal(t, "Attach propagate to base station 70B3D59CD00009E6 completed (opId -1770)",
		s.attachPropagateCompletion(completionSession(), completionOpID, completionOp(completionOtherTenant)))
	assert.Equal(t, "Attach propagate to base station 70B3D59CD00009E6 completed (opId -1770)",
		s.attachPropagateCompletion(completionSession(), completionOpID, nil))
}

// Short addresses are shown in hex, the form the Add End Point form takes.
func TestKeysPropagatedDescription_ShortAddressInHex(t *testing.T) {
	assert.Contains(t, eventDescFmtKeysPropagated, "%04X")
}
