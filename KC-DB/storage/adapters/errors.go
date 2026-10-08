// Package adapters bridges the KC-DB repositories to the consumer-side ports of
// KC-Core and KC-Identity.
package adapters

import "errors"

// Error-wrap contexts and error format strings, grouped by the file that
// uses them; the shared group serves multiple files.
const (
	// shared
	errWrapRollback = "rollback"

	// downlink_reservation_adapter.go
	errWrapDownlinkReservationReserve = "downlink reservation: reserve"

	// refresh_token_adapter.go
	errWrapRefreshTokenAdapterCreate         = "refresh token adapter: create"
	errWrapRefreshTokenAdapterGetByHash      = "refresh token adapter: get_by_hash"
	errWrapRefreshTokenAdapterMarkReplaced   = "refresh token adapter: mark_replaced"
	errWrapRefreshTokenAdapterRevokeByHash   = "refresh token adapter: revoke_by_hash"
	errWrapRefreshTokenAdapterRevokeByUserID = "refresh token adapter: revoke_by_user_id"

	// tenant_adapter.go
	errWrapTenantAdapterCreate    = "tenant adapter: create"
	errWrapTenantAdapterDelete    = "tenant adapter: delete"
	errWrapTenantAdapterGet       = "tenant adapter: get"
	errWrapTenantAdapterList      = "tenant adapter: list"
	errWrapTenantAdapterSetStatus = "tenant adapter: set_status"
	errWrapTenantAdapterUpdate    = "tenant adapter: update"

	// user_adapter.go
	errWrapUserAdapterCount           = "user adapter: count"
	errWrapUserAdapterCreate          = "user adapter: create"
	errWrapUserAdapterDelete          = "user adapter: delete"
	errWrapUserAdapterGetByEmail      = "user adapter: get_by_email"
	errWrapUserAdapterGetByExternalID = "user adapter: get_by_external_id"
	errWrapUserAdapterGetByID         = "user adapter: get_by_id"
	errWrapUserAdapterList            = "user adapter: list"
	errWrapUserAdapterSetPasswordHash = "user adapter: set_password_hash" // #nosec G101 -- error context string, not a credential
	errWrapUserAdapterUpdate          = "user adapter: update"
)

// Sentinel errors migrated from in-function errors.New literals.
var (
	// tenant_adapter.go
	errTextTenantAdapterUpdateNoFieldsUpdate = errors.New("tenant adapter: update: no fields to update")
)
