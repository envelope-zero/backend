package models

import (
	"github.com/google/uuid"
)

type MatchRule struct {
	DefaultModel
	MatchRuleCreate
}

type MatchRuleCreate struct {
	Priority  uint      `json:"priority" example:"3"`                                     // The priority of the match rule
	Match     string    `json:"match" example:"Bank*"`                                    // The matching applied to the opposite account. This is a glob pattern. Multiple globs are allowed. Globbing is case sensitive.
	AccountID uuid.UUID `json:"accountId" example:"f9e873c2-fb96-4367-bfb6-7ecd9bf4a6b5"` // The account to map matching transactions to
}

func (r MatchRule) Self() string {
	return "Match Rule"
}
