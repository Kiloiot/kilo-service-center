package postgres

import (
	"fmt"
	"strings"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// orderDirectionDesc is the default event ordering (most recent first).
const orderDirectionDesc = "DESC"

// eventOrderColumns are the columns GetEvents sorts by; no column sorts by
// occurrence. Only these names ever reach the ORDER BY clause.
var eventOrderColumns = map[string]string{
	"":                        "occurred_at",
	"occurred_at":             "occurred_at",
	"recorded_at":             "recorded_at",
	"severity":                "severity",
	"event_type":              "event_type",
	"event_category":          "event_category",
	models.EventOrderByStored: "stored_at",
}

// eventOrderDirections are the sort directions GetEvents accepts; no
// direction sorts newest first.
var eventOrderDirections = map[string]string{
	"":     orderDirectionDesc,
	"asc":  "ASC",
	"desc": orderDirectionDesc,
}

// eventTieBreakColumns settle events equal in the sorted column, so a page
// lists them the same way on every read: recording order, then id.
var eventTieBreakColumns = []string{"recorded_at", "id"}

// eventOrdering resolves the filter's ordering through the allow-lists.
func eventOrdering(filter models.SystemEventFilter) (string, string, error) {
	column, ok := eventOrderColumns[strings.ToLower(filter.OrderBy)]
	if !ok {
		return "", "", fmt.Errorf("%w: %s", errTextUnsortableEvents, filter.OrderBy)
	}
	direction, ok := eventOrderDirections[strings.ToLower(filter.OrderDirection)]
	if !ok {
		return "", "", fmt.Errorf("%w: %s", errTextUnsortableEvents, filter.OrderDirection)
	}
	return column, direction, nil
}

// eventOrderClause sorts by column, then by the tie-break columns, all in
// direction.
func eventOrderClause(column, direction string) string {
	terms := []string{column + " " + direction}
	for _, tieBreak := range eventTieBreakColumns {
		if tieBreak != column {
			terms = append(terms, tieBreak+" "+direction)
		}
	}
	return strings.Join(terms, ", ")
}
