package models

// Budget represents a budget
//
// A budget is the highest level of organization in Envelope Zero, all other
// resources reference it directly or transitively.
type Budget struct {
	Model
	Name string `json:"name,omitempty"`
	Note string `json:"note,omitempty"`
}
