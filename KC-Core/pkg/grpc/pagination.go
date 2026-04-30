package grpc

import "strconv"

// Pagination defaults for standard List RPC handlers.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// High-volume pagination for endpoint, base station, and downlink lists.
const (
	DefaultHighVolumePageSize = 100
	MaxHighVolumePageSize     = 1000
)

// ClampPageSize normalizes a requested page size to the allowed range.
func ClampPageSize(requested int32) int {
	n := int(requested)
	if n <= 0 || n > MaxPageSize {
		return DefaultPageSize
	}
	return n
}

// ClampHighVolumePageSize normalizes page size for high-volume list endpoints.
func ClampHighVolumePageSize(requested int32, defaultSize int32) int32 {
	if requested <= 0 {
		return defaultSize
	}
	if requested > MaxHighVolumePageSize {
		return MaxHighVolumePageSize
	}
	return requested
}

// ParsePaginationToken parses an offset-based pagination token.
// Returns (true, nil) if token was parsed, (false, nil) if token is empty.
func ParsePaginationToken(token string, offset *int) (bool, error) {
	if token == "" {
		return false, nil
	}
	val, err := strconv.Atoi(token)
	if err != nil {
		return false, err
	}
	*offset = val
	return true, nil
}

// GeneratePaginationToken creates an offset-based pagination token.
func GeneratePaginationToken(offset int) string {
	return strconv.Itoa(offset)
}
