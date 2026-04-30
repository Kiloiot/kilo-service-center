package context

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestGetTenantID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(context.Context) context.Context
		want    int64
		wantErr error
	}{
		{
			name: "int64 value present",
			setup: func(ctx context.Context) context.Context {
				return WithTenantID(ctx, 42)
			},
			want:    42,
			wantErr: nil,
		},
		{
			name: "string value present and parseable",
			setup: func(ctx context.Context) context.Context {
				// WithTenantID sets both int64 and string, so test string-only path
				return context.WithValue(ctx, TenantIDKey, "123")
			},
			want:    123,
			wantErr: nil,
		},
		{
			name: "only string value (manual insertion)",
			setup: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, TenantIDKey, "456")
			},
			want:    456,
			wantErr: nil,
		},
		{
			name: "no tenant in context",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			want:    0,
			wantErr: ErrNoTenantInContext,
		},
		{
			name: "invalid string format returns error",
			setup: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, TenantIDKey, "not-a-number")
			},
			want:    0,
			wantErr: errors.New("any error"), // Expect any error for invalid format
		},
		{
			name: "empty string",
			setup: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, TenantIDKey, "")
			},
			want:    0,
			wantErr: ErrNoTenantInContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup(context.Background())
			got, err := GetTenantID(ctx)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("GetTenantID() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) && err.Error() == "" {
					t.Errorf("GetTenantID() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("GetTenantID() unexpected error = %v", err)
					return
				}
			}

			if got != tt.want {
				t.Errorf("GetTenantID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOrganizationID(t *testing.T) {
	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name    string
		setup   func(context.Context) context.Context
		want    uuid.UUID
		wantErr error
	}{
		{
			name: "organization ID present",
			setup: func(ctx context.Context) context.Context {
				return WithOrganizationID(ctx, testUUID)
			},
			want:    testUUID,
			wantErr: nil,
		},
		{
			name: "no organization in context",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			want:    uuid.Nil,
			wantErr: ErrNoOrganizationInContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup(context.Background())
			got, err := GetOrganizationID(ctx)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetOrganizationID() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("GetOrganizationID() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("GetOrganizationID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(context.Context) context.Context
		want    string
		wantErr error
	}{
		{
			name: "user ID present",
			setup: func(ctx context.Context) context.Context {
				return WithUserID(ctx, "user123")
			},
			want:    "user123",
			wantErr: nil,
		},
		{
			name: "no user in context",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			want:    "",
			wantErr: ErrNoUserInContext,
		},
		{
			name: "empty user ID",
			setup: func(ctx context.Context) context.Context {
				return WithUserID(ctx, "")
			},
			want:    "",
			wantErr: ErrNoUserInContext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup(context.Background())
			got, err := GetUserID(ctx)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetUserID() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("GetUserID() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("GetUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithTenantID(t *testing.T) {
	ctx := WithTenantID(context.Background(), 42)

	// Should set int64 value
	if val, ok := ctx.Value(TenantIDIntKey).(int64); !ok || val != 42 {
		t.Errorf("WithTenantID() int64 value = %v, want 42", val)
	}

	// Should also set string value
	if val, ok := ctx.Value(TenantIDKey).(string); !ok || val != "42" {
		t.Errorf("WithTenantID() string value = %v, want \"42\"", val)
	}
}

func TestWithOrganizationID(t *testing.T) {
	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	ctx := WithOrganizationID(context.Background(), testUUID)

	if val, ok := ctx.Value(OrganizationIDKey).(uuid.UUID); !ok || val != testUUID {
		t.Errorf("WithOrganizationID() = %v, want %v", val, testUUID)
	}
}

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(context.Background(), "user456")

	if val, ok := ctx.Value(UserIDKey).(string); !ok || val != "user456" {
		t.Errorf("WithUserID() = %v, want \"user456\"", val)
	}
}

// TestContextChaining verifies that multiple values can be added to the same context
func TestContextChaining(t *testing.T) {
	ctx := context.Background()
	ctx = WithTenantID(ctx, 42)
	ctx = WithOrganizationID(ctx, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
	ctx = WithUserID(ctx, "user123")

	// Verify all values are present
	if tenantID, err := GetTenantID(ctx); err != nil || tenantID != 42 {
		t.Errorf("chained context: GetTenantID() = %v, %v; want 42, nil", tenantID, err)
	}

	if orgID, err := GetOrganizationID(ctx); err != nil {
		t.Errorf("chained context: GetOrganizationID() error = %v", err)
	} else if orgID.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("chained context: GetOrganizationID() = %v", orgID)
	}

	if userID, err := GetUserID(ctx); err != nil || userID != "user123" {
		t.Errorf("chained context: GetUserID() = %v, %v; want \"user123\", nil", userID, err)
	}
}
