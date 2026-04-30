package proxy

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeConn implements grpc.ClientConnInterface for routing assertions.
type fakeConn struct {
	name string
}

func (f *fakeConn) Invoke(_ context.Context, _ string, _ interface{}, _ interface{}, _ ...grpc.CallOption) error {
	return nil
}

func (f *fakeConn) NewStream(_ context.Context, _ *grpc.StreamDesc, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func TestSelectUpstream_IdentityServiceRoutesToIdentity(t *testing.T) {
	core := &fakeConn{name: "core"}
	identity := &fakeConn{name: "identity"}

	got, err := SelectUpstream("/kilocenter.api.v1.IdentityService/Login", core, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != identity {
		t.Errorf("expected identity conn, got %v", got)
	}
}

func TestSelectUpstream_CoreServiceRoutesToCore(t *testing.T) {
	core := &fakeConn{name: "core"}
	identity := &fakeConn{name: "identity"}

	got, err := SelectUpstream("/kilocenter.api.v1.CoreService/CreateEndPoint", core, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != core {
		t.Errorf("expected core conn, got %v", got)
	}
}

func TestSelectUpstream_CompatIdentityMethodRoutesToIdentity(t *testing.T) {
	core := &fakeConn{name: "core"}
	identity := &fakeConn{name: "identity"}

	got, err := SelectUpstream("/kilocenter.api.v1.KiloCenterService/Login", core, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != identity {
		t.Errorf("expected identity conn for compat Login, got %v", got)
	}
}

func TestSelectUpstream_CompatCoreMethodRoutesToCore(t *testing.T) {
	core := &fakeConn{name: "core"}
	identity := &fakeConn{name: "identity"}

	got, err := SelectUpstream("/kilocenter.api.v1.KiloCenterService/CreateEndPoint", core, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != core {
		t.Errorf("expected core conn for compat CreateEndPoint, got %v", got)
	}
}

func TestSelectUpstream_IdentityInternalServiceDenied(t *testing.T) {
	core := &fakeConn{name: "core"}
	identity := &fakeConn{name: "identity"}

	got, err := SelectUpstream("/kilocenter.api.v1.IdentityInternalService/ResolveOrg", core, identity)
	if got != nil {
		t.Errorf("expected nil conn for denied internal service, got %v", got)
	}
	if err == nil {
		t.Fatal("expected error for IdentityInternalService, got nil")
	}
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("expected Unimplemented, got %v", status.Code(err))
	}
}

func TestSelectUpstream_UnknownServiceFallsToCore(t *testing.T) {
	core := &fakeConn{name: "core"}
	identity := &fakeConn{name: "identity"}

	got, err := SelectUpstream("/foo.Bar/Baz", core, identity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != core {
		t.Errorf("expected core conn for unknown service fallthrough, got %v", got)
	}
}

func TestSelectUpstream_AllCompatIdentityMethods(t *testing.T) {
	core := &fakeConn{name: "core"}
	identity := &fakeConn{name: "identity"}

	for method := range compatIdentityMethods {
		got, err := SelectUpstream(method, core, identity)
		if err != nil {
			t.Errorf("method %s: unexpected error: %v", method, err)
			continue
		}
		if got != identity {
			t.Errorf("method %s: expected identity conn, got core", method)
		}
	}
}
