package scaciservices

import (
	"time"

	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/org"
	"github.com/kilocenter/KC-Core/pkg/scaci"
	"github.com/kilocenter/KC-DB/storage/interfaces"
)

// SCACIServiceBundle packages all SCACI service dependencies
type SCACIServiceBundle struct {
	HandshakeSvc       scaci.HandshakeService
	EndpointSvc        scaci.EndpointService
	ULSvc              scaci.ULService
	DLSvc              scaci.DLService
	StatusSvc          scaci.StatusService
	SessionValidator   scaci.SessionValidator
	OperationRecorder  scaci.OperationRecorder
	SessionPersistence scaci.SessionPersistence
	ErrorRecorder      scaci.ErrorRecorder // §3.14 error persistence
}

// NewSCACIServices creates all SCACI services with explicit dependencies
func NewSCACIServices(
	sessionRepo interfaces.SCACISessionRepository,
	operationRepo interfaces.SCACIOperationRepository,
	endpointRepo interfaces.EndpointRepository,
	baseStationRepo interfaces.BaseStationRepository,
	storage interfaces.Storage,
	eventStore interfaces.SystemEventStore,
	bssciServer interface {
		scaci.ULTransmitScheduler
		scaci.DownlinkScheduler
		scaci.DetachPropagator
	},
	log logger.Logger,
	orgResolver org.Resolver,
	defaultTenantID int64,
	strictOrgResolution bool,
	scEui uint64,
	scVendor, scModel, scName, scSwVersion string,
	serviceStart time.Time,
) *SCACIServiceBundle {
	// Create certificate verifier
	certVerifier := NewCertificateVerifier(log)

	// Create handshake service with all dependencies
	handshakeSvc := NewHandshakeService(
		sessionRepo,
		log,
		orgResolver,
		defaultTenantID,
		strictOrgResolution,
		certVerifier,
		scEui,
		scVendor,
		scModel,
		scName,
		scSwVersion,
	)

	// Create endpoint service
	endpointSvc := NewEndpointService(
		endpointRepo,
		bssciServer, // DetachPropagator
		log,
	)

	// Create status service (must be created before UL service for preference lookup)
	statusSvc := NewStatusService(
		baseStationRepo,
		endpointRepo,
		serviceStart,
	)

	// Create UL service (delegates to BSSCI scheduler, uses statusSvc for preference lookup)
	ulSvc := NewULService(
		bssciServer, // ULTransmitScheduler
		statusSvc,   // StatusService for BS preference lookup (§3.9.1)
		log,
	)

	// Create DL service (delegates to BSSCI scheduler)
	dlSvc := NewDLService(
		bssciServer, // DownlinkScheduler
		storage,
		log,
	)

	// Create session validator
	sessionValidator := NewSessionValidator()

	// Create operation recorder
	operationRecorder := NewOperationRecorder(operationRepo)

	// Create session persistence
	sessionPersistence := NewSessionPersistence(sessionRepo, log)

	// Create error recorder (§3.14 error persistence and event emission)
	errorRecorder := scaci.NewErrorRecorder(operationRepo, eventStore, log)

	return &SCACIServiceBundle{
		HandshakeSvc:       handshakeSvc,
		EndpointSvc:        endpointSvc,
		ULSvc:              ulSvc,
		DLSvc:              dlSvc,
		StatusSvc:          statusSvc,
		SessionValidator:   sessionValidator,
		OperationRecorder:  operationRecorder,
		SessionPersistence: sessionPersistence,
		ErrorRecorder:      errorRecorder,
	}
}
