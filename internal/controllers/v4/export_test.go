package v4_test

import (
	"encoding/json"
	"net/http"
	"time"

	v4 "github.com/envelope-zero/backend/v5/internal/controllers/v4"
	"github.com/envelope-zero/backend/v5/internal/models"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExport verifies that the export works correctly
//
// Thorough checks are only executed for the non-data fields since
// the data fields are populated by the Export() methods of the models
func (suite *TestSuiteStandard) TestExport() {
	t := suite.T()

	b := createTestBudget(t, v4.BudgetEditable{})
	c := createTestCategory(t, v4.CategoryEditable{BudgetID: b.Data.ID})

	recorder := test.Request(t, http.MethodGet, "http://example.com/v4/export", "")
	test.AssertHTTPStatus(t, &recorder, http.StatusOK)

	var response v4.ExportResponse
	test.DecodeResponse(t, &recorder, &response)

	// Verify the version and clacks fields
	assert.Equal(t, "GNU Terry Pratchett", response.Clacks)
	assert.Equal(t, "0.0.0", response.Version)

	// Not sure if this is a good test, if it ever fails we'll re-evaluate
	now := time.Now()
	difference := response.CreationTime.Sub(now).Seconds()
	assert.Less(t, difference, float64(1))

	// Basic tests for the data fields. Full testing is done in the respective Export() methods
	// of the models
	assert.Len(t, response.Data, len(models.Registry), "Number of models in export does not match registry")

	// CreatedAt check for budget
	var budgets []models.Budget
	require.Nil(t, json.Unmarshal(response.Data["Budget"], &budgets))
	require.Len(t, budgets, 1, "Number of budgets in export must be 1")
	assert.Equal(t, b.Data.CreatedAt, budgets[0].CreatedAt)

	// CreatedAt check for category
	var categories []models.Category
	require.Nil(t, json.Unmarshal(response.Data["Category"], &categories))
	require.Len(t, categories, 1, "Number of categories in export must be 1")
	assert.Equal(t, c.Data.CreatedAt, categories[0].CreatedAt)
}
