package org

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kilocenter/KC-DB/storage"
)

var (
	testDefaultTenantID = int64(1)
	testDefaultOrgUUID  = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
)

func newTestCommunityResolver() *CommunityResolver {
	return NewCommunityResolver(testDefaultTenantID, testDefaultOrgUUID)
}

func TestCommunityResolver_LookupTenant_DefaultOrg(t *testing.T) {
	r := newTestCommunityResolver()
	tenantID, err := r.LookupTenant(context.Background(), testDefaultOrgUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenantID != testDefaultTenantID {
		t.Fatalf("expected tenant %d, got %d", testDefaultTenantID, tenantID)
	}
}

func TestCommunityResolver_LookupTenant_NonDefaultOrg(t *testing.T) {
	r := newTestCommunityResolver()
	otherOrg := uuid.New()
	_, err := r.LookupTenant(context.Background(), otherOrg)
	if err == nil {
		t.Fatal("expected error for non-default org")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestCommunityResolver_ResolveCert(t *testing.T) {
	r := newTestCommunityResolver()
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "test-cert"}}
	orgUUID, tenantID, err := r.ResolveCert(context.Background(), cert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgUUID != testDefaultOrgUUID {
		t.Fatalf("expected org %s, got %s", testDefaultOrgUUID, orgUUID)
	}
	if tenantID != testDefaultTenantID {
		t.Fatalf("expected tenant %d, got %d", testDefaultTenantID, tenantID)
	}
}

func TestCommunityResolver_ResolveCert_NilCert(t *testing.T) {
	r := newTestCommunityResolver()
	_, _, err := r.ResolveCert(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil cert")
	}
}

func TestCommunityResolver_GetDefaultOrgForTenant_DefaultTenant(t *testing.T) {
	r := newTestCommunityResolver()
	orgUUID, err := r.GetDefaultOrgForTenant(context.Background(), testDefaultTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgUUID != testDefaultOrgUUID {
		t.Fatalf("expected org %s, got %s", testDefaultOrgUUID, orgUUID)
	}
}

func TestCommunityResolver_GetDefaultOrgForTenant_OtherTenant(t *testing.T) {
	r := newTestCommunityResolver()
	_, err := r.GetDefaultOrgForTenant(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for non-default tenant")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestCommunityResolver_ResolveOrgByExternalID(t *testing.T) {
	r := newTestCommunityResolver()
	orgUUID, err := r.ResolveOrgByExternalID(context.Background(), "some-external-id")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
	if orgUUID != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %s", orgUUID)
	}
}

func TestCommunityResolver_ConcurrentAccess(_ *testing.T) {
	r := newTestCommunityResolver()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_, _ = r.LookupTenant(ctx, testDefaultOrgUUID)
		}()
		go func() {
			defer wg.Done()
			_, _ = r.GetDefaultOrgForTenant(ctx, testDefaultTenantID)
		}()
		go func() {
			defer wg.Done()
			_, _ = r.ResolveOrgByExternalID(ctx, "ext-id")
		}()
	}
	wg.Wait()
}
