package models

import (
	"fmt"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RenameRule struct {
	DefaultModel
	RenameRuleCreate
	Links struct {
		Self string `json:"self" example:"https://example.com/api/v2/rename-rules/95685c82-53c6-455d-b235-f49960b73b21"`
	} `json:"links" gorm:"-"`
}

type RenameRuleCreate struct {
	Priority  uint      `json:"priority" example:"3"`                                     // The priority of the rename rule
	Match     string    `json:"match" example:"Bank*"`                                    // The matching applied to the opposite account. This is a glob pattern. Multiple globs are allowed. Globbing is case sensitive.
	AccountID uuid.UUID `json:"accountId" example:"f9e873c2-fb96-4367-bfb6-7ecd9bf4a6b5"` // The account to map matching transactions to
}

func (r RenameRule) Self() string {
	return "Rename Rule"
}

func (r *RenameRule) links(tx *gorm.DB) {
	r.Links.Self = fmt.Sprintf("%s/v2/rename-rules/%s", tx.Statement.Context.Value(database.ContextURL), r.ID)
}

func (r *RenameRule) AfterSave(tx *gorm.DB) (err error) {
	r.links(tx)
	return
}

func (r *RenameRule) AfterFind(tx *gorm.DB) (err error) {
	r.links(tx)
	return
}
