package mqtt

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct{}

func (m *MockLogger) Debug(_ string, _ ...interface{})                           {}
func (m *MockLogger) Info(_ string, _ ...interface{})                            {}
func (m *MockLogger) Warn(_ string, _ ...interface{})                            {}
func (m *MockLogger) Error(_ string, _ ...interface{})                           {}
func (m *MockLogger) Fatal(_ string, _ ...interface{})                           {}
func (m *MockLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *MockLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *MockLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (m *MockLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *MockLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (m *MockLogger) WithField(_ string, _ interface{}) logger.Logger {
	return m
}
func (m *MockLogger) WithFields(_ map[string]interface{}) logger.Logger {
	return m
}

// testConfig returns a minimal MQTT configuration for testing.
// Uses public KC-Core/pkg/config package.
func testConfig() *config.MQTTConfig {
	return &config.MQTTConfig{
		Host:                 "localhost",
		Port:                 1883,
		ClientID:             "kilocenter-test",
		KeepAlive:            60,
		CleanSession:         true,
		MaxReconnectInterval: 300,
		TLS: config.TLSConfig{
			Enabled: false,
		},
	}
}

// TestNewClient verifies client creation with valid config and config/logger retention
func TestNewClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping MQTT config assertions in short mode")
	}

	cfg := testConfig()
	log := &MockLogger{}

	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Cast to concrete type to verify internal config fields
	concreteClient, ok := client.(*Client)
	if !ok {
		t.Fatal("NewClient() did not return *Client type")
	}

	// Verify all accessible config fields are retained correctly
	if concreteClient.config.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", concreteClient.config.Host)
	}
	if concreteClient.config.Port != 1883 {
		t.Errorf("Expected port 1883, got %d", concreteClient.config.Port)
	}
	if concreteClient.config.ClientID != "kilocenter-test" {
		t.Errorf("Expected ClientID 'kilocenter-test', got '%s'", concreteClient.config.ClientID)
	}
	if concreteClient.config.KeepAlive != 60 {
		t.Errorf("Expected KeepAlive 60, got %d", concreteClient.config.KeepAlive)
	}
	if !concreteClient.config.CleanSession {
		t.Error("Expected CleanSession true")
	}
	if concreteClient.config.MaxReconnectInterval != 300 {
		t.Errorf("Expected MaxReconnectInterval 300, got %d", concreteClient.config.MaxReconnectInterval)
	}
	if concreteClient.config.TLS.Enabled {
		t.Error("Expected TLS.Enabled false")
	}

	// Verify client is not connected (no broker needed for this test)
	if client.IsConnected() {
		t.Error("Expected client to not be connected without calling Connect()")
	}
}

// TestClientNilConfig verifies error handling for nil config
func TestClientNilConfig(t *testing.T) {
	log := &MockLogger{}

	client, err := NewClient(nil, log)
	if err == nil {
		t.Fatal("Expected error for nil config, got nil")
	}
	if client != nil {
		t.Fatal("Expected nil client for nil config")
	}
}

// TestPublishNotConnected verifies publish fails when not connected
func TestPublishNotConnected(t *testing.T) {
	cfg := testConfig()
	log := &MockLogger{}

	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Attempt publish without connecting (use QoS 0, not retained)
	err = client.Publish(testutil.TestContext(), "test/topic", 0, false, []byte("test payload"))
	if err == nil {
		t.Error("Expected error when publishing without connection")
	}
}

// TestSubscribeNotConnected verifies subscribe fails when not connected
func TestSubscribeNotConnected(t *testing.T) {
	cfg := testConfig()
	log := &MockLogger{}

	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Attempt subscribe without connecting (use QoS 0, nil handler is OK for test)
	err = client.Subscribe(testutil.TestContext(), "test/topic", 0, nil)
	if err == nil {
		t.Error("Expected error when subscribing without connection")
	}
}

// TestPublishNotConnected_WithHandler verifies "not connected" error with actual payload.
// This test ensures the connection guard is hit, not a nil-handler fast path.
func TestPublishNotConnected_WithHandler(t *testing.T) {
	cfg := testConfig()
	log := &MockLogger{}

	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Attempt publish without connecting (real payload, not nil)
	payload := []byte("test message payload")
	err = client.Publish(testutil.TestContext(), "mioty/uplink/test", 0, false, payload)

	// Must fail with "not connected" error (proves connection guard is triggered)
	if err == nil {
		t.Error("Expected error when publishing without connection")
	}
	if err != nil && err.Error() != "mqtt client not connected" {
		t.Errorf("Expected 'mqtt client not connected' error, got: %v", err)
	}
}

// TestSubscribeNotConnected_WithHandler verifies "not connected" error with actual handler.
// This test ensures the connection guard is hit, not a handler-validation fast path.
func TestSubscribeNotConnected_WithHandler(t *testing.T) {
	cfg := testConfig()
	log := &MockLogger{}

	client, err := NewClient(cfg, log)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Attempt subscribe without connecting (with real handler, not nil)
	handlerCalled := false
	handler := func(_ context.Context, _ string, _ []byte) {
		handlerCalled = true
	}

	err = client.Subscribe(testutil.TestContext(), "mioty/uplink/#", 0, handler)

	// Must fail with "not connected" error (proves connection guard is triggered, not handler validation)
	if err == nil {
		t.Error("Expected error when subscribing without connection")
	}
	if err != nil && err.Error() != "mqtt client not connected" {
		t.Errorf("Expected 'mqtt client not connected' error, got: %v", err)
	}
	if handlerCalled {
		t.Error("Handler should not have been called without connection")
	}
}
