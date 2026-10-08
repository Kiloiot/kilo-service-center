package builders

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// The acknowledged downlink: its service center queue id, the ref of the MQTT
// command that queued it, and the window it was transmitted in.
const (
	ackTestQueID  = 918001
	ackTestRef    = "order-918"
	ackTestWindow = 41
)

// publishedAck is one acknowledgement an MQTT publisher was handed.
type publishedAck struct {
	orgUUID, ref string
	epEUI, queID uint64
	packetCnt    uint32
}

// errBrokerDown is the publish failure of an unreachable MQTT broker.
var errBrokerDown = errors.New("broker unreachable")

// recordingMQTT stands in for the core's MQTT publisher and keeps the
// acknowledgements it publishes while its broker is reachable.
type recordingMQTT struct {
	mu   sync.Mutex
	down bool
	acks []publishedAck
}

func (*recordingMQTT) PublishUplink(context.Context, string, *mioty.ULDataMessage) error { return nil }

func (r *recordingMQTT) PublishDownlinkAcknowledged(_ context.Context, orgUUID, ref string, epEUI, queID uint64, packetCnt uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.down {
		return errBrokerDown
	}
	r.acks = append(r.acks, publishedAck{orgUUID: orgUUID, ref: ref, epEUI: epEUI, queID: queID, packetCnt: packetCnt})
	return nil
}

func (r *recordingMQTT) setDown(down bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.down = down
}

func (r *recordingMQTT) published() []publishedAck {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]publishedAck(nil), r.acks...)
}

// ackTestDatabase is a migrated database with a tenant owning the endpoint.
func ackTestDatabase(t *testing.T) (*sqlx.DB, *postgres.Repositories, int64) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping endpoint acknowledgement delivery integration test in short mode")
	}
	dsn, drop := testsupport.NewMigratedDatabase(t)
	t.Cleanup(drop)
	cfg, err := testsupport.ParseDSN(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(cfg.Port)
	require.NoError(t, err)
	db, err := postgres.New(StorageOptions(pkgconfig.StorageConfig{Host: cfg.Host, Port: port, Database: cfg.Database,
		Username: cfg.User, Password: cfg.Password, SSLMode: "disable"}), testsupport.TestCipher())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.Close()) })
	var tenantID int64
	require.NoError(t, db.Sqlx().QueryRow(`INSERT INTO tenants (name) VALUES ($1) RETURNING id`, t.Name()).Scan(&tenantID))
	_, err = db.Sqlx().Exec(`INSERT INTO endpoints (tenant_id, owner_tenant_id, ep_eui, name, description) VALUES ($1, $1, $2, $3, '')`,
		tenantID, mioty.EUI64Bytes(wakeTestEpEui), t.Name())
	require.NoError(t, err)
	return db.Sqlx(), postgres.NewRepositories(db), tenantID
}

// ackTestInfrastructure is the infrastructure the federation ingress
// assembles: MQTT enabled for the deployment, no MQTT client in the process.
func ackTestInfrastructure(repos *postgres.Repositories, tenantID int64) *Infrastructure {
	infra := deliveryTestInfrastructure(testMQTTEnabled)
	infra.Config.Protocol.DuplicateWindow = int(wakeTestWindow / time.Second)
	infra.Repos = repos
	infra.SystemEventStore = repos.SystemEvents
	infra.Clock = clock.SystemClock{}
	infra.Log = logger.NewNop()
	infra.TenantID = tenantID
	return infra
}

// seedAcknowledgeableDownlink stores the downlink an organization of the
// tenant queued through MQTT and a base station transmitted in ackTestWindow.
func seedAcknowledgeableDownlink(t *testing.T, db *sqlx.DB, tenantID int64) uuid.UUID {
	t.Helper()
	orgID := uuid.New()
	_, err := db.Exec(`INSERT INTO organizations (org_id, tenant_id, name) VALUES ($1, $2, $3)`, orgID, tenantID, orgID.String())
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, organization_id, payload, status, priority,
			transmission_packet_cnt, transmitted_at, earliest_at, latest_at, ref)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7, NOW(), NULL, NULL, $8)`,
		ackTestQueID, mioty.EUI64Bytes(wakeTestEpEui), tenantID, orgID, []byte{0x01},
		mioty.DLQueueStatusTransmitted, ackTestWindow, ackTestRef)
	require.NoError(t, err)
	return orgID
}

// pendingAcks counts the acknowledgements waiting in the outbox.
func pendingAcks(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var pending int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM message_delivery_outbox
		WHERE acknowledged_downlink_id IS NOT NULL AND status = 'pending'`).Scan(&pending))
	return pending
}

// A relayed uplink whose dlAck acknowledges a transmitted downlink: the
// federation ingress, which has no MQTT client, stores it, and the core's
// delivery worker publishes the acknowledgement once, on the MQTT topic of
// the organization that queued the downlink, with the command's ref, after
// waiting out a broker outage in the outbox.
func TestFederationIngest_TheRelayedAcknowledgementReachesMQTTThroughTheCore(t *testing.T) {
	db, repos, tenantID := ackTestDatabase(t)
	orgID := seedAcknowledgeableDownlink(t, db, tenantID)
	infra := ackTestInfrastructure(repos, tenantID)
	ctx := testutil.TestContext()

	ingest, err := BuildFederationIngestDeps(ctx, infra)
	require.NoError(t, err)
	_, err = ingest.Ingest(ctx, &bssci.UplinkPayload{
		EpEUI: wakeTestEpEui, BsEUI: wakeTestBsEui, PacketCnt: ackTestWindow + 1, UserData: []byte{0x02},
		RxTime: time.Now().UnixNano(), SNR: 12.5, RSSI: -80, DlAck: true,
	}, bssci.UplinkIngestOptions{Source: bssci.UplinkSourceFederation})
	require.NoError(t, err)

	core := &recordingMQTT{down: true}
	worker, err := buildDeliveryWorker(infra, drainedChannels(infra), discardSCACI{}, core, discardEvents{})
	require.NoError(t, err)
	worker.DrainOnce(ctx)
	require.Empty(t, core.published())
	require.Equal(t, 1, pendingAcks(t, db), "the acknowledgement waits out the outage in the outbox")

	core.setDown(false)
	_, err = db.Exec(`UPDATE message_delivery_outbox SET next_attempt_at = NOW() - INTERVAL '1 second'
		WHERE acknowledged_downlink_id IS NOT NULL`)
	require.NoError(t, err)
	worker.DrainOnce(ctx)
	worker.DrainOnce(ctx)

	assert.Equal(t, []publishedAck{{
		orgUUID: orgID.String(), ref: ackTestRef, epEUI: wakeTestEpEui, queID: ackTestQueID, packetCnt: ackTestWindow,
	}}, core.published())
	assert.Zero(t, pendingAcks(t, db))
}
