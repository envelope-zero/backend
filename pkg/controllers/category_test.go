package controllers_test

import (
	"net/http"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteEnv) TestGetCategories() {
	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/categories", "")

	var response controllers.CategoryListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	if !assert.Len(suite.T(), response.Data, 1) {
		assert.FailNow(suite.T(), "Response does not have exactly 1 item")
	}

	assert.Equal(suite.T(), "Running costs", response.Data[0].Name)
	assert.Equal(suite.T(), "For e.g. groceries and energy bills", response.Data[0].Note)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
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
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var categoryObject, savedCategory controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &categoryObject)

	recorder = test.Request(suite.T(), "GET", categoryObject.Data.Links.Self, "")
	test.DecodeResponse(suite.T(), &recorder, &savedCategory)

	assert.Equal(suite.T(), savedCategory, categoryObject)
}

func (suite *TestSuiteEnv) TestCreateBrokenCategory() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "createdAt": "New Category", "note": "More tests for categories to ensure less brokenness something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateBrokenNoBudget() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "budgetId": "f8c74664-9b79-4e15-8d3d-4618f3e3c230" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestCreateCategoryNoBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateCategory() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &category)

	recorder = test.Request(suite.T(), "PATCH", category.Data.Links.Self, `{ "name": "Updated new category for testing" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedCategory controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedCategory)

	assert.Equal(suite.T(), category.Data.Note, updatedCategory.Data.Note)
	assert.Equal(suite.T(), "Updated new category for testing", updatedCategory.Data.Name)
}

func (suite *TestSuiteEnv) TestUpdateCategoryBroken() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "name": "New Category", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &category)

	recorder = test.Request(suite.T(), "PATCH", category.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingCategory() {
	recorder := test.Request(suite.T(), "PATCH", "http://example.com/v1/categories/f9288848-517e-4b8c-9f14-b3d849ca275b", `{ "name": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteCategory() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &category)

	recorder = test.Request(suite.T(), "DELETE", category.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingCategory() {
	recorder := test.Request(suite.T(), "DELETE", "http://example.com/v1/categories/a2aa0569-5ac5-42e1-8563-7c61194cc7d9", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteCategoryWithBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/categories", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var category controllers.CategoryResponse
	test.DecodeResponse(suite.T(), &recorder, &category)

	recorder = test.Request(suite.T(), "DELETE", category.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}
