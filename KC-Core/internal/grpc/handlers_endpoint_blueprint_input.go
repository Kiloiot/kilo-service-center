package grpc

import (
	"encoding/json"
	"time"

	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	pkgblueprint "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// inputSnapshot checks the request's blueprint_input the way the decoder
// checks a specification and builds the snapshot it pins. A request without
// input, or with an empty specification, yields no snapshot.
func inputSnapshot(ep *pb.EndPoint, adoptedAt time.Time) (json.RawMessage, error) {
	spec := ep.GetBlueprintInput().GetSpec()
	if len(spec) == 0 {
		return nil, nil
	}
	if ep.GetBlueprintId() != "" || ep.GetDeviceModelId() != "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInputConflict),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInputConflict))
	}
	parsed, err := pkgblueprint.ParseAndValidate(spec)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInputSpecInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInputSpecInvalid)+": "+err.Error())
	}
	// Validate only requires a non-empty typeEui; the snapshot needs the 8-byte EUI.
	typeEUI, err := pkgblueprint.TypeEUIFromSpec(spec)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintInputTypeEUIInvalid),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintInputTypeEUIInvalid))
	}
	raw, err := buildSpecSnapshot(spec, parsed.Version, typeEUI, adoptedAt)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBlueprintSnapshotBuildFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBlueprintSnapshotBuildFailed))
	}
	return raw, nil
}

// pinInputSnapshot pins a blueprint_input snapshot: the endpoint leaves its
// KC model and takes its type EUI from the specification.
func pinInputSnapshot(endpoint *models.EndPoint, snapshot json.RawMessage) {
	endpoint.DeviceModelID = nil
	endpoint.BlueprintSnapshot = snapshot
	endpoint.TypeEUI = snapshotTypeEUI(snapshot)
}
