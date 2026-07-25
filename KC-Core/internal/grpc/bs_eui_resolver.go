package grpc

import (
	"google.golang.org/grpc/status"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
)

// resolveBaseStationEUI resolves the target base station EUI from a request that
// carries both the browser-safe hex field and the deprecated uint64 field.
// The hex form exists because JavaScript clients cannot represent EUIs above
// 2^53 as numbers; when both fields are set they must refer to the same EUI.
func resolveBaseStationEUI(bsEuiHex string, legacyBsEui uint64) (uint64, error) {
	if bsEuiHex == "" {
		if legacyBsEui == 0 {
			return 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
		}
		return legacyBsEui, nil
	}

	parsed, err := validation.ParseEUI(bsEuiHex)
	if err != nil {
		return 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	if legacyBsEui != 0 && legacyBsEui != parsed {
		return 0, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationEUIMismatch),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationEUIMismatch))
	}

	return parsed, nil
}
