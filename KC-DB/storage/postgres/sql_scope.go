package postgres

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Columns the scoped downlink and uplink predicates narrow on.
const (
	colID             = "id"
	colTenantID       = "tenant_id"
	colOrganizationID = "organization_id"
	colEpEUI          = "ep_eui"
	colBsEUI          = "bs_eui"
	colQueID          = "que_id"
	colPacketCnt      = "packet_cnt"
	colStatus         = "status"
	colPriority       = "priority"
	colCreatedAt      = "created_at"
	colTransmittedAt  = "transmitted_at"
	colCommandType    = "command_type"
	colRxTime         = "rx_time"
	colDuplicate      = "duplicate"
	colDlOpen         = "dl_open"
	colProfile        = "profile"
	colMode           = "mode"
)

// Condition formats the scoped predicates render.
const (
	// sqlAnyOfFmt matches a column against a bound array of values.
	sqlAnyOfFmt = "%s = ANY(%s)"
	// sqlAtLeastFmt and sqlAtMostFmt bound a column inclusively.
	sqlAtLeastFmt = "%s >= %s"
	sqlAtMostFmt  = "%s <= %s"
)

// sqlScope builds a WHERE predicate of AND-ed conditions whose positional
// parameters continue after the arguments already bound, so a listing and
// its count, or a statement and its follow-up lookup, share one predicate.
type sqlScope struct {
	conditions []string
	args       []interface{}
}

// newSQLScope starts a predicate after the statement's leading arguments.
func newSQLScope(args ...interface{}) *sqlScope {
	return &sqlScope{args: args}
}

// bind adds a parameter and returns its placeholder.
func (s *sqlScope) bind(value interface{}) string {
	s.args = append(s.args, value)
	return fmt.Sprintf("$%d", len(s.args))
}

// and adds a condition that binds no parameter of its own.
func (s *sqlScope) and(condition string) {
	s.conditions = append(s.conditions, condition)
}

// equals adds column = value.
func (s *sqlScope) equals(column string, value interface{}) {
	s.and(column + " = " + s.bind(value))
}

// anyOf adds column = ANY(values), values bound as one array.
func (s *sqlScope) anyOf(column string, values interface{}) {
	s.and(fmt.Sprintf(sqlAnyOfFmt, column, s.bind(values)))
}

// atLeast adds column >= value.
func (s *sqlScope) atLeast(column string, value interface{}) {
	s.and(fmt.Sprintf(sqlAtLeastFmt, column, s.bind(value)))
}

// atMost adds column <= value.
func (s *sqlScope) atMost(column string, value interface{}) {
	s.and(fmt.Sprintf(sqlAtMostFmt, column, s.bind(value)))
}

// organization narrows the predicate to one organization's rows; nil
// narrows nothing. Every organization-scoped downlink statement goes through
// it, so a row without an organization never satisfies a scoped one.
func (s *sqlScope) organization(orgID *uuid.UUID) {
	if orgID != nil {
		s.equals(colOrganizationID, *orgID)
	}
}

// where renders the AND-ed conditions.
func (s *sqlScope) where() string {
	return strings.Join(s.conditions, " AND ")
}

// page appends LIMIT and OFFSET after the predicate's parameters.
func (s *sqlScope) page(limit, offset int) string {
	return fmt.Sprintf(" LIMIT %s OFFSET %s", s.bind(limit), s.bind(offset))
}
