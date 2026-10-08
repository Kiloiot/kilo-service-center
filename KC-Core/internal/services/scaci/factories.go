package scaciservices

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/org"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scaci"
)

// SCACIServiceBundle packages all SCACI service dependencies
type SCACIServiceBundle struct {
	HandshakeSvc      scaci.HandshakeService
	EndpointSvc       scaci.EndpointService
	ULSvc             scaci.ULService
	DLSvc             scaci.DLService
	StatusSvc         scaci.StatusService
	SessionValidator  scaci.SessionValidator
	OperationRecorder scaci.OperationRecorder
	ErrorRecorder     scaci.ErrorRecorder // §3.14 error persistence
}

// NewSCACIServices creates all SCACI services with explicit dependencies
func NewSCACIServices(
	sessionRepo SessionResumeReader,
	operationRepo SCACIOperationStore,
	endpointRepo EndpointStore,
	baseStationRepo BaseStationStore,
	downlinks DownlinkStore,
	downlinkEvents EnqueueRecorder,
	eventStore scaci.ErrorEventStore,
	bssciServer interface {
		scaci.ULTransmitScheduler
		scaci.DownlinkScheduler
		scaci.DetachPropagator
	},
	decider AttachmentDecider,
	queueIDs QueueIDAllocator,
	downlinkLifetime time.Duration,
	log logger.Logger,
	orgResolver org.Resolver,
	defaultTenantID int64,
	strictOrgResolution bool,
	scEui uint64,
	scVendor, scModel, scName, scSwVersion string,
	serviceStart time.Time,
	clk clock.Clock,
) (*SCACIServiceBundle, error) {
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
		scaci.NewSessionFactory(clk),
	)

	// Create endpoint service
	endpointSvc, err := NewEndpointService(
		endpointRepo,
		bssciServer, // DetachPropagator
		decider,
		log,
	)
	if err != nil {
		return nil, err
	}

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
	dlSvc, err := NewDLService(
		bssciServer, // DownlinkScheduler
		downlinks,
		queueIDs,
		downlinkLifetime,
		downlinkEvents,
		log,
	)
	if err != nil {
		return nil, err
	}

	// Create session validator
	sessionValidator := NewSessionValidator()

	// Create operation recorder
	operationRecorder := NewOperationRecorder(operationRepo)

	// Create error recorder (§3.14 error persistence and event emission)
	errorRecorder := scaci.NewErrorRecorder(operationRepo, eventStore, log)

	return &SCACIServiceBundle{
		HandshakeSvc:      handshakeSvc,
		EndpointSvc:       endpointSvc,
		ULSvc:             ulSvc,
		DLSvc:             dlSvc,
		StatusSvc:         statusSvc,
		SessionValidator:  sessionValidator,
		OperationRecorder: operationRecorder,
		ErrorRecorder:     errorRecorder,
	}, nil
}
