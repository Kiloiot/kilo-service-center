package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// encodeBaseStationTags renders tags as the JSON object the tags column
// stores; a station without tags stores none.
func encodeBaseStationTags(tags map[string]string) (*string, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	encoded, err := json.Marshal(tags)
	if err != nil {
		return nil, fmt.Errorf(errFmtEncodeBaseStationTags, err)
	}
	stored := string(encoded)
	return &stored, nil
}

// decodeBaseStationTags reads the stored tags JSON object back.
func decodeBaseStationTags(stored *string) (map[string]string, error) {
	if stored == nil || *stored == "" {
		return nil, nil
	}
	var tags map[string]string
	if err := json.Unmarshal([]byte(*stored), &tags); err != nil {
		return nil, fmt.Errorf(errFmtDecodeBaseStationTags, err)
	}
	return tags, nil
}

// baseStationResponse converts a stored base station for a response; a
// station that cannot be converted fails the call as an internal error.
func (s *BaseStationHandlers) baseStationResponse(ctx context.Context, baseStation *models.BaseStation) (*pb.BaseStation, error) {
	converted, err := baseStationToProto(baseStation)
	if err != nil {
		return nil, s.internalFailure(ctx, LogFailedToDecodeBaseStationTags, err)
	}
	return converted, nil
}

// internalFailure logs err under msg and returns the catalog's internal error.
func (s *BaseStationHandlers) internalFailure(ctx context.Context, msg string, err error) error {
	s.log.ErrorContext(ctx, msg, logger.FieldError, err)
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
}
