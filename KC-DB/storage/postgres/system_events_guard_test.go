package postgres

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

func TestGetEvents_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.GetEvents(testutil.TestContext(), interfaces.SystemEventFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestCountEvents_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.CountEvents(testutil.TestContext(), interfaces.SystemEventFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestCountActiveAlerts_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.CountActiveAlerts(testutil.TestContext(), interfaces.AlertFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestGetActiveAlerts_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.GetActiveAlerts(testutil.TestContext(), interfaces.AlertFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestGetEventStats_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.GetEventStats(testutil.TestContext(), "", time.Now())
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}
