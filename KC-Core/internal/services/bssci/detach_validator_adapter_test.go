package bssciservices

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/kilocenter/KC-Core/pkg/bssci"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	"github.com/kilocenter/KC-DB/storage"
	"github.com/kilocenter/KC-DB/storage/models"
)

// testDetachValidatorRepo wraps mockEndpointRepository with configurable error return
type testDetachValidatorRepo struct {
	mockEndpointRepository
	errorToReturn error
}

func (t *testDetachValidatorRepo) GetEndpointWithKeysForDetachValidation(ctx context.Context, eui models.EUI) (*models.EndPoint, error) {
	if t.errorToReturn != nil {
		return nil, t.errorToReturn
	}
	return t.mockEndpointRepository.GetEndpointWithKeysForDetachValidation(ctx, eui)
}

func newTestDetachValidatorRepo() *testDetachValidatorRepo {
	return &testDetachValidatorRepo{
		mockEndpointRepository: mockEndpointRepository{
			endpoints: make(map[uint64]*models.EndPoint),
			mu:        &sync.RWMutex{},
		},
	}
}

// TestDetachValidatorDirectAdapter_ValidSignature_ReturnsValidResult verifies successful validation
func TestDetachValidatorDirectAdapter_ValidSignature_ReturnsValidResult(t *testing.T) {
	// Setup: Create endpoint with matching signatures
	signature := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	epEUI := uint64(0x1122334455667788)

	var eui models.EUI
	eui[0] = byte(epEUI >> 56)
	eui[1] = byte(epEUI >> 48)
	eui[2] = byte(epEUI >> 40)
	eui[3] = byte(epEUI >> 32)
	eui[4] = byte(epEUI >> 24)
	eui[5] = byte(epEUI >> 16)
	eui[6] = byte(epEUI >> 8)
	eui[7] = byte(epEUI)

	endpoint := &models.EndPoint{
		EUI:           eui,
		TenantID:      123,
		OwnerTenantID: 123,
		Sign:          signature,
		NwkSnKey:      make([]byte, 16),
		PresharedKey:  make([]byte, 16),
	}

	mockRepo := newTestDetachValidatorRepo()
	mockRepo.endpoints[epEUI] = endpoint

	log := logger.NewNop()
	adapter := NewDetachValidatorDirectAdapter(mockRepo, log)

	// Test: Validate with matching signature
	ctx := testutil.TestContext()
	detachSign := signature // Same as endpoint.Sign

	result, err := adapter.ValidateDetachSignature(ctx, epEUI, detachSign)

	// Assert: No error, valid result
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if !result.Valid {
		t.Error("Expected result.Valid to be true")
	}
	if result.TenantID != 123 {
		t.Errorf("Expected TenantID 123, got %d", result.TenantID)
	}
	if result.OwnerTenantID != 123 {
		t.Errorf("Expected OwnerTenantID 123, got %d", result.OwnerTenantID)
	}
	if result.ValidationStatus != bssci.ValidationStatusValidated {
		t.Errorf("Expected ValidationStatus '%s', got '%s'", bssci.ValidationStatusValidated, result.ValidationStatus)
	}
}

// TestDetachValidatorDirectAdapter_InvalidSignature_ReturnsError verifies signature mismatch handling
func TestDetachValidatorDirectAdapter_InvalidSignature_ReturnsError(t *testing.T) {
	// Setup: Create endpoint with different signature
	epEUI := uint64(0x1122334455667788)

	var eui models.EUI
	eui[0] = byte(epEUI >> 56)
	eui[1] = byte(epEUI >> 48)
	eui[2] = byte(epEUI >> 40)
	eui[3] = byte(epEUI >> 32)
	eui[4] = byte(epEUI >> 24)
	eui[5] = byte(epEUI >> 16)
	eui[6] = byte(epEUI >> 8)
	eui[7] = byte(epEUI)

	endpoint := &models.EndPoint{
		EUI:           eui,
		TenantID:      123,
		OwnerTenantID: 456,
		Sign:          []byte{0x11, 0x22, 0x33, 0x44}, // Different from detachSign
		NwkSnKey:      make([]byte, 16),
		PresharedKey:  make([]byte, 16),
	}

	mockRepo := newTestDetachValidatorRepo()
	mockRepo.endpoints[epEUI] = endpoint

	log := logger.NewNop()
	adapter := NewDetachValidatorDirectAdapter(mockRepo, log)

	// Test: Validate with mismatched signature
	ctx := testutil.TestContext()
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD} // Different from endpoint.Sign

	result, err := adapter.ValidateDetachSignature(ctx, epEUI, detachSign)

	// Assert: Returns ErrDetachSignatureInvalid sentinel
	if !errors.Is(err, bssci.ErrDetachSignatureInvalid) {
		t.Errorf("Expected error to be ErrDetachSignatureInvalid, got: %v", err)
	}

	// Assert: Result is non-nil with tenant metadata
	if result == nil {
		t.Fatal("Expected non-nil result even on error")
	}
	if result.Valid {
		t.Error("Expected result.Valid to be false")
	}
	if result.TenantID != 123 {
		t.Errorf("Expected TenantID 123, got %d", result.TenantID)
	}
	if result.OwnerTenantID != 456 {
		t.Errorf("Expected OwnerTenantID 456, got %d", result.OwnerTenantID)
	}
	if result.ValidationStatus != bssci.ValidationStatusInvalidSignature {
		t.Errorf("Expected ValidationStatus '%s', got '%s'", bssci.ValidationStatusInvalidSignature, result.ValidationStatus)
	}
}

// TestDetachValidatorDirectAdapter_EndpointNotFound_ReturnsSentinel verifies 404 handling
func TestDetachValidatorDirectAdapter_EndpointNotFound_ReturnsSentinel(t *testing.T) {
	// Setup: Empty repo (endpoint not found)
	mockRepo := newTestDetachValidatorRepo()
	// Don't add any endpoint

	log := logger.NewNop()
	adapter := NewDetachValidatorDirectAdapter(mockRepo, log)

	// Test: Validate non-existent endpoint
	ctx := testutil.TestContext()
	epEUI := uint64(0x9999999999999999)
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	result, err := adapter.ValidateDetachSignature(ctx, epEUI, detachSign)

	// Assert: Returns ErrDetachValidationEndpointNotFound sentinel
	if !errors.Is(err, bssci.ErrDetachValidationEndpointNotFound) {
		t.Errorf("Expected error to be ErrDetachValidationEndpointNotFound, got: %v", err)
	}

	// Assert: Result is nil (endpoint doesn't exist)
	if result != nil {
		t.Errorf("Expected nil result for non-existent endpoint, got: %v", result)
	}
}

// TestDetachValidatorDirectAdapter_DatabaseError_ReturnsWrappedError verifies DB error handling
func TestDetachValidatorDirectAdapter_DatabaseError_ReturnsWrappedError(t *testing.T) {
	// Setup: Mock returns generic database error
	mockRepo := newTestDetachValidatorRepo()
	mockRepo.errorToReturn = errors.New("connection refused")

	log := logger.NewNop()
	adapter := NewDetachValidatorDirectAdapter(mockRepo, log)

	// Test: Validate with database error
	ctx := testutil.TestContext()
	epEUI := uint64(0x1122334455667788)
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	result, err := adapter.ValidateDetachSignature(ctx, epEUI, detachSign)

	// Assert: Returns wrapped error (not a sentinel)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if errors.Is(err, bssci.ErrDetachValidationEndpointNotFound) {
		t.Error("Should not be ErrDetachValidationEndpointNotFound")
	}
	if errors.Is(err, bssci.ErrDetachSignatureInvalid) {
		t.Error("Should not be ErrDetachSignatureInvalid")
	}

	// Assert: Result is nil
	if result != nil {
		t.Errorf("Expected nil result for database error, got: %v", result)
	}
}

// TestDetachValidatorDirectAdapter_ErrNotFound_TranslatedToSentinel verifies storage.ErrNotFound translation
func TestDetachValidatorDirectAdapter_ErrNotFound_TranslatedToSentinel(t *testing.T) {
	// Setup: Mock returns storage.ErrNotFound
	mockRepo := newTestDetachValidatorRepo()
	mockRepo.errorToReturn = storage.ErrNotFound

	log := logger.NewNop()
	adapter := NewDetachValidatorDirectAdapter(mockRepo, log)

	// Test: Validate with ErrNotFound error
	ctx := testutil.TestContext()
	epEUI := uint64(0x1122334455667788)
	detachSign := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	result, err := adapter.ValidateDetachSignature(ctx, epEUI, detachSign)

	// Assert: Returns ErrDetachValidationEndpointNotFound sentinel (translated)
	if !errors.Is(err, bssci.ErrDetachValidationEndpointNotFound) {
		t.Errorf("Expected error to be ErrDetachValidationEndpointNotFound, got: %v", err)
	}

	// Assert: Result is nil
	if result != nil {
		t.Errorf("Expected nil result for ErrNotFound, got: %v", result)
	}
}
