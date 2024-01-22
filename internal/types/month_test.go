package types_test

import (
	"encoding/json"
	"testing"

	"github.com/envelope-zero/backend/v5/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestMonthUnmarshalJSON(t *testing.T) {
	var target struct {
		Month types.Month
	}
	jsonString := []byte(`{ "month": "2024-05-12T17:59:23+02:00" }`)

	err := json.Unmarshal(jsonString, &target)

	assert.Nil(t, err)
	assert.Equal(t, types.NewMonth(2024, 5), target.Month)
}
