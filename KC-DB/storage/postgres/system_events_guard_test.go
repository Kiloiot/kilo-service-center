package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/kilocenter/KC-DB/storage/interfaces"
)

func TestGetEvents_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.GetEvents(context.Background(), interfaces.SystemEventFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestCountEvents_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.CountEvents(context.Background(), interfaces.SystemEventFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestCountActiveAlerts_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.CountActiveAlerts(context.Background(), interfaces.AlertFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestGetActiveAlerts_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.GetActiveAlerts(context.Background(), interfaces.AlertFilter{})
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestGetEventStats_EmptyTenantID_ReturnsError(t *testing.T) {
	store := NewSystemEventStore(nil)
	_, err := store.GetEventStats(context.Background(), "", time.Now())
	if err == nil {
		t.Fatal("expected error for empty tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}
