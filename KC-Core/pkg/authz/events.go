package authz

import (
	"sort"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// categoryRequirements names who may read each event category. Security,
// system, audit and error events describe the whole installation, so only
// administrators read them.
var categoryRequirements = map[string]Requirement{
	models.EventCategoryBaseStation: BaseStationManager,
	models.EventCategoryBSSCI:       BaseStationManager,
	models.EventCategoryEndpoint:    EndpointManager,
	models.EventCategoryMessage:     EndpointManager,
	models.EventCategorySCACI:       EndpointManager,
	models.EventCategoryRoaming:     EndpointManager,
	models.EventCategoryProtocol:    AnyManager,
	models.EventCategorySession:     AnyManager,
	models.EventCategorySecurity:    AdminOnly,
	models.EventCategorySystem:      AdminOnly,
	models.EventCategoryAudit:       AdminOnly,
	models.EventCategoryError:       AdminOnly,
}

var knownCategories = func() []string {
	categories := make([]string, 0, len(categoryRequirements))
	for category := range categoryRequirements {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}()

// VisibleEventCategories narrows the requested categories, or every category
// when none is requested, to those the roles may read. unrestricted reports
// that the caller may read everything it asked for without a category filter.
func VisibleEventCategories(r Roles, requested []string) (categories []string, unrestricted bool) {
	if r.Admin {
		return requested, len(requested) == 0
	}
	if len(requested) == 0 {
		requested = knownCategories
	}
	visible := make([]string, 0, len(requested))
	for _, category := range requested {
		if CanReadCategory(r, category) {
			visible = append(visible, category)
		}
	}
	return visible, false
}

// CanReadCategory reports whether the roles may read events of the category;
// an unknown category is readable by administrators only.
func CanReadCategory(r Roles, category string) bool {
	requirement, known := categoryRequirements[category]
	if !known {
		return r.Admin
	}
	return requirement.GrantedTo(r)
}

// CanReadCategories reports whether the roles may read every listed category.
func CanReadCategories(r Roles, categories []string) bool {
	for _, category := range categories {
		if !CanReadCategory(r, category) {
			return false
		}
	}
	return true
}
