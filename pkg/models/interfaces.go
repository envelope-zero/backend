package models

import "encoding/json"

// Model is an interface that
type Model interface {
	Export() (json.RawMessage, error) // All instances of this model for export.
}

// The "Registry" is a slice of all models available
//
// It is maintained so that operations that affect all models do not need to explicitly iterate over every single model,
// increasing the risk of forgetting something when adding a new model
var Registry = []Model{
	Account{},
	Budget{},
	Category{},
	Envelope{},
	Goal{},
	MatchRule{},
	MonthConfig{},
	Transaction{},
}
