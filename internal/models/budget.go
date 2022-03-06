package models

import (
	"time"
)

// Budget represents a budget
//
// A budget is the highest level of organization in Envelope Zero, all other
// resources reference it directly or transitively.
type Budget struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Name      string    `json:"name"`
}

// CreateBudget defines all values required to create a new budget
type CreateBudget struct {
	Name string `json:"name" binding:"required"`
}
