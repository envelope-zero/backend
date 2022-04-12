package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/stretchr/testify/assert"
)

type CategoryListResponse struct {
	test.APIResponse
	Data []models.Category
}

type CategoryDetailResponse struct {
	test.APIResponse
	Data models.Category
}

func TestGetCategories(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories", "")

	var response CategoryListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 1) {
		assert.FailNow(t, "Response does not have exactly 1 item")
	}

	assert.Equal(t, uint64(1), response.Data[0].BudgetID)
	assert.Equal(t, "Running costs", response.Data[0].Name)
	assert.Equal(t, "For e.g. groceries and energy bills", response.Data[0].Note)

	diff := time.Now().Sub(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Now().Sub(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoCategoryNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/2", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

// TestNonexistingBudgetCategories404 is a regression test for https://github.com/envelope-zero/backend/issues/89.
//
// It verifies that for a non-existing budget, the accounts endpoint raises a 404
// instead of returning an empty list.
func TestNonexistingBudgetCategories404(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/999/categories", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var apiCategory CategoryDetailResponse
	test.DecodeResponse(t, &recorder, &apiCategory)

	var dbCategory models.Category
	models.DB.First(&dbCategory, apiCategory.Data.ID)

	assert.Equal(t, dbCategory, apiCategory.Data)
}

func TestCreateBrokenCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "createdAt": "New Category", "note": "More tests for categories to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateCategoryNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestGetCategory(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1", "")
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var category CategoryDetailResponse
	test.DecodeResponse(t, &recorder, &category)

	var dbCategory models.Category
	models.DB.First(&dbCategory, category.Data.ID)

	assert.Equal(t, dbCategory, category.Data)
}

func TestUpdateCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category CategoryDetailResponse
	test.DecodeResponse(t, &recorder, &category)

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new category for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedCategory CategoryDetailResponse
	test.DecodeResponse(t, &recorder, &updatedCategory)

	assert.Equal(t, category.Data.Note, updatedCategory.Data.Note)
	assert.Equal(t, "Updated new category for testing", updatedCategory.Data.Name)
}

func TestUpdateCategoryBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category CategoryDetailResponse
	test.DecodeResponse(t, &recorder, &category)

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingCategory(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/1/categories/48902805", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category CategoryDetailResponse
	test.DecodeResponse(t, &recorder, &category)

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	recorder = test.Request(t, "DELETE", path, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingCategory(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/categories/48902805", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteCategoryWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category CategoryDetailResponse
	test.DecodeResponse(t, &recorder, &category)

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
