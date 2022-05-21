package controllers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestGetCategories(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(t, &recorder, &response)

	assert.Equal(t, 200, recorder.Code)
	if !assert.Len(t, response.Data, 1) {
		assert.FailNow(t, "Response does not have exactly 1 item")
	}

	assert.Equal(t, "Running costs", response.Data[0].Name)
	assert.Equal(t, "For e.g. groceries and energy bills", response.Data[0].Note)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(t, diff, test.TOLERANCE)
}

func TestNoCategoryNotFound(t *testing.T) {
	recorder := test.Request(t, "GET", "/v1/categories/4e743e94-6a4b-44d6-aba5-d77c87103ff7", "")

	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCategoryInvalidIDs(t *testing.T) {
	/*
	 *  GET
	 */
	r := test.Request(t, http.MethodGet, "/v1/categories/-56", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/categories/notANumber", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodGet, "/v1/categories/23", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(t, http.MethodPatch, "/v1/categories/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodPatch, "/v1/categories/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(t, http.MethodDelete, "/v1/categories/-274", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)

	r = test.Request(t, http.MethodDelete, "/v1/categories/stringRandom", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &r)
}

func TestCreateCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var categoryObject, savedCategory controllers.CategoryResponse
	test.DecodeResponse(t, &recorder, &categoryObject)

	recorder = test.Request(t, "GET", categoryObject.Data.Links.Self, "")
	test.DecodeResponse(t, &recorder, &savedCategory)

	assert.Equal(t, savedCategory, categoryObject)
}

func TestCreateBrokenCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", `{ "createdAt": "New Category", "note": "More tests for categories to ensure less brokenness something" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestCreateBrokenNoBudget(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", `{ "budgetId": "f8c74664-9b79-4e15-8d3d-4618f3e3c230" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestCreateCategoryNoBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", "")
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(t, &recorder, &category)

	recorder = test.Request(t, "PATCH", category.Data.Links.Self, `{ "name": "Updated new category for testing" }`)
	test.AssertHTTPStatus(t, http.StatusOK, &recorder)

	var updatedCategory controllers.CategoryResponse
	test.DecodeResponse(t, &recorder, &updatedCategory)

	assert.Equal(t, category.Data.Note, updatedCategory.Data.Note)
	assert.Equal(t, "Updated new category for testing", updatedCategory.Data.Name)
}

func TestUpdateCategoryBroken(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(t, &recorder, &category)

	recorder = test.Request(t, "PATCH", category.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(t, http.StatusBadRequest, &recorder)
}

func TestUpdateNonExistingCategory(t *testing.T) {
	recorder := test.Request(t, "PATCH", "/v1/categories/f9288848-517e-4b8c-9f14-b3d849ca275b", `{ "name": "2" }`)
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteCategory(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(t, &recorder, &category)

	recorder = test.Request(t, "DELETE", category.Data.Links.Self, "")
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}

func TestDeleteNonExistingCategory(t *testing.T) {
	recorder := test.Request(t, "DELETE", "/v1/categories/a2aa0569-5ac5-42e1-8563-7c61194cc7d9", "")
	test.AssertHTTPStatus(t, http.StatusNotFound, &recorder)
}

func TestDeleteCategoryWithBody(t *testing.T) {
	recorder := test.Request(t, "POST", "/v1/categories", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(t, http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(t, &recorder, &category)

	recorder = test.Request(t, "DELETE", category.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(t, http.StatusNoContent, &recorder)
}
