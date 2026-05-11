package postgres

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func TestSCACIListSessions_NilTenantID_ReturnsError(t *testing.T) {
	repo := &SCACISessionRepository{db: nil}
	_, _, err := repo.ListSessions(context.Background(), &models.SCACISessionFilter{})
	if err == nil {
		t.Fatal("expected error for nil tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestSCACIListSessions_NilFilter_ReturnsError(t *testing.T) {
	repo := &SCACISessionRepository{db: nil}
	_, _, err := repo.ListSessions(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil filter (nil tenant ID), got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}
