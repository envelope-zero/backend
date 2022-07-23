package controllers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func createTestCategory(t *testing.T, c models.CategoryCreate) controllers.CategoryResponse {
	if c.BudgetID == uuid.Nil {
		c.BudgetID = createTestBudget(t, models.BudgetCreate{Name: "Testing budget"}).Data.ID
	}

	r := test.Request(t, http.MethodPost, "http://example.com/v1/categories", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var category controllers.CategoryResponse
	test.DecodeResponse(t, &r, &category)

	return category
}

func (suite *TestSuiteEnv) TestGetCategories() {
	_ = createTestCategory(suite.T(), models.CategoryCreate{})

	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
}

func (suite *TestSuiteEnv) TestGetCategoriesEnvelopes() {
	category := createTestCategory(suite.T(), models.CategoryCreate{})
	_ = createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	_ = createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})

	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)
	assert.Len(suite.T(), response.Data[0].Envelopes, 2)
}

func (suite *TestSuiteEnv) TestGetCategoriesNoEnvelopesEmptyArray() {
	_ = createTestCategory(suite.T(), models.CategoryCreate{})

	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)
	assert.NotNil(suite.T(), response.Data[0].Envelopes, "Envelopes must be an empty array when no envelopes are present, not nil")
	assert.Len(suite.T(), response.Data[0].Envelopes, 0)
}

func (suite *TestSuiteEnv) TestGetCategory() {
	category := createTestCategory(suite.T(), models.CategoryCreate{Name: "Catch me if you can!"})
	recorder := test.Request(suite.T(), http.MethodGet, category.Data.Links.Self, "")

	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)
}

func (suite *TestSuiteEnv) TestNoCategoryNotFound() {
	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/categories/4e743e94-6a4b-44d6-aba5-d77c87103ff7", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestCategoryInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/categories/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/categories/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/categories/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/categories/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/categories/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/categories/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/categories/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateCategory() {
	_ = createTestCategory(suite.T(), models.CategoryCreate{Name: "New Category", Note: "Something to test creation"})
}

func (suite *TestSuiteEnv) TestCreateBrokenCategory() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "createdAt": "New Category", "note": "More tests for categories to ensure less brokenness something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateBudgetDoesNotExist() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "budgetId": "f8c74664-9b79-4e15-8d3d-4618f3e3c230" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestCreateCategoryNoBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateCategory() {
	category := createTestCategory(suite.T(), models.CategoryCreate{Name: "New category", Note: "Mor(r)e tests"})

	recorder := test.Request(suite.T(), "PATCH", category.Data.Links.Self, `{ "name": "Updated new category for testing" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedCategory controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedCategory)

	assert.Equal(suite.T(), category.Data.Note, updatedCategory.Data.Note)
	assert.Equal(suite.T(), "Updated new category for testing", updatedCategory.Data.Name)
}

func (suite *TestSuiteEnv) TestUpdateCategoryBroken() {
	category := createTestCategory(suite.T(), models.CategoryCreate{Name: "New category", Note: "Mor(r)e tests"})

	recorder := test.Request(suite.T(), "PATCH", category.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingCategory() {
	recorder := test.Request(suite.T(), "PATCH", "http://example.com/v1/categories/f9288848-517e-4b8c-9f14-b3d849ca275b", `{ "name": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteCategory() {
	category := createTestCategory(suite.T(), models.CategoryCreate{Name: "Delete me now!"})

	recorder := test.Request(suite.T(), "DELETE", category.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingCategory() {
	recorder := test.Request(suite.T(), "DELETE", "http://example.com/v1/categories/a2aa0569-5ac5-42e1-8563-7c61194cc7d9", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteCategoryWithBody() {
	category := createTestCategory(suite.T(), models.CategoryCreate{Name: "Delete me now!"})

	recorder := test.Request(suite.T(), "DELETE", category.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}
