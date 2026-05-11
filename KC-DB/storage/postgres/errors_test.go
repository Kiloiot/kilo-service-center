package postgres

import (
	"errors"
	"testing"

	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/lib/pq"
)

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unique violation",
			err:  &pq.Error{Code: "23505"},
			want: true,
		},
		{
			name: "foreign key violation",
			err:  &pq.Error{Code: "23503"},
			want: false,
		},
		{
			name: "not null violation",
			err:  &pq.Error{Code: "23502"},
			want: false,
		},
		{
			name: "non-pq error",
			err:  errors.New("generic error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUniqueViolation(tt.err); got != tt.want {
				t.Errorf("IsUniqueViolation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrapDuplicateError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		resource string
		wantType error
		wantMsg  string
	}{
		{
			name:     "unique violation wrapped",
			err:      &pq.Error{Code: "23505"},
			resource: "endpoint",
			wantType: kcerrors.ErrDuplicate,
			wantMsg:  "endpoint already exists",
		},
		{
			name:     "other pq error unwrapped",
			err:      &pq.Error{Code: "23503"},
			resource: "base station",
			wantType: nil,
			wantMsg:  "",
		},
		{
			name:     "generic error unwrapped",
			err:      errors.New("some error"),
			resource: "test",
			wantType: nil,
			wantMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapDuplicateError(tt.err, tt.resource)

			// Check if it wraps ErrDuplicate when expected
			if tt.wantType != nil {
				if !errors.Is(got, tt.wantType) {
					t.Errorf("WrapDuplicateError() error = %v, want to wrap %v", got, tt.wantType)
				}
			}

			// Check error message contains expected text
			if tt.wantMsg != "" {
				if got == nil || !contains(got.Error(), tt.wantMsg) {
					t.Errorf("WrapDuplicateError() message = %v, want to contain %v", got, tt.wantMsg)
				}
			}

			// Check that non-unique errors are returned as-is
			if tt.wantType == nil && got != tt.err {
				t.Errorf("WrapDuplicateError() should return original error for non-unique violations")
			}
		})
	}
}

func TestWrapDuplicateError_PreservesErrorChain(t *testing.T) {
	uniqueErr := &pq.Error{
		Code:       "23505",
		Constraint: "unique_ep_eui",
		Table:      "endpoints",
	}
	wrapped := WrapDuplicateError(uniqueErr, "endpoint")

	// **TEST 1**: Assert errors.Is still works (tests first %w)
	if !errors.Is(wrapped, kcerrors.ErrDuplicate) {
		t.Error("Expected errors.Is(wrapped, ErrDuplicate) to return true - first %w failed")
	}

	// **TEST 2**: Assert original error is preserved (tests second %w)
	var originalPqErr *pq.Error
	if !errors.As(wrapped, &originalPqErr) {
		t.Error("Expected errors.As(wrapped, &pqErr) to return true - second %w failed")
	}
	if originalPqErr.Constraint != "unique_ep_eui" {
		t.Errorf("Expected constraint 'unique_ep_eui', got %q", originalPqErr.Constraint)
	}

	// **TEST 3**: Assert error message includes constraint details
	msg := wrapped.Error()
	if !contains(msg, "endpoint") {
		t.Error("Expected error message to include resource name")
	}
	if !contains(msg, "already exists") {
		t.Error("Expected error message to include 'already exists'")
	}
	if !contains(msg, "unique_ep_eui") {
		t.Errorf("Expected constraint name in error, got: %s", msg)
	}
	if !contains(msg, "endpoints") {
		t.Errorf("Expected table name in error, got: %s", msg)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
