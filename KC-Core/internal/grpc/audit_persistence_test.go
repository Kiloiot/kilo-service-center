package grpc

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	persistTenantA        int64 = 7101
	persistTenantB        int64 = 7102
	persistPlatformTenant int64 = 1
	persistEpEUI                = "70B3D5677011A001"
	persistEpEUIDashed          = "70-B3-D5-67-70-11-A0-01"
	persistBsEUI                = "70B3D59CD000A002"
	persistBsEUIDashed          = "70-B3-D5-9C-D0-00-A0-02"
	persistBsName               = "audit-persistence-station"
	persistListLimit            = 50
)

// persistedEvent is one system_events row as the audit trail stores it.
type persistedEvent struct {
	TenantID      sql.NullInt64 `db:"tenant_id"`
	EndpointID    sql.NullInt64 `db:"endpoint_id"`
	BasestationID sql.NullInt64 `db:"basestation_id"`
	SourceName    string        `db:"source_name"`
	UserID        string        `db:"user_id"`
	Data          []byte        `db:"data"`
}

func (e persistedEvent) details(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(e.Data, &out))
	return out
}

// eventRows reads audit rows straight from the table.
type eventRows interface {
	Select(dest any, query string, args ...any) error
}

// eventLister lists a tenant's events as the activity feed does.
type eventLister interface {
	GetEvents(ctx context.Context, filter models.SystemEventFilter) ([]*models.SystemEvent, error)
}

// auditFixture wires the handlers to the real repositories and the real
// audit recorder over a migrated PostgreSQL database.
type auditFixture struct {
	rows      eventRows
	endpoints grpcservices.EndpointStore
	stations  grpcservices.BaseStationStore
	events    eventLister
	drops     *countingDrops
	svc       *CoreService
}

// countingDrops counts the audit events the recorder reports as dropped.
type countingDrops struct{ byType map[string]int }

func (c *countingDrops) Inc(eventType string) { c.byType[eventType]++ }

func newAuditFixture(t *testing.T, certSvc grpcservices.CertificateService) *auditFixture {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := testsupport.SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	for _, id := range []int64{persistTenantA, persistTenantB} {
		_, err := db.Exec(`INSERT INTO tenants (id, name, description, status, created_at, updated_at)
			VALUES ($1, $2, $3, 'active', NOW(), NOW()) ON CONFLICT (id) DO NOTHING`,
			id, "audit-persistence-"+strconv.FormatInt(id, 10), t.Name())
		require.NoError(t, err)
	}
	log := logger.NewNop()
	eventStore := postgres.NewSystemEventStore(db.DB, clock.SystemClock{}, log)
	f := &auditFixture{
		rows:      db,
		endpoints: postgres.NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, log),
		stations:  postgres.NewBaseStationRepository(db, clock.SystemClock{}, log),
		events:    eventStore,
		drops:     &countingDrops{byType: map[string]int{}},
	}
	emitter, err := audit.NewEmitter(eventStore, clock.SystemClock{})
	require.NoError(t, err)
	recorder, err := audit.NewRecorder(emitter, log, f.drops)
	require.NoError(t, err)
	f.svc = testCoreService(coreFields{
		endpointSvc:      grpcservices.NewEndpointService(f.endpoints, nil),
		basestationSvc:   grpcservices.NewBaseStationService(f.stations, nil),
		certSvc:          certSvc,
		audit:            recorder,
		platformTenantID: persistPlatformTenant,
		log:              log,
	})
	t.Cleanup(func() { assert.Empty(t, f.drops.byType, "no audit event is dropped") })
	return f
}

func (f *auditFixture) createEndpoint(t *testing.T, tenantID int64) *models.EndPoint {
	t.Helper()
	ep := &models.EndPoint{
		EUI: models.EUIFromString(persistEpEUI), Name: "audit-persistence-endpoint", TenantID: tenantID,
		EPClass: "A", Tags: make(map[string]string), NwkSnKey: testNetworkKey(),
	}
	require.NoError(t, f.endpoints.Create(testutil.TestContext(), ep))
	stored, err := f.endpoints.GetByEUI(testutil.TestContext(), tenantID, ep.EUI[:])
	require.NoError(t, err)
	return stored
}

func (f *auditFixture) createStation(t *testing.T, tenantID int64) *models.BaseStation {
	t.Helper()
	url := "tls://kilocenter.local:5000"
	bs := &models.BaseStation{
		EUI: models.EUIFromString(persistBsEUI), TenantID: tenantID, Name: persistBsName,
		ConnectionType: models.ConnectionTypeBSSCI, ServiceCenterURL: &url,
	}
	require.NoError(t, f.stations.Create(testutil.TestContext(), bs))
	stored, err := f.stations.GetByEUI(testutil.TestContext(), tenantID, bs.EUI[:])
	require.NoError(t, err)
	return stored
}

func (f *auditFixture) eventsOfType(t *testing.T, eventType string) []persistedEvent {
	t.Helper()
	var rows []persistedEvent
	require.NoError(t, f.rows.Select(&rows, `SELECT tenant_id, endpoint_id, basestation_id,
		COALESCE(source_name, '') AS source_name, COALESCE(user_id, '') AS user_id, data
		FROM system_events WHERE event_type = $1`, eventType))
	return rows
}

func (f *auditFixture) tenantSees(t *testing.T, tenantID int64, eventType string) bool {
	t.Helper()
	listed, err := f.events.GetEvents(testutil.TestContext(), models.SystemEventFilter{
		TenantID: strconv.FormatInt(tenantID, 10), EventTypes: []string{eventType}, Limit: persistListLimit,
	})
	require.NoError(t, err)
	return len(listed) > 0
}

func tenantCtx(tenantID int64) context.Context {
	return pkgcontext.WithUserID(testutil.TestContextWithTenantAndOrg(tenantID, auditTestOrg), auditTestActor)
}

func TestDeleteEndPoint_PersistsTheDeletionInTheDeletingTenantOnly(t *testing.T) {
	f := newAuditFixture(t, nil)
	removed := f.createEndpoint(t, persistTenantA)

	_, err := f.svc.DeleteEndPoint(tenantCtx(persistTenantB), &pb.DeleteEndPointRequest{EpEui: persistEpEUI})
	require.Error(t, err, "another tenant cannot delete the endpoint")
	assert.Empty(t, f.eventsOfType(t, models.EventTypeEndpointDeleted), "a refused delete records nothing")

	_, err = f.svc.DeleteEndPoint(tenantCtx(persistTenantA), &pb.DeleteEndPointRequest{EpEui: persistEpEUIDashed})
	require.NoError(t, err)

	rows := f.eventsOfType(t, models.EventTypeEndpointDeleted)
	require.Len(t, rows, 1, "the deletion is on the audit trail")
	got := rows[0]
	assert.Equal(t, persistTenantA, got.TenantID.Int64)
	assert.False(t, got.EndpointID.Valid, "the removed row cannot be linked")
	assert.Equal(t, persistEpEUI, got.SourceName)
	assert.Equal(t, auditTestActor, got.UserID)
	details := got.details(t)
	assert.Equal(t, persistEpEUI, details[models.EventDetailKeyEpEui])
	assert.EqualValues(t, removed.ID, details[models.EventDetailKeyEndpointID], "the removed row's id is kept")
	assert.True(t, f.tenantSees(t, persistTenantA, models.EventTypeEndpointDeleted))
	assert.False(t, f.tenantSees(t, persistTenantB, models.EventTypeEndpointDeleted))
}

func TestDeleteBaseStation_PersistsTheDeregistrationInTheDeletingTenantOnly(t *testing.T) {
	f := newAuditFixture(t, nil)
	removed := f.createStation(t, persistTenantA)

	_, err := f.svc.DeleteBaseStation(tenantCtx(persistTenantB), &pb.DeleteBaseStationRequest{BsEui: persistBsEUI})
	require.Error(t, err, "another tenant cannot delete the station")
	assert.Empty(t, f.eventsOfType(t, models.EventTypeBSDeregistered), "a refused delete records nothing")

	_, err = f.svc.DeleteBaseStation(tenantCtx(persistTenantA), &pb.DeleteBaseStationRequest{BsEui: persistBsEUIDashed})
	require.NoError(t, err)

	rows := f.eventsOfType(t, models.EventTypeBSDeregistered)
	require.Len(t, rows, 1, "the deregistration is on the audit trail")
	got := rows[0]
	assert.Equal(t, persistTenantA, got.TenantID.Int64)
	assert.False(t, got.BasestationID.Valid, "the removed row cannot be linked")
	assert.Equal(t, persistBsEUI, got.SourceName)
	details := got.details(t)
	assert.Equal(t, persistBsEUI, details[models.EventDetailKeyBsEui])
	assert.Equal(t, persistBsName, details[models.EventDetailKeyBaseStationName])
	assert.EqualValues(t, removed.ID, details[models.EventDetailKeyBaseStationID], "the removed row's id is kept")
	assert.True(t, f.tenantSees(t, persistTenantA, models.EventTypeBSDeregistered))
	assert.False(t, f.tenantSees(t, persistTenantB, models.EventTypeBSDeregistered))
}

// persistCertSvc answers the certificate RPCs the way the certificate
// service does once issuance and renewal have succeeded.
type persistCertSvc struct {
	mockCertSvc
	station *models.BaseStation
}

func (s *persistCertSvc) GenerateCertificate(_ context.Context, req *grpcservices.CertificateRequest) (*grpcservices.CertificateResponse, error) {
	return &grpcservices.CertificateResponse{BsEUI: req.BsEUI, BaseStationID: s.station.ID}, nil
}

func TestGenerateCertificate_PersistsTheEventOnTheStation(t *testing.T) {
	certs := &persistCertSvc{}
	f := newAuditFixture(t, certs)
	certs.station = f.createStation(t, persistTenantA)

	_, err := f.svc.GenerateCertificate(tenantCtx(persistTenantA),
		&pb.GenerateCertificateRequest{BsEui: persistBsEUIDashed, ValidityDays: 365})
	require.NoError(t, err)

	rows := f.eventsOfType(t, models.EventTypeCertificateGenerated)
	require.Len(t, rows, 1)
	got := rows[0]
	assert.Equal(t, persistTenantA, got.TenantID.Int64)
	require.True(t, got.BasestationID.Valid, "the station page lists the certificate")
	assert.Equal(t, certs.station.ID, got.BasestationID.Int64)
	assert.Equal(t, persistBsEUI, got.SourceName, "the canonical EUI, as basestation.registered records it")
	assert.Equal(t, persistBsEUI, got.details(t)[models.EventDetailKeyBsEui])
}

func TestServerCertificateEvents_PersistUnderThePlatformTenant(t *testing.T) {
	f := newAuditFixture(t, &persistCertSvc{})

	_, err := f.svc.RenewServerCertificates(tenantCtx(persistTenantA), &pb.RenewServerCertificatesRequest{})
	require.NoError(t, err)
	_, err = f.svc.GenerateServerCertificates(tenantCtx(persistTenantA), &pb.GenerateServerCertificatesRequest{})
	require.NoError(t, err)

	for _, eventType := range []string{models.EventTypeCertificateServerRenewed, models.EventTypeCertificateServerGenerated} {
		rows := f.eventsOfType(t, eventType)
		require.Len(t, rows, 1, eventType)
		assert.Equal(t, persistPlatformTenant, rows[0].TenantID.Int64, "%s is a server-level event", eventType)
		assert.Equal(t, auditTestActor, rows[0].UserID, "%s names the operator", eventType)
	}
}
