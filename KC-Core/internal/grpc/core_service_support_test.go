package grpc

import (
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinks"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// coreFields is the flat collaborator set tests wire; testCoreService fans it
// out into the handler groups the facade embeds.
type coreFields struct {
	endpointSvc            grpcservices.EndpointService
	basestationSvc         grpcservices.BaseStationService
	messageSvc             messageSvcFake
	downlinkCmd            bssci.DownlinkCommander
	revoker                downlinks.Revoker
	sessionDir             bssci.SessionDirectory
	ulTransmit             bssci.ULTransmitter
	statusReq              bssci.StatusRequester
	pingCmd                bssci.PingCommander
	log                    logger.Logger
	statsStore             MessageStore
	dlrxStorage            DLRXStatusQueryStorage
	systemStatusSvc        grpcservices.SystemStatusService
	scaciQueuer            downlinks.Queuer
	scEui                  uint64
	scVendor               string
	scModel                string
	scName                 string
	scSwVersion            string
	edition                string
	endpointAttachmentSvc  grpcservices.EndpointAttachmentService
	analyticsSvc           grpcservices.AnalyticsService
	eventSvc               EventFeed
	alertSvc               grpcservices.AlertService
	scaciMonitorSvc        grpcservices.ScaciMonitoringService
	certSvc                grpcservices.CertificateService
	blueprintSvc           grpcservices.BlueprintService
	msgListingSvc          grpcservices.MessageListingService
	integrationSvc         grpcservices.IntegrationService
	statisticsSvc          grpcservices.StatisticsService
	endpointStatsStore     EndpointStatsStore
	registrationWindow     grpcservices.RegistrationWindow
	opStatusAdapter        OperationStatusAdapter
	activitySvc            grpcservices.ActivityService
	serviceStart           time.Time
	endpointActivityWindow time.Duration
	bssciSessionCloser     BSSCISessionCloser
	bsEventRecorder        basestation.EventRecorder
	audit                  AuditRecorder
	keyReveals             RequiredAuditRecorder
	platformTenantID       int64
	orgMapper              OrgMapper
	ceBootstrapSvc         CEBootstrapHandler
	ceRegistrySvc          CERegistryHandler
	availabilityReader     BaseStationAvailabilityReader
	messageBucketReader    BaseStationMessageBucketReader
}

func testCoreService(f coreFields) *CoreService {
	recorder := f.audit
	if recorder == nil {
		recorder = &captureAuditRecorder{}
	}
	keyReveals := f.keyReveals
	if keyReveals == nil {
		keyReveals = &captureAuditRecorder{}
	}
	return &CoreService{
		EndpointHandlers:    &EndpointHandlers{endpointSvc: f.endpointSvc, blueprintSvc: f.blueprintSvc, endpointAttachmentSvc: f.endpointAttachmentSvc, endpointStatsStore: f.endpointStatsStore, registrationWindow: f.registrationWindow, opStatusAdapter: f.opStatusAdapter, audit: recorder, keyReveals: keyReveals, endpointActivityWindow: f.endpointActivityWindow, clock: clock.SystemClock{}, log: f.log},
		BaseStationHandlers: &BaseStationHandlers{basestationSvc: f.basestationSvc, statsStore: f.statsStore, statusReq: f.statusReq, pingCmd: f.pingCmd, sessionDir: f.sessionDir, bssciSessionCloser: f.bssciSessionCloser, bsEventRecorder: f.bsEventRecorder, audit: recorder, availabilityReader: f.availabilityReader, messageBucketReader: f.messageBucketReader, orgMapper: f.orgMapper, log: f.log},
		DownlinkHandlers:    testDownlinkHandlers(downlinkFakes{messages: f.messageSvc, queuer: f.scaciQueuer, revoker: f.revoker, endpoints: f.endpointSvc, audit: recorder}),
		ULTransmitHandlers:  &ULTransmitHandlers{sessions: f.sessionDir, transmitter: f.ulTransmit, baseStations: f.basestationSvc, log: f.log},
		DLRXHandlers:        &DLRXHandlers{queries: f.dlrxStorage, statuses: f.messageSvc, stations: f.messageSvc, command: f.downlinkCmd, sessions: f.sessionDir, log: f.log},
		MessageHandlers:     &MessageHandlers{msgListingSvc: f.msgListingSvc, activitySvc: f.activitySvc, log: f.log},
		AnalyticsHandlers:   &AnalyticsHandlers{analyticsSvc: f.analyticsSvc, eventSvc: f.eventSvc, alertSvc: f.alertSvc, log: f.log},
		CertificateHandlers: &CertificateHandlers{certSvc: f.certSvc, audit: recorder, platformTenantID: f.platformTenantID, log: f.log},
		BlueprintHandlers:   &BlueprintHandlers{blueprintSvc: f.blueprintSvc, endpointSvc: f.endpointSvc, audit: recorder, log: f.log},
		IntegrationHandlers: &IntegrationHandlers{integrationSvc: f.integrationSvc, audit: recorder, log: f.log},
		ScaciHandlers:       &ScaciHandlers{scaciMonitorSvc: f.scaciMonitorSvc, log: f.log},
		FederationHandlers:  &FederationHandlers{ceBootstrapSvc: f.ceBootstrapSvc, ceRegistrySvc: f.ceRegistrySvc, log: f.log},
		SystemHandlers:      &SystemHandlers{statisticsSvc: f.statisticsSvc, systemStatusSvc: f.systemStatusSvc, scEui: f.scEui, scVendor: f.scVendor, scModel: f.scModel, scName: f.scName, scSwVersion: f.scSwVersion, edition: f.edition, serviceStart: f.serviceStart, log: f.log},
	}
}

// testServiceStart is the process start every handler test reports uptime from.
var testServiceStart = time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
