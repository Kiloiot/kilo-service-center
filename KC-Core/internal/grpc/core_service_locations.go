package grpc

import (
	"context"
	"encoding/hex"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// ListAllBaseStationLocations returns locations of all base stations across all tenants.
// Restricted to server admins (user.IsAdmin=true).
func (s *CoreService) ListAllBaseStationLocations(ctx context.Context, _ *pb.ListAllBaseStationLocationsRequest) (*pb.ListAllBaseStationLocationsResponse, error) {
	// Extract caller identity
	userID, err := GetUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Verify server-level admin
	if s.adminChecker == nil {
		s.log.ErrorContext(ctx, "admin checker not configured")
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	isAdmin, err := s.adminChecker.IsServerAdmin(ctx, userID.String())
	if err != nil {
		s.log.ErrorContext(ctx, "server admin check failed", "user_id", userID, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}
	if !isAdmin {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenAdminRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenAdminRequired))
	}

	// Fetch all base stations with coordinates
	stations, err := s.basestationSvc.ListAllLocations(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to list all base station locations", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	// Collect unique tenant IDs and batch-resolve to org UUIDs
	tenantOrgMap := make(map[int64]string)
	for _, bs := range stations {
		if _, ok := tenantOrgMap[bs.TenantID]; !ok {
			tenantOrgMap[bs.TenantID] = ""
		}
	}

	if s.orgMapper == nil {
		s.log.ErrorContext(ctx, "org mapper not configured")
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	for tid := range tenantOrgMap {
		orgID, mapErr := s.orgMapper.GetDefaultOrgForTenant(ctx, tid)
		if mapErr != nil {
			s.log.ErrorContext(ctx, "failed to map tenant to org", "tenant_id", tid, "error", mapErr)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
		}
		tenantOrgMap[tid] = orgID
	}

	// Build response
	locations := make([]*pb.BaseStationLocation, 0, len(stations))
	for _, bs := range stations {
		loc := &pb.BaseStationLocation{
			BsEui:    hex.EncodeToString(bs.EUI[:]),
			Name:     bs.Name,
			IsOnline: bs.IsOnline,
			OrgId:    tenantOrgMap[bs.TenantID],
		}

		if bs.Latitude != nil {
			loc.Latitude = *bs.Latitude
		}
		if bs.Longitude != nil {
			loc.Longitude = *bs.Longitude
		}
		if bs.Altitude != nil {
			loc.Altitude = wrapperspb.Double(*bs.Altitude)
		}
		if bs.LocationSource != nil {
			loc.LocationSource = *bs.LocationSource
		}

		locations = append(locations, loc)
	}

	return &pb.ListAllBaseStationLocationsResponse{
		Locations:  locations,
		TotalCount: int32(len(locations)), //nolint:gosec // location count is bounded by DB query, no overflow risk
	}, nil
}
