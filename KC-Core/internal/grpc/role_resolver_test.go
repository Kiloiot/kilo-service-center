package grpc

import (
	"context"
	"net"
	"testing"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/kilocenter/KC-Core/pkg/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type mockIdentityServer struct {
	pb.UnimplementedIdentityInternalServiceServer
	receivedMetadata metadata.MD
}

func (m *mockIdentityServer) GetUserMembership(ctx context.Context, _ *pb.GetUserMembershipRequest) (*pb.GetUserMembershipResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	m.receivedMetadata = md
	return &pb.GetUserMembershipResponse{Role: "admin", IsActive: true}, nil
}

func TestRoleResolver_SendsPeerSecretHeader(t *testing.T) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	mock := &mockIdentityServer{}
	srv := grpc.NewServer()
	pb.RegisterIdentityInternalServiceServer(srv, mock)

	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Errorf("server exited with error: %v", err)
		}
	}()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := pb.NewIdentityInternalServiceClient(conn)
	resolver := NewRoleResolver(client, "test-secret")

	role, active, err := resolver.GetUserRole(context.Background(), "org-1", "user-1")
	if err != nil {
		t.Fatalf("GetUserRole returned error: %v", err)
	}

	if role != "admin" {
		t.Errorf("expected role %q, got %q", "admin", role)
	}
	if !active {
		t.Error("expected active to be true, got false")
	}

	got := mock.receivedMetadata[grpcconst.MetadataKeyInternalPeerSecret]
	want := []string{"test-secret"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("expected peer-secret header %v, got %v", want, got)
	}
}
