// Package bsscitest provides the shared BSSCI service fixture used by
// protocol-level tests: fully wired services over in-memory fakes. It is
// imported only from test files and is not linked into any binary.
package bsscitest

import (
	"fmt"
	"sync"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	repodoubles "github.com/Kiloiot/kilo-service-center/KC-Core/internal/testsupport/repodoubles"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
)

// ============================================================================
// Mock Repository Implementations for Testing
// ============================================================================

// Fixture wiring faults; the fixture has no test to fail.
const (
	errFmtAuditLoggerWiring     = "bsscitest: audit logger: %v"
	errFmtDownlinkServiceWiring = "bsscitest: downlink service: %v"
)

// CreateTestServices creates minimal service instances for testing.
// Returns services that won't panic but don't provide full functionality.
//
// Usage in tests:
//
//	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver := bssciservices.CreateTestServices(logger, mockEventStore)
//	server := &Server{
//	    logger:          logger,
//	    sessionSvc:      sessionSvc,
//	    downlinkSvc:     downlinkSvc,
//	    statusSvc:       statusSvc,
//	    connectionSvc:   connectionSvc,
//	    broadcaster:     broadcaster,
//	    queueSerializer: queueSerializer,
//	    auditLogger:     auditLogger,
//	    tenantResolver:  tenantResolver,
//	    // ... other fields
//	}
//
// Note: Services are created with nil DB dependencies. Tests that need real
// persistence should provide their own mock implementations or use integration tests.
func CreateTestServices(log logger.Logger, eventStore interfaces.SystemEventStore) (
	bssci.SessionService,
	bssci.DownlinkService,
	bssci.StatusService,
	bssci.BaseStationConnectionRegistry,
	bssci.SCACIBroadcaster,
	bssci.QueueSerializer,
	bssci.AuditLogger,
	bssci.TenantResolver,
	repodoubles.TestStore,
) {
	// Wrap zap logger for migrated services

	// SessionService with complete mock repositories
	// All repository interfaces fully implemented - no nil pointer panics
	// Order: bsSessionRepo, pendingOpsRepo, systemEventStore, tenantID, log
	sessionSvc := bssciservices.NewSessionService(
		repodoubles.NewBaseStationSessionRepo(),   // BaseStationSessionRepository
		&repodoubles.PendingOperationRepository{}, // PendingOperationRepository
		&repodoubles.SystemEventStore{},           // SystemEventStore
		1,                                         // tenantID
		bssci.TestScEui01,                         // serviceCenterEUI
		log,                                       // logger
	)

	// StatusService - maintains in-memory pendingOps map
	testPendingOps := make(map[bssci.SessionOpKey]*bssci.PendingOperation)
	var testMu sync.RWMutex
	statusSvc := bssciservices.NewStatusService(&testPendingOps, &testMu, &repodoubles.PendingOperationRepository{}, log)

	// Connection registry over a nil manager mock is unusable; tests that
	// exercise registration provide their own fake
	var connectionSvc bssci.BaseStationConnectionRegistry

	// SCACIForwarder - initially unwired, safe to call (returns nil if no SCACI)
	broadcaster := bssciservices.NewSCACIForwarder(log)

	// QueueSerializer - stateless, no dependencies
	queueSerializer := bssciservices.NewQueueSerializer()

	// AuditLogger writes through the caller's event store; tests that pass
	// none get a discarding recorder so audit paths stay exercisable.
	var recorder bssciservices.SystemEventRecorder = eventStore
	if eventStore == nil {
		recorder = &repodoubles.SystemEventStore{}
	}
	auditLogger, err := bssciservices.NewAuditLogger(bssciservices.AuditLogDeps{
		Events: recorder, Downlinks: repodoubles.DownlinkLookup{}, Stations: &repodoubles.BaseStationRepo{},
		Clock: clock.SystemClock{}, Logger: log,
	})
	if err != nil {
		panic(fmt.Sprintf(errFmtAuditLoggerWiring, err))
	}

	// TenantResolver - with nil queueStore for tests
	tenantResolver := bssciservices.NewTenantResolver(nil)

	// Create mock storage with MIOTY repositories
	mockStore := repodoubles.NewStorage()

	resultReporter, err := bssciservices.NewDownlinkResultReporter(broadcaster, bssciservices.DownlinkResultsWithoutMQTT{},
		auditLogger, auditLogger, bssciservices.NewBackgroundWork(), log)
	if err != nil {
		panic(fmt.Sprintf(errFmtDownlinkServiceWiring, err))
	}

	// DownlinkService - with all dependencies, mock storage for tests
	revokeAnswers, err := bssciservices.NewRevokeAnswers(bssciservices.RevokeAnswerDeps{
		Logger: log, Tenants: tenantResolver, Revocations: mockStore, Expiries: resultReporter,
		Serializer: queueSerializer, NotHeldCodes: []int{bssci.POSIX_ENOENT},
	})
	if err != nil {
		panic(fmt.Sprintf(errFmtDownlinkServiceWiring, err))
	}
	downlinkSvc, err := bssciservices.NewDownlinkService(bssciservices.DownlinkServiceDeps{
		Logger: log, Tenants: tenantResolver, Outcomes: mockStore, Holders: mockStore,
		Results: resultReporter, Serializer: queueSerializer, Clock: clock.SystemClock{},
	}, revokeAnswers)
	if err != nil {
		panic(fmt.Sprintf(errFmtDownlinkServiceWiring, err))
	}

	return sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStore
}
