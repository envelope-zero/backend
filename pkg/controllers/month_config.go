package controllers

import (
	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/google/uuid"
)

type MonthConfigFilter struct {
	EnvelopeID uuid.UUID
	Month      types.Month
}
