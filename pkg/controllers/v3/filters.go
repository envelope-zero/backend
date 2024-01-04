package v3

import (
	"fmt"

	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

func stringFilters(db, query *gorm.DB, setFields []string, name, note, search string) *gorm.DB {
	if name != "" {
		query = query.Where("name LIKE ?", fmt.Sprintf("%%%s%%", name))
	} else if slices.Contains(setFields, "Name") {
		query = query.Where("name = ''")
	}

	if note != "" {
		query = query.Where("note LIKE ?", fmt.Sprintf("%%%s%%", note))
	} else if slices.Contains(setFields, "Note") {
		query = query.Where("note = ''")
	}

	if search != "" {
		query = query.Where(
			db.Where("note LIKE ?", fmt.Sprintf("%%%s%%", search)).Or(
				db.Where("name LIKE ?", fmt.Sprintf("%%%s%%", search)),
			),
		)
	}

	return query
}
