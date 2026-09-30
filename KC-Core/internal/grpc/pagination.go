package grpc

import (
	"math"

	"google.golang.org/grpc/status"

	grpchelpers "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// Pagination defaults — delegated to KC-Core/pkg/grpc for cross-service use.
const (
	DefaultPageSize           = grpchelpers.DefaultPageSize
	MaxPageSize               = grpchelpers.MaxPageSize
	DefaultHighVolumePageSize = grpchelpers.DefaultHighVolumePageSize
)

// Downlink result and DL RX status pagination policy.
const (
	// DefaultDownlinkResultsPageSize is the page size applied when
	// GetDownlinkResults receives no or a non-positive page size.
	DefaultDownlinkResultsPageSize = 50
	// MaxDownlinkResultsPageSize caps the GetDownlinkResults page size.
	MaxDownlinkResultsPageSize = 500
	// DefaultDLRXStatusLimit is the result limit applied when GetDLRXStatus
	// receives no or a non-positive limit.
	DefaultDLRXStatusLimit = 100
	// DefaultActivityPageSize is the page size applied when activity list
	// RPCs receive no or an out-of-range page size.
	DefaultActivityPageSize = 50
)

// clampPageSize delegates to the shared package.
func clampPageSize(requested int32) int {
	return grpchelpers.ClampPageSize(requested)
}

// clampHighVolumePageSize delegates to the shared package.
func clampHighVolumePageSize(requested int32, defaultSize int32) int32 {
	return grpchelpers.ClampHighVolumePageSize(requested, defaultSize)
}

// clampDownlinkResultsPageSize bounds a GetDownlinkResults page size.
func clampDownlinkResultsPageSize(requested int32) int {
	switch {
	case requested <= 0:
		return DefaultDownlinkResultsPageSize
	case requested > MaxDownlinkResultsPageSize:
		return MaxDownlinkResultsPageSize
	default:
		return int(requested)
	}
}

// pageRequest is the page of a listing a request asks for. The page size is
// the one the listing's policy allowed; the stores page exactly that.
type pageRequest struct {
	limit  int
	offset int
}

// readPage reads a listing's page from the page size its policy allowed and
// the request's token; a token that is not a non-negative offset, or whose
// next page's offset would overflow, is refused.
func readPage(limit int, token string) (pageRequest, error) {
	offset := 0
	if _, err := grpchelpers.ParsePaginationToken(token, &offset); err != nil || offset < 0 || offset > math.MaxInt-limit {
		return pageRequest{}, status.Error(grpchelpers.GetGRPCCode(grpchelpers.ErrTokenInvalidPageToken),
			grpchelpers.ResolveErrorMessage(grpchelpers.ErrTokenInvalidPageToken))
	}
	return pageRequest{limit: limit, offset: offset}, nil
}

// pageResponse is where a served page stands in its listing.
type pageResponse struct {
	nextToken  string
	totalCount int32
}

// respond places the page among total matching rows: the next page's token
// while rows remain after it, and the total, which the API carries as int32.
func (p pageRequest) respond(total int64) (pageResponse, error) {
	totalCount, err := totalCountOf(total)
	if err != nil {
		return pageResponse{}, err
	}
	response := pageResponse{totalCount: totalCount}
	if next := p.offset + p.limit; int64(next) < total {
		response.nextToken = grpchelpers.GeneratePaginationToken(next)
	}
	return response, nil
}

// totalCountOf carries a count in the int32 the API declares, refusing a
// count that does not fit.
func totalCountOf(total int64) (int32, error) {
	if total < 0 || total > math.MaxInt32 {
		return 0, status.Error(grpchelpers.GetGRPCCode(grpchelpers.ErrTokenResultCountOverflow),
			grpchelpers.ResolveErrorMessage(grpchelpers.ErrTokenResultCountOverflow))
	}
	return int32(total), nil
}
