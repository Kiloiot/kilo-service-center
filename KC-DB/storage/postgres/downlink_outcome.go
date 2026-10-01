package postgres

import (
	"database/sql"
	"strconv"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/lib/pq"
)

// downlinkOutcomeColumns identify a downlink to the originators its result is
// reported to: the queue ids, the endpoint, the owner tenant, the enqueuing
// organization, the Application Center that queued it and the ref of the
// MQTT command that queued it.
const downlinkOutcomeColumns = `id, que_id, ac_que_id, ep_eui, tenant_id, organization_id, ac_eui, ref`

// sqlDownlinkInFlight holds for a downlink_queue row still in flight. The
// database function (migration 000172) is the one declaration of the
// in-flight states, which the Application Center queue id index shares.
const sqlDownlinkInFlight = `downlink_queue_in_flight(status)`

// statusArray binds queue states as a text[] parameter.
func statusArray(statuses []mioty.DLQueueStatus) interface{} {
	values := make([]string, len(statuses))
	for i, status := range statuses {
		values[i] = string(status)
	}
	return pq.Array(values)
}

// scanDownlinkOutcome reads one row of downlinkOutcomeColumns followed by
// the columns the statement returns into extra.
func scanDownlinkOutcome(row rowScanner, extra ...interface{}) (*storage.DownlinkMessage, error) {
	var downlink storage.DownlinkMessage
	var epEUI, acEUI []byte
	var tenantID int64
	var ref sql.NullString
	dest := append([]interface{}{
		&downlink.ID, &downlink.QueID, applicationQueueIDScanner{&downlink.ACQueID}, &epEUI, &tenantID, &downlink.OrganizationID, &acEUI, &ref,
	}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	downlink.EPEUI = mioty.FormatEUIBytes(epEUI)
	downlink.ACEUI = mioty.OptionalEUI64FromBytes(acEUI)
	downlink.TenantID = strconv.FormatInt(tenantID, 10)
	downlink.Ref = ref.String
	return &downlink, nil
}
