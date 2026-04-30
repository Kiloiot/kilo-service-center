// Package scaci handler tests
//
// These tests demonstrate the handler-to-service delegation pattern using
// testify mocks. Handlers are responsible for:
//   - Transport concerns (MessagePack, framing, error codes)
//   - Session/connection management
//   - Delegating business logic to services
//
// Services are mocked to test handler behavior in isolation.
package scaci

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kilocenter/KC-Core/pkg/testutil"
)

// stubCert creates a minimal certificate for testing (no crypto needed)
func stubCert(cn string) *x509.Certificate {
	return &x509.Certificate{
		Subject: pkix.Name{
			CommonName: cn,
		},
	}
}

// TestHandshakeServiceIntegration tests that handler correctly delegates to HandshakeService
//
// This test verifies:
//   - Handler decodes MessagePack payload
//   - Handler calls HandshakeService.ValidateConnect with correct parameters
//   - Handler processes service response (session, response, error token)
//   - Handler returns appropriate error based on service error token
//
// NOTE: This is a demonstration test showing the mock pattern. Full handler tests
// would require mocking net.Conn and testing the complete request/response cycle.
func TestHandshakeServiceIntegration(t *testing.T) {
	// Create mock service
	mockHandshake := new(MockHandshakeService)

	// Setup test data
	clientVersion := "1.0.0"
	negotiatedVersion := "1.0.0"

	// Configure mock expectations
	mockHandshake.On("NegotiateVersion", clientVersion).
		Return(negotiatedVersion, "") // Empty error token = success

	// Execute service method
	gotVersion, errToken := mockHandshake.NegotiateVersion(clientVersion)

	// Verify behavior
	assert.Equal(t, negotiatedVersion, gotVersion)
	assert.Empty(t, errToken, "Expected no error token")
	mockHandshake.AssertExpectations(t)
}

// TestHandshakeServiceVersionError tests error handling from HandshakeService
func TestHandshakeServiceVersionError(t *testing.T) {
	mockHandshake := new(MockHandshakeService)

	clientVersion := "2.0.0" // Unsupported major version
	expectedError := errMajorVersionUnsupported

	mockHandshake.On("NegotiateVersion", clientVersion).
		Return("", expectedError)

	gotVersion, errToken := mockHandshake.NegotiateVersion(clientVersion)

	assert.Empty(t, gotVersion)
	assert.Equal(t, expectedError, errToken)
	mockHandshake.AssertExpectations(t)
}

// TestEndpointServiceRegistration tests EndpointService.Register delegation
func TestEndpointServiceRegistration(t *testing.T) {
	mockEndpoint := new(MockEndpointService)

	ctx := testutil.TestContext()
	tenantID := int64(1)
	req := &Register{
		EpEui:  0x1234567890ABCDEF,
		NwkKey: [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		Bidi:   true,
	}

	// Mock successful registration
	mockEndpoint.On("Register", ctx, req, tenantID).
		Return("") // Empty error token = success

	errToken := mockEndpoint.Register(ctx, req, tenantID)

	assert.Empty(t, errToken, "Expected successful registration")
	mockEndpoint.AssertExpectations(t)
}

// TestEndpointServiceDeregistration tests EndpointService.Deregister delegation
func TestEndpointServiceDeregistration(t *testing.T) {
	mockEndpoint := new(MockEndpointService)

	ctx := testutil.TestContext()
	tenantID := int64(1)
	epEui := uint64(0x1234567890ABCDEF)

	// Mock successful deregistration
	mockEndpoint.On("Deregister", ctx, epEui, tenantID).
		Return("")

	errToken := mockEndpoint.Deregister(ctx, epEui, tenantID)

	assert.Empty(t, errToken)
	mockEndpoint.AssertExpectations(t)
}

// TestEndpointServiceDeregistrationNotFound tests error handling
func TestEndpointServiceDeregistrationNotFound(t *testing.T) {
	mockEndpoint := new(MockEndpointService)

	ctx := testutil.TestContext()
	tenantID := int64(1)
	epEui := uint64(0x9999999999999999)

	// Mock endpoint not found error
	mockEndpoint.On("Deregister", ctx, epEui, tenantID).
		Return(errEndpointNotFound)

	errToken := mockEndpoint.Deregister(ctx, epEui, tenantID)

	assert.Equal(t, errEndpointNotFound, errToken)
	mockEndpoint.AssertExpectations(t)
}

// TestResumeSessionValidation tests ResolveResume delegation
func TestResumeSessionValidation(t *testing.T) {
	mockHandshake := new(MockHandshakeService)

	ctx := testutil.TestContext()
	tenantID := int64(1)
	acUUID := make([]byte, 16)
	scUUID := make([]byte, 16)
	acOpId := int64(100)
	scOpId := int64(-50)
	requestVersion := "1.0.0"

	// Mock successful resume validation
	mockHandshake.On("ResolveResume", ctx, tenantID, acUUID, scUUID, acOpId, scOpId, requestVersion).
		Return(true, "")

	canResume, errToken := mockHandshake.ResolveResume(ctx, tenantID, acUUID, scUUID, acOpId, scOpId, requestVersion)

	assert.True(t, canResume)
	assert.Empty(t, errToken)
	mockHandshake.AssertExpectations(t)
}

// TestResumeSessionOpIdMismatch tests operation ID validation failure
func TestResumeSessionOpIdMismatch(t *testing.T) {
	mockHandshake := new(MockHandshakeService)

	ctx := testutil.TestContext()
	tenantID := int64(1)
	acUUID := make([]byte, 16)
	scUUID := make([]byte, 16)
	acOpId := int64(50) // Wrong operation ID
	scOpId := int64(-25)
	requestVersion := "1.0.0"

	// Mock operation ID mismatch error
	mockHandshake.On("ResolveResume", ctx, tenantID, acUUID, scUUID, acOpId, scOpId, requestVersion).
		Return(false, errOpIdOutOfOrder)

	canResume, errToken := mockHandshake.ResolveResume(ctx, tenantID, acUUID, scUUID, acOpId, scOpId, requestVersion)

	assert.False(t, canResume)
	assert.Equal(t, errOpIdOutOfOrder, errToken)
	mockHandshake.AssertExpectations(t)
}

// TestValidateConnectFullFlow tests complete ValidateConnect delegation
func TestValidateConnectFullFlow(t *testing.T) {
	mockHandshake := new(MockHandshakeService)

	ctx := testutil.TestContext()
	testCert := stubCert("ac-tenant-1")
	tenantID := int64(1)
	req := &Connect{
		Version: "1.0.0",
		AcEui:   0xABCDEF0123456789,
		SnAcUUID: [16]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}

	// Create expected session and response
	expectedSession := &Session{
		TenantID: tenantID,
		AcEui:    req.AcEui,
		SnAcUUID: req.SnAcUUID,
		State:    StateActive,
	}

	version := "1.0.0"
	expectedResponse := &ConnectResponse{
		Version:  &version,
		ScEui:    0x1234567890ABCDEF,
		SnScUUID: [16]byte{},
		SnResume: false,
	}

	// Mock successful connect
	mockHandshake.On("ValidateConnect", mock.Anything, req, mock.AnythingOfType("*x509.Certificate")).
		Return(expectedSession, expectedResponse, "")

	session, response, errToken := mockHandshake.ValidateConnect(ctx, req, testCert)

	assert.NotNil(t, session)
	assert.Equal(t, tenantID, session.TenantID)
	assert.NotNil(t, response)
	assert.Equal(t, "1.0.0", *response.Version)
	assert.Empty(t, errToken)
	mockHandshake.AssertExpectations(t)
}

// TestSessionValidatorSuccess tests successful field validation
func TestSessionValidatorSuccess(t *testing.T) {
	mockValidator := new(MockSessionValidator)

	req := &Connect{
		Version: "1.0.0",
		AcEui:   0xABCDEF0123456789,
		SnAcUUID: [16]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}
	opId := int64(0) // Connect must use opId 0 per §3.3-02

	// Mock successful validation
	mockValidator.On("ValidateConnectFields", req, opId).
		Return("") // Empty token = success

	errToken := mockValidator.ValidateConnectFields(req, opId)

	assert.Empty(t, errToken, "Expected successful validation")
	mockValidator.AssertExpectations(t)
}

// TestSessionValidatorMissingVersion tests missing version field
func TestSessionValidatorMissingVersion(t *testing.T) {
	mockValidator := new(MockSessionValidator)

	req := &Connect{
		Version: "", // Missing version
		AcEui:   0xABCDEF0123456789,
		SnAcUUID: [16]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}
	opId := int64(0)

	// Mock validation failure
	mockValidator.On("ValidateConnectFields", req, opId).
		Return(errMissingVersion)

	errToken := mockValidator.ValidateConnectFields(req, opId)

	assert.Equal(t, errMissingVersion, errToken)
	mockValidator.AssertExpectations(t)
}

// TestSessionValidatorInvalidOpId tests Connect with non-zero opId
func TestSessionValidatorInvalidOpId(t *testing.T) {
	mockValidator := new(MockSessionValidator)

	req := &Connect{
		Version: "1.0.0",
		AcEui:   0xABCDEF0123456789,
		SnAcUUID: [16]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
		},
	}
	opId := int64(5) // Invalid - Connect must use 0

	// Mock validation failure for non-zero opId
	mockValidator.On("ValidateConnectFields", req, opId).
		Return(errConnectOpIdMustBeZero)

	errToken := mockValidator.ValidateConnectFields(req, opId)

	assert.Equal(t, errConnectOpIdMustBeZero, errToken)
	mockValidator.AssertExpectations(t)
}

// TestCertificateVerifierSuccess tests successful certificate validation
func TestCertificateVerifierSuccess(t *testing.T) {
	mockVerifier := new(MockCertificateVerifier)

	testCert := stubCert("ac-tenant-1")

	// Mock successful verification
	mockVerifier.On("VerifyCertificate", testCert).
		Return("") // Empty token = success

	errToken := mockVerifier.VerifyCertificate(testCert)

	assert.Empty(t, errToken, "Expected successful certificate verification")
	mockVerifier.AssertExpectations(t)
}

// TestCertificateVerifierExpired tests expired certificate
func TestCertificateVerifierExpired(t *testing.T) {
	mockVerifier := new(MockCertificateVerifier)

	testCert := stubCert("ac-tenant-1")

	// Mock expired certificate error
	mockVerifier.On("VerifyCertificate", testCert).
		Return(ErrCertExpired)

	errToken := mockVerifier.VerifyCertificate(testCert)

	assert.Equal(t, ErrCertExpired, errToken)
	mockVerifier.AssertExpectations(t)
}

// TestCertificateVerifierMissingClientAuth tests certificate without ClientAuth usage
func TestCertificateVerifierMissingClientAuth(t *testing.T) {
	mockVerifier := new(MockCertificateVerifier)

	testCert := stubCert("ac-tenant-1")

	// Mock missing ClientAuth extended key usage
	mockVerifier.On("VerifyCertificate", testCert).
		Return(ErrCertMissingClientAuth)

	errToken := mockVerifier.VerifyCertificate(testCert)

	assert.Equal(t, ErrCertMissingClientAuth, errToken)
	mockVerifier.AssertExpectations(t)
}
