package grpc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	testOwnerTenant   = int64(42)
	testForeignTenant = int64(99)
	testOwnedBsEui    = "1122334455667788"
	testWrapNotFound  = "base station not found"
	testMaskName      = "name"
	testRenamedBsName = "renamed"

	testOwnedBaseStationID = int64(601)
)

// ownedBaseStations holds one base station for testOwnerTenant and answers
// every other tenant the way the repository does: storage.ErrNotFound wrapped
// in context.
type ownedBaseStations struct {
	mockBasestationSvcIsolation
}

func (o *ownedBaseStations) notFound() error {
	return fmt.Errorf("%s: %w", testWrapNotFound, storage.ErrNotFound)
}

func (o *ownedBaseStations) GetByEUI(_ context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	if tenantID != testOwnerTenant {
		return nil, o.notFound()
	}
	bs := &models.BaseStation{ID: testOwnedBaseStationID, TenantID: tenantID}
	copy(bs.EUI[:], eui)
	return bs, nil
}

func (o *ownedBaseStations) Delete(_ context.Context, eui []byte, tenantID int64) (*models.BaseStation, error) {
	if tenantID != testOwnerTenant {
		return nil, o.notFound()
	}
	bs := &models.BaseStation{ID: testOwnedBaseStationID, TenantID: tenantID}
	copy(bs.EUI[:], eui)
	return bs, nil
}

func TestBaseStationHandlers_ForeignTenantGetsNotFound(t *testing.T) {
	svc := testCoreService(coreFields{basestationSvc: &ownedBaseStations{}, log: &mockLogger{}})

	calls := map[string]func(ctx context.Context) error{
		"GetBaseStation": func(ctx context.Context) error {
			_, err := svc.GetBaseStation(ctx, &pb.GetBaseStationRequest{BsEui: testOwnedBsEui})
			return err
		},
		"UpdateBaseStation": func(ctx context.Context) error {
			_, err := svc.UpdateBaseStation(ctx, &pb.UpdateBaseStationRequest{
				Basestation: &pb.BaseStation{BsEui: testOwnedBsEui, Name: testRenamedBsName},
				UpdateMask:  &fieldmaskpb.FieldMask{Paths: []string{testMaskName}},
			})
			return err
		},
		"DeleteBaseStation": func(ctx context.Context) error {
			_, err := svc.DeleteBaseStation(ctx, &pb.DeleteBaseStationRequest{BsEui: testOwnedBsEui})
			return err
		},
		"RequestBaseStationStatus": func(ctx context.Context) error {
			_, err := svc.RequestBaseStationStatus(ctx, &pb.BaseStationStatusRequest{BsEuiHex: testOwnedBsEui})
			return err
		},
		"InitiatePing": func(ctx context.Context) error {
			_, err := svc.InitiatePing(ctx, &pb.InitiatePingRequest{BsEuiHex: testOwnedBsEui})
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call(contextForTenant(testForeignTenant))
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code(), "a base station of another tenant must read as absent: %v", err)
		})
	}

	_, err := svc.GetBaseStation(contextForTenant(testOwnerTenant), &pb.GetBaseStationRequest{BsEui: testOwnedBsEui})
	require.NoError(t, err, "the owning tenant still reads its base station")
}
