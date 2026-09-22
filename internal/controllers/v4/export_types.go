package v4

import (
	"encoding/json"
	"time"
)

type ExportResponse struct {
	Version      string                     `json:"version"`      // The version of the backend the export was made with
	Data         map[string]json.RawMessage `json:"data"`         // The exported data
	CreationTime time.Time                  `json:"creationTime"` // Time the export was created
	Clacks       string                     `json:"clacks"`       // This will always have the value "GNU Terry Pratchett"
}
