package scaciservices

import "errors"

// baseStationListAll disables pagination when fetching base stations for
// SCACI status reporting; a zero limit makes the repository return every row.
const baseStationListAll = 0

// errResumeRequiresPersistedSession refuses a resume write for a session that
// was not resumed from a persisted row.
var errResumeRequiresPersistedSession = errors.New("resume persistence requires a resumed session with a persisted ID")
