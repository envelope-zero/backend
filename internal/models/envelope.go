package models

// Envelope represents an envelope in your budget.
type Envelope struct {
	Model
	Name       string   `json:"name,omitempty"`
	CategoryID uint64   `json:"categoryId"`
	Category   Category `json:"-"`
	Note       string   `json:"note,omitempty"`
}
