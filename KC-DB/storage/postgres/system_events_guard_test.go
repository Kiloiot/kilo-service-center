package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

func TestGetEvents_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil, clock.SystemClock{}, logger.Get())
	_, err := store.GetEvents(testutil.TestContext(), models.SystemEventFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestCountEvents_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil, clock.SystemClock{}, logger.Get())
	_, err := store.CountEvents(testutil.TestContext(), models.SystemEventFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestCountActiveAlerts_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil, clock.SystemClock{}, logger.Get())
	_, err := store.CountActiveAlerts(testutil.TestContext(), models.AlertFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestGetActiveAlerts_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil, clock.SystemClock{}, logger.Get())
	_, err := store.GetActiveAlerts(testutil.TestContext(), models.AlertFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestCountAlertsBySeverity_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil, clock.SystemClock{}, logger.Get())
	_, err := store.CountAlertsBySeverity(testutil.TestContext(), models.AlertFilter{})
	if !errors.Is(err, errTextTenantIDRequired) {
		t.Fatalf("expected the tenant ID guard, got %v", err)
	}
}

func TestGetEventStats_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil, clock.SystemClock{}, logger.Get())
	_, err := store.GetEventStats(testutil.TestContext(), "", time.Now())
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}
