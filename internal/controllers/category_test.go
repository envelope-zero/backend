package controllers_test

import (
	"encoding/json"
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
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, 1, len(response.Data))
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

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestCreateCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	assert.Equal(t, http.StatusCreated, recorder.Code)

	var apiCategory CategoryDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiCategory)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbCategory models.Category
	models.DB.First(&dbCategory, apiCategory.Data.ID)

	assert.Equal(t, dbCategory, apiCategory.Data)
}

func TestCreateBrokenCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "createdAt": "New Category", "note": "More tests for categories to ensure less brokenness something" }`)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCreateCategoryNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", "")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestGetCategory(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/budgets/1/categories/1", "")
	assert.Equal(t, http.StatusOK, recorder.Code)

	var category CategoryDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&category)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	var dbCategory models.Category
	models.DB.First(&dbCategory, category.Data.ID)

	assert.Equal(t, dbCategory, category.Data)
}

func TestUpdateCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	assert.Equal(t, http.StatusCreated, recorder.Code)

	var category CategoryDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&category)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": "Updated new category for testing" }`)
	assert.Equal(t, http.StatusOK, recorder.Code)

	var updatedCategory CategoryDetailResponse
	err = json.NewDecoder(recorder.Body).Decode(&updatedCategory)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, category.Data.Note, updatedCategory.Data.Note)
	assert.Equal(t, "Updated new category for testing", updatedCategory.Data.Name)
}

func TestUpdateCategoryBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	assert.Equal(t, http.StatusCreated, recorder.Code)

	var category CategoryDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&category)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	recorder = test.Request(t, "PATCH", path, `{ "name": 2" }`)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestUpdateNonExistingCategory(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/budgets/1/categories/48902805", `{ "name": 2" }`)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestDeleteCategory(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/categories/1", "")
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestDeleteNonExistingCategory(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/budgets/1/categories/48902805", "")
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestDeleteCategoryWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/budgets/1/categories", `{ "name": "Delete me now!" }`)

	var category CategoryDetailResponse
	err := json.NewDecoder(recorder.Body).Decode(&category)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	path := fmt.Sprintf("/v1/budgets/1/categories/%v", category.Data.ID)
	recorder = test.Request(t, "DELETE", path, `{ "name": "test name 23" }`)
	assert.Equal(t, http.StatusNoContent, recorder.Code)
}
