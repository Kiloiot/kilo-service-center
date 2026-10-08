// Package scaci test mocks
//
// This file provides testify/mock implementations for the service interfaces
// defined in contracts.go. These mocks enable unit testing of handlers without
// requiring concrete service implementations or database connections.
//
// Usage Pattern:
//
//	mockHandshake := new(MockHandshakeService)
//	mockHandshake.On("ValidateConnect", mock.Anything, mock.Anything, int64(1)).
//		Return(session, response, "")
//
//	server := &Server{
//		handshakeSvc: mockHandshake,
//		// ... other fields
//	}
//
//	// Test handler code
//	err := server.handleConnect(conn, &session, tenantID, opId, payload)
//
//	mockHandshake.AssertExpectations(t)
package scaci

import (
	"context"
	"crypto/x509"
	"net"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/scheduler"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockHandshakeService implements HandshakeService interface for testing
//
// Provides mocks for:
//   - ValidateConnect: Connect flow, version negotiation, session resumption
//   - NegotiateVersion: SCACI §2.1-2.3 version compatibility checks
//   - ResolveResume: SCACI §3.3 session resumption validation
//   - CertificateTenant: tenant the client certificate resolves to
type MockHandshakeService struct {
	mock.Mock
}

// ValidateConnect mocks HandshakeService.ValidateConnect
func (m *MockHandshakeService) ValidateConnect(
	ctx context.Context,
	req *Connect,
	cert *x509.Certificate,
) (*Session, *ConnectResponse, string) {
	args := m.Called(ctx, req, cert)

	var session *Session
	if args.Get(0) != nil {
		session = args.Get(0).(*Session)
	}

	var response *ConnectResponse
	if args.Get(1) != nil {
		response = args.Get(1).(*ConnectResponse)
	}

	return session, response, args.String(2)
}

// NegotiateVersion mocks HandshakeService.NegotiateVersion
func (m *MockHandshakeService) NegotiateVersion(ctx context.Context, clientVersion string) (string, string) {
	args := m.Called(ctx, clientVersion)
	return args.String(0), args.String(1)
}

// CertificateTenant mocks HandshakeService.CertificateTenant
func (m *MockHandshakeService) CertificateTenant(ctx context.Context, cert *x509.Certificate) (int64, bool) {
	args := m.Called(ctx, cert)
	return args.Get(0).(int64), args.Bool(1)
}

// ResolveResume mocks HandshakeService.ResolveResume
func (m *MockHandshakeService) ResolveResume(
	ctx context.Context,
	ac ApplicationCenter,
	acUUID []byte,
	scUUID []byte,
	acOpId, scOpId int64,
	requestVersion string,
) (bool, string) {
	args := m.Called(ctx, ac, acUUID, scUUID, acOpId, scOpId, requestVersion)
	return args.Bool(0), args.String(1)
}

// MockEndpointService implements EndpointService interface for testing
//
// Provides mocks for:
//   - Register: SCACI §3.6 endpoint registration
//   - Deregister: SCACI §3.7 endpoint deregistration
type MockEndpointService struct {
	mock.Mock
}

// Register mocks EndpointService.Register
func (m *MockEndpointService) Register(
	ctx context.Context,
	req *Register,
	tenantID int64,
) string {
	args := m.Called(ctx, req, tenantID)
	return args.String(0)
}

// Deregister mocks EndpointService.Deregister
func (m *MockEndpointService) Deregister(
	ctx context.Context,
	epEui uint64,
	tenantID int64,
) string {
	args := m.Called(ctx, epEui, tenantID)
	return args.String(0)
}

// MockULService implements ULService interface for testing
//
// Provides mocks for:
//   - ScheduleULTransmit: SCACI §3.9 UL data transmit scheduling
type MockULService struct {
	mock.Mock
}

// ScheduleULTransmit mocks ULService.ScheduleULTransmit
func (m *MockULService) ScheduleULTransmit(
	ctx context.Context,
	req *mioty.ULDataTransmit,
	tenantID int64,
) (int64, uint64, string) {
	args := m.Called(ctx, req, tenantID)
	return args.Get(0).(int64), args.Get(1).(uint64), args.String(2)
}

// MockDLService implements DLService interface for testing
//
// Provides mocks for:
//   - QueueDownlink: SCACI §3.10 downlink queue operation
//   - RevokeDownlink: SCACI §3.11 downlink revocation
type MockDLService struct {
	mock.Mock
}

// QueueDownlink mocks DLService.QueueDownlink
func (m *MockDLService) QueueDownlink(
	ctx context.Context,
	req *mioty.DLDataQueue,
	tenantID int64,
	organizationID uuid.UUID,
) (DownlinkQueueOutcome, string) {
	args := m.Called(ctx, req, tenantID, organizationID)
	return args.Get(0).(DownlinkQueueOutcome), args.String(1)
}

// RevokeDownlink mocks DLService.RevokeDownlink
func (m *MockDLService) RevokeDownlink(
	ctx context.Context,
	ref scheduler.DownlinkRef,
) (uint64, string) {
	args := m.Called(ctx, ref)
	return args.Get(0).(uint64), args.String(1)
}

// MockSessionValidator implements SessionValidator interface for testing
//
// Provides mocks for:
//   - ValidateConnectFields: SCACI §3.3 Connect message field validation
type MockSessionValidator struct {
	mock.Mock
}

// ValidateConnectFields mocks SessionValidator.ValidateConnectFields
func (m *MockSessionValidator) ValidateConnectFields(req *Connect) string {
	args := m.Called(req)
	return args.String(0)
}

// MockCertificateVerifier implements CertificateVerifier interface for testing
//
// Provides mocks for:
//   - VerifyCertificate: Certificate expiry, key usage, and subject validation
type MockCertificateVerifier struct {
	mock.Mock
}

// VerifyCertificate mocks CertificateVerifier.VerifyCertificate
func (m *MockCertificateVerifier) VerifyCertificate(ctx context.Context, cert *x509.Certificate) string {
	args := m.Called(ctx, cert)
	return args.String(0)
}

// MockOperationRecorder implements OperationRecorder interface for testing
//
// Provides mocks for:
//   - Record: SCACI operation persistence for resume safety
type MockOperationRecorder struct {
	mock.Mock
}

// Record mocks OperationRecorder.Record
func (m *MockOperationRecorder) Record(
	ctx context.Context,
	session *Session,
	opId int64,
	command string,
	direction models.OperationDirection,
	data map[string]interface{},
) error {
	args := m.Called(ctx, session, opId, command, direction, data)
	return args.Error(0)
}

// EnsureUplinkOperation mocks OperationRecorder.EnsureUplinkOperation
func (m *MockOperationRecorder) EnsureUplinkOperation(
	ctx context.Context,
	session *Session,
	opId int64,
	sourceMessageID string,
	data map[string]interface{},
) (*models.SCACIOperation, bool, error) {
	args := m.Called(ctx, session, opId, sourceMessageID, data)
	operation, _ := args.Get(0).(*models.SCACIOperation)
	return operation, args.Bool(1), args.Error(2)
}

// ============================================================================
// MockSessionPersistence for ping heartbeat tests (SCACI §3.4)
// ============================================================================

// MockSessionPersistence implements SessionPersistence interface for testing
//
// Provides mocks for:
//   - PersistConnectSync: Synchronous session creation for audit trail
//   - PersistHeartbeat: SCACI §3.4 ping heartbeat persistence
//   - PersistOpIDs: SCACI §3.2 operation ID counter persistence
type MockSessionPersistence struct {
	mock.Mock
}

// PersistConnectSync mocks SessionPersistence.PersistConnectSync
func (m *MockSessionPersistence) PersistConnectSync(ctx context.Context, session *Session, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, negotiatedVersion string) (int64, error) {
	args := m.Called(ctx, session, certFingerprint, certSubject, remoteAddr, tlsVersion, cipherSuite, negotiatedVersion)
	return args.Get(0).(int64), args.Error(1)
}

// PersistHeartbeat mocks SessionPersistence.PersistHeartbeat
func (m *MockSessionPersistence) PersistHeartbeat(ctx context.Context, session *Session) error {
	return m.Called(ctx, session).Error(0)
}

// PersistOpIDs mocks SessionPersistence.PersistOpIDs
func (m *MockSessionPersistence) PersistOpIDs(ctx context.Context, session *Session, ids OpIDPair) error {
	return m.Called(ctx, session, ids).Error(0)
}

// ============================================================================
// Mock net.Conn for Send* Helper Tests (SCACI §2.5)
// ============================================================================

// mockRemoteAddr is the peer address of every mockConn.
var mockRemoteAddr net.Addr = &net.UnixAddr{Name: "application-center-under-test", Net: "unix"}

// mockConn implements net.Conn for testing Send* helpers.
// Only Write is called by sendResponse; other methods return defaults.
type mockConn struct {
	written []byte
}

func (m *mockConn) Read(_ []byte) (n int, err error) { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error) {
	m.written = append(m.written, b...)
	return len(b), nil
}
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return mockRemoteAddr }
func (m *mockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(_ time.Time) error { return nil }

// ============================================================================
// Additional DLService Mock Methods for Deregister Cleanup Tests (SCACI §3.7.3)
// ============================================================================

// GetDownlinkQueue mocks DLService.GetDownlinkQueue for deregister cleanup
func (m *MockDLService) GetDownlinkQueue(
	ctx context.Context,
	tenantID int64,
	filter storage.DownlinkQueueFilter,
) ([]*storage.DownlinkMessage, error) {
	args := m.Called(ctx, tenantID, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.DownlinkMessage), args.Error(1)
}

// EnqueueDownlink mocks DLService.EnqueueDownlink
func (m *MockDLService) EnqueueDownlink(
	ctx context.Context,
	dlMsg *storage.DownlinkMessage,
) (*storage.DownlinkMessage, error) {
	args := m.Called(ctx, dlMsg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.DownlinkMessage), args.Error(1)
}

// GetDownlinksByPacketCnt mocks DLService.GetDownlinksByPacketCnt
func (m *MockDLService) GetDownlinksByPacketCnt(
	ctx context.Context,
	query storage.PacketCounterDownlinks,
) ([]*storage.DownlinkMessage, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.DownlinkMessage), args.Error(1)
}

// ============================================================================
// Additional EndpointService Mock Methods (SCACI §3.7.3)
// ============================================================================

// PropagateDetachToAll mocks EndpointService.PropagateDetachToAll
func (m *MockEndpointService) PropagateDetachToAll(ctx context.Context, tenantID int64, epEui uint64) []error {
	args := m.Called(ctx, tenantID, epEui)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]error)
}

// GetByEUI mocks EndpointService.GetByEUI
func (m *MockEndpointService) GetByEUI(
	ctx context.Context,
	tenantID int64,
	eui []byte,
) (*models.EndPoint, string) {
	args := m.Called(ctx, tenantID, eui)
	if args.Get(0) == nil {
		return nil, args.String(1)
	}
	return args.Get(0).(*models.EndPoint), args.String(1)
}

// Attach mocks EndpointService.Attach
func (m *MockEndpointService) Attach(ctx context.Context, endpoint *models.EndPoint) string {
	args := m.Called(ctx, endpoint)
	return args.String(0)
}

// ============================================================================
// Mock Repository Types for Constructor Validation Tests
//
// These stub types satisfy the interfaces.SCACI*Repository interfaces
// without implementing actual logic. All methods panic if called,
// which is fine since constructor validation tests only check nil rejection.
// ============================================================================

// mockOperationRepoStub satisfies interfaces.SCACIOperationRepository for constructor tests
type mockOperationRepoStub struct{}

func (m *mockOperationRepoStub) RecordOperation(_ context.Context, _ *models.SCACIOperationRequest) (*models.SCACIOperation, error) {
	panic("not implemented")
}

func (m *mockOperationRepoStub) UpdateOperationState(_ context.Context, _, _ int64, _ models.OperationState, _ map[string]interface{}) error {
	panic("not implemented")
}

func (m *mockOperationRepoStub) GetOperationByOpID(_ context.Context, _, _ int64) (*models.SCACIOperation, error) {
	panic("not implemented")
}

func (m *mockOperationRepoStub) GetPendingOperations(_ context.Context, _ int64) ([]*models.SCACIOperation, error) {
	panic("not implemented")
}

func (m *mockOperationRepoStub) UpdateOperationStateWithError(_ context.Context, _, _ int64, _ models.OperationState, _ int, _, _ string, _ map[string]interface{}) error {
	panic("not implemented")
}

func (m *mockOperationRepoStub) CompleteFailedOperation(_ context.Context, _, _ int64, _ map[string]interface{}) error {
	panic("not implemented")
}

// ============================================================================
// MockStatusService for Constructor Validation Tests (SCACI §3.5)
// ============================================================================

// MockStatusService implements StatusService interface for testing
type MockStatusService struct {
	mock.Mock
}

// GetUptime mocks StatusService.GetUptime
func (m *MockStatusService) GetUptime() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// GetBaseStations mocks StatusService.GetBaseStations
func (m *MockStatusService) GetBaseStations(ctx context.Context, tenantID int64) ([]*models.BaseStation, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.BaseStation), args.Error(1)
}

// GetBaseStation mocks StatusService.GetBaseStation
func (m *MockStatusService) GetBaseStation(ctx context.Context, tenantID int64, eui []byte) (*models.BaseStation, error) {
	args := m.Called(ctx, tenantID, eui)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BaseStation), args.Error(1)
}

// GetPreferredBaseStation mocks StatusService.GetPreferredBaseStation
func (m *MockStatusService) GetPreferredBaseStation(ctx context.Context, tenantID int64, epEui []byte) (*uint64, bool, error) {
	args := m.Called(ctx, tenantID, epEui)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).(*uint64), args.Bool(1), args.Error(2)
}

// tenantDownlinkRef is the revoke reference of a session without an
// organization, which reaches the whole tenant.
func tenantDownlinkRef(tenantID int64, epEUI, queID uint64) scheduler.DownlinkRef {
	return scheduler.DownlinkRef{TenantID: tenantID, QueID: queID, EpEUI: &epEUI}
}

// tenantCounterDownlinks is the dlDataRev lookup of a session without an
// organization.
func tenantCounterDownlinks(tenantID int64, epEUI uint64, packetCnt uint32) storage.PacketCounterDownlinks {
	return storage.PacketCounterDownlinks{TenantID: tenantID, EpEUI: epEUI, PacketCnt: packetCnt}
}
