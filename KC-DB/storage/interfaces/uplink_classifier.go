package interfaces

import "context"

// UplinkClassifierReset forgets the duplicate-classifier state of an endpoint
// whose packet counter an over-the-air attach restarted.
type UplinkClassifierReset interface {
	ForgetEndpoint(ctx context.Context, tenantID int64, endpointID int64) error
}
