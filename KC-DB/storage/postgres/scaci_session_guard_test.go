package postgres

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"

	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

func TestSCACIListSessions_NilTenantID_ReturnsError(t *testing.T) {
	repo := &SCACISessionRepository{clock: clock.SystemClock{}, db: nil}
	_, _, err := repo.ListSessions(testutil.TestContext(), &models.SCACISessionFilter{})
	if err == nil {
		t.Fatal("expected error for nil tenant ID, got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}

func TestSCACIListSessions_NilFilter_ReturnsError(t *testing.T) {
	repo := &SCACISessionRepository{clock: clock.SystemClock{}, db: nil}
	_, _, err := repo.ListSessions(testutil.TestContext(), nil)
	if err == nil {
		t.Fatal("expected error for nil filter (nil tenant ID), got nil")
	}
	if err.Error() != "tenant ID required" {
		t.Fatalf("expected 'tenant ID required', got %q", err.Error())
	}
}
