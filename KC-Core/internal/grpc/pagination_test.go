package grpc

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

const (
	pageTestLimit = 20
	pageTestTotal = int64(45)
)

func TestReadPage_TokenIsTheOffset(t *testing.T) {
	page, err := readPage(pageTestLimit, "")
	require.NoError(t, err)
	assert.Equal(t, pageRequest{limit: pageTestLimit}, page, "no token starts at the top")

	page, err = readPage(pageTestLimit, grpcerrors.GeneratePaginationToken(40))
	require.NoError(t, err)
	assert.Equal(t, pageRequest{limit: pageTestLimit, offset: 40}, page)
}

func TestReadPage_RefusesATokenThatIsNoOffset(t *testing.T) {
	for _, token := range []string{"page-2", "-20", "1.5"} {
		_, err := readPage(pageTestLimit, token)
		require.Error(t, err, token)
		assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken), status.Code(err), token)
	}
}

// TestReadPage_RefusesAnOffsetWithoutANextPage: an offset whose next page
// would overflow cannot place its page, so it is no offset of any listing.
func TestReadPage_RefusesAnOffsetWithoutANextPage(t *testing.T) {
	for _, offset := range []int{math.MaxInt, math.MaxInt - pageTestLimit + 1} {
		_, err := readPage(pageTestLimit, grpcerrors.GeneratePaginationToken(offset))
		assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken), status.Code(err), offset)
	}

	page, err := readPage(pageTestLimit, grpcerrors.GeneratePaginationToken(math.MaxInt-pageTestLimit))
	require.NoError(t, err)
	placed, err := page.respond(pageTestTotal)
	require.NoError(t, err)
	assert.Empty(t, placed.nextToken, "the last addressable offset has no next page")
}

func TestRespond_NextTokenWhileRowsRemain(t *testing.T) {
	placed, err := pageRequest{limit: pageTestLimit, offset: 20}.respond(pageTestTotal)
	require.NoError(t, err)
	assert.Equal(t, pageResponse{nextToken: "40", totalCount: int32(pageTestTotal)}, placed)

	placed, err = pageRequest{limit: pageTestLimit, offset: 40}.respond(pageTestTotal)
	require.NoError(t, err)
	assert.Empty(t, placed.nextToken, "the last page has no next one")
}

func TestRespond_RefusesATotalOutsideInt32(t *testing.T) {
	for _, total := range []int64{math.MaxInt32 + 1, -1} {
		_, err := pageRequest{limit: pageTestLimit}.respond(total)
		assert.Equal(t, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenResultCountOverflow), status.Code(err))
	}
}

func TestClampDownlinkResultsPageSize(t *testing.T) {
	assert.Equal(t, DefaultDownlinkResultsPageSize, clampDownlinkResultsPageSize(0))
	assert.Equal(t, DefaultDownlinkResultsPageSize, clampDownlinkResultsPageSize(-5))
	assert.Equal(t, MaxDownlinkResultsPageSize, clampDownlinkResultsPageSize(MaxDownlinkResultsPageSize+1))
	assert.Equal(t, pageTestLimit, clampDownlinkResultsPageSize(pageTestLimit))
}
