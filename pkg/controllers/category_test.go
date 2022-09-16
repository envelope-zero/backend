package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestCategory(t *testing.T, c models.CategoryCreate) controllers.CategoryResponse {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = suite.createTestBudget(t, models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	if c.Name == "" {
		c.Name = uuid.NewString()
	}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v1/categories", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var category controllers.CategoryResponse
	test.DecodeResponse(t, &r, &category)

	return category
}

func (suite *TestSuiteStandard) TestOptionsCategory() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/categories", uuid.New())
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/categories/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestCategory(suite.T(), models.CategoryCreate{}).Data.Links.Self
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestGetCategories() {
	_ = suite.createTestCategory(suite.T(), models.CategoryCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
}

func (suite *TestSuiteStandard) TestGetCategoriesEnvelopes() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{})
	_ = suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	_ = suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)
	assert.Len(suite.T(), response.Data[0].Envelopes, 2)
}

func (suite *TestSuiteStandard) TestGetCategoriesNoEnvelopesEmptyArray() {
	_ = suite.createTestCategory(suite.T(), models.CategoryCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)
	assert.NotNil(suite.T(), response.Data[0].Envelopes, "Envelopes must be an empty array when no envelopes are present, not nil")
	assert.Len(suite.T(), response.Data[0].Envelopes, 0)
}

func (suite *TestSuiteStandard) TestGetCategoriesInvalidQuery() {
	tests := []string{
		"budget=NotAUUID",
	}

	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v1/categories?%s", tt), "")
			test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
		})
	}
}

func (suite *TestSuiteStandard) TestGetCategoriesFilter() {
	b1 := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	b2 := suite.createTestBudget(suite.T(), models.BudgetCreate{})

	_ = suite.createTestCategory(suite.T(), models.CategoryCreate{
		Name:     "Category Name",
		Note:     "A note for this category",
		BudgetID: b1.Data.ID,
	})

	_ = suite.createTestCategory(suite.T(), models.CategoryCreate{
		Name:     "Saving",
		Note:     "For later",
		BudgetID: b2.Data.ID,
	})

	_ = suite.createTestCategory(suite.T(), models.CategoryCreate{
		Name:     "Daily stuff",
		Note:     "Groceries, Drug Store, …",
		BudgetID: b2.Data.ID,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Budget 1", fmt.Sprintf("budget=%s", b1.Data.ID), 1},
		{"Budget Not Existing", "budget=c9e4ee7a-e702-4f92-b168-11a95b22c7aa", 0},
		{"Empty Note", "note=", 0},
		{"Empty Name", "name=", 0},
		{"Name & Note", "name=Category Name&note=A note for this category", 1},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.CategoryListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v1/categories?%s", tt.query), "")
			test.AssertHTTPStatus(t, http.StatusOK, &r)
			test.DecodeResponse(t, &r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestGetCategory() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "Catch me if you can!"})
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, category.Data.Links.Self, "")

	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)
}

func (suite *TestSuiteStandard) TestNoCategoryNotFound() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/categories/4e743e94-6a4b-44d6-aba5-d77c87103ff7", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestCategoryInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/categories/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/categories/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/categories/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/categories/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/categories/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/categories/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/categories/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteStandard) TestCreateCategory() {
	_ = suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "New Category", Note: "Something to test creation"})
}

func (suite *TestSuiteStandard) TestCreateBrokenCategory() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/categories", `{ "createdAt": "New Category", "note": "More tests for categories to ensure less brokenness something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestCreateBudgetDoesNotExist() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/categories", `{ "budgetId": "f8c74664-9b79-4e15-8d3d-4618f3e3c230" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestCreateCategoryNoBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/categories", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestCreateCategoryDuplicateName() {
	c := suite.createTestCategory(suite.T(), models.CategoryCreate{
		Name: "Unique Category Name",
	})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/categories", models.CategoryCreate{
		BudgetID: c.Data.BudgetID,
		Name:     c.Data.Name,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateCategory() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "New category", Note: "Mor(r)e tests"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, category.Data.Links.Self, map[string]any{
		"name": "Updated new category for testing",
		"note": "",
	})
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedCategory controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedCategory)

	assert.Equal(suite.T(), "", updatedCategory.Data.Note)
	assert.Equal(suite.T(), "Updated new category for testing", updatedCategory.Data.Name)
}

func (suite *TestSuiteStandard) TestUpdateCategoryBrokenJSON() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "New category", Note: "Mor(r)e tests"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, category.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateCategoryInvalidType() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "New category", Note: "Mor(r)e tests"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, category.Data.Links.Self, map[string]any{
		"name": 2,
	})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateCategoryInvalidBudgetID() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "New category", Note: "Mor(r)e tests"})

	// Sets the BudgetID to uuid.Nil
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, category.Data.Links.Self, models.CategoryCreate{})
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingCategory() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/categories/f9288848-517e-4b8c-9f14-b3d849ca275b", `{ "name": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestDeleteCategory() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "Delete me now!"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, category.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteStandard) TestDeleteNonExistingCategory() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/categories/a2aa0569-5ac5-42e1-8563-7c61194cc7d9", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteStandard) TestDeleteCategoryWithBody() {
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{Name: "Delete me now!"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, category.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}
