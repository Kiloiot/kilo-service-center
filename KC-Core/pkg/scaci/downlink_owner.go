package scaci

import (
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// downlinkRef names the Application Center's downlink of the endpoint by its
// service center queue id; another organization's reads as not found.
func (a ApplicationCenter) downlinkRef(epEUI, queID uint64) scheduler.DownlinkRef {
	return scheduler.DownlinkRef{TenantID: a.TenantID, QueID: queID, OrganizationID: a.organization(), EpEUI: &epEUI}
}

// counterDownlinks names the Application Center's downlinks of the endpoint
// scheduled for the packet counter (SCACI §3.11.1).
func (a ApplicationCenter) counterDownlinks(epEUI uint64, packetCnt uint32) storage.PacketCounterDownlinks {
	return storage.PacketCounterDownlinks{TenantID: a.TenantID, OrganizationID: a.organization(), EpEUI: epEUI, PacketCnt: packetCnt}
}

// endpointQueue narrows the in-flight queue to the Application Center's
// downlinks of the endpoint.
func (a ApplicationCenter) endpointQueue(epEUI uint64) storage.DownlinkQueueFilter {
	endpoint := [8]byte(mioty.EUI64Bytes(epEUI))
	return storage.DownlinkQueueFilter{EpEUI: &endpoint, OrganizationID: a.organization()}
}
