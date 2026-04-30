package models

import (
	"time"

	"github.com/google/uuid"
)

// OrganizationMembershipWithOrg represents a joined view of membership with organization details.
// Used for listing user memberships with organization context.
type OrganizationMembershipWithOrg struct {
	OrgID              uuid.UUID `db:"org_id"`
	OrgName            string    `db:"org_name"`
	Role               string    `db:"role"`
	Status             string    `db:"status"`
	IsOrgAdmin         bool      `db:"is_org_admin"`
	IsBaseStationAdmin bool      `db:"is_base_station_admin"`
	IsEndpointAdmin    bool      `db:"is_endpoint_admin"`
	CreatedAt          time.Time `db:"created_at"`
}
