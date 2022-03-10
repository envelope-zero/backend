package models

// Envelope represents an envelope in your budget
type Envelope struct {
	Model
	Name       string   `json:"name"`
	CategoryID int      `json:"categoryId"`
	Category   Category `json:"-"`
}

// CreateEnvelope defines all values required to create a new envelope
type CreateEnvelope struct {
	Name string `json:"name" binding:"required"`
}
