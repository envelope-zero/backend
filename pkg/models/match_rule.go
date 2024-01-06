package models

import (
	"github.com/google/uuid"
)

type MatchRule struct {
	DefaultModel
	AccountID uuid.UUID
	Priority  uint
	Match     string
}

func (r MatchRule) Self() string {
	return "Match Rule"
}
