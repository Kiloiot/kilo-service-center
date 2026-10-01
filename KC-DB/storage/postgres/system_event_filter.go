package postgres

import (
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// sqlCountEvents is the event count; filters append to its WHERE.
const sqlCountEvents = `SELECT COUNT(*) FROM system_events WHERE 1=1`

// Predicates of the event listing and its count; both bounds of the
// occurrence window are inclusive.
const (
	sqlEventTenantFmt     = " AND tenant_id = $%d"
	sqlEventSinceFmt      = " AND occurred_at >= $%d"
	sqlEventUntilFmt      = " AND occurred_at <= $%d"
	sqlEventCategoryFmt   = " AND event_category = ANY($%d)"
	sqlEventSeverityFmt   = " AND severity = ANY($%d)"
	sqlEventTypeFmt       = " AND event_type = ANY($%d)"
	sqlEventSourceTypeFmt = " AND source_type = ANY($%d)"
	sqlEventStatusFmt     = " AND status = ANY($%d)"
	sqlEventSourceIDFmt   = " AND source_id = $%d"
	sqlEventStoredFmt     = " AND stored_at >= $%d"
)

// eventPredicates accumulates AND-ed predicates and the values they bind.
type eventPredicates struct {
	sql  string
	args []interface{}
}

// bind appends a predicate whose format takes the placeholder of value.
func (p *eventPredicates) bind(predicateFmt string, value interface{}) {
	p.args = append(p.args, value)
	p.sql += fmt.Sprintf(predicateFmt, len(p.args))
}

// bindAny appends a membership predicate unless values is empty.
func (p *eventPredicates) bindAny(predicateFmt string, values []string) {
	if len(values) > 0 {
		p.bind(predicateFmt, pq.Array(values))
	}
}

// eventFilterWhere renders the filter the event listing and its count share,
// as predicates to append to a WHERE clause, and the values they bind.
func eventFilterWhere(filter models.SystemEventFilter) (string, []interface{}, error) {
	if filter.TenantID == "" {
		return "", nil, errTextTenantIDRequired
	}
	tenantID, err := strconv.ParseInt(filter.TenantID, 10, 64)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", errWrapInvalidTenantIDFormat, err)
	}
	p := &eventPredicates{}
	p.bind(sqlEventTenantFmt, tenantID)
	if filter.Since != nil {
		p.bind(sqlEventSinceFmt, *filter.Since)
	}
	if filter.Until != nil {
		p.bind(sqlEventUntilFmt, *filter.Until)
	}
	p.bindAny(sqlEventCategoryFmt, filter.Categories)
	p.bindAny(sqlEventSeverityFmt, filter.Severities)
	p.bindAny(sqlEventTypeFmt, filter.EventTypes)
	p.bindAny(sqlEventSourceTypeFmt, filter.SourceTypes)
	p.bindAny(sqlEventStatusFmt, filter.Status)
	if filter.SourceID != nil {
		p.bind(sqlEventSourceIDFmt, *filter.SourceID)
	}
	if filter.StoredSince != nil {
		p.bind(sqlEventStoredFmt, *filter.StoredSince)
	}
	query, args, argNum := appendDeviceScope(p.sql, p.args, len(p.args)+1, filter)
	query, args, _ = appendEventSearch(query, args, argNum, filter)
	return query, args, nil
}
