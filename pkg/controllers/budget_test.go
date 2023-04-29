package controllers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/v2/internal/types"
	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/envelope-zero/backend/v2/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestBudget(c models.BudgetCreate, expectedStatus ...int) controllers.BudgetResponse {
	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", c)
	assertHTTPStatus(suite.T(), &r, expectedStatus...)

	var a controllers.BudgetResponse
	suite.decodeResponse(&r, &a)

	return a
}

func (suite *TestSuiteStandard) TestCreateBudgetNoDB() {
	suite.CloseDB()
	suite.createTestBudget(models.BudgetCreate{}, http.StatusInternalServerError)
}

func (suite *TestSuiteStandard) TestBudgets() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusInternalServerError)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteStandard) TestOptionsBudget() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/budgets", uuid.New())
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestBudget(models.BudgetCreate{}).Data.Links.Self
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestBudget(models.BudgetCreate{}).Data.Links.MonthAllocations
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestOptionsBudgetMonth() {
	budgetLink := suite.createTestBudget(models.BudgetCreate{}).Data.Links.Month

	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, strings.Replace(budgetLink, "YYYY-MM", "1970-01", 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
	assert.Equal(suite.T(), recorder.Header().Get("allow"), "OPTIONS, GET")

	// Bad Request for invalid UUID
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/nouuid/2022-01", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid month
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, budgetLink, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/5b95e1a9-522d-4a36-9074-32f7c2ff0513/1980-06", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestGetBudget() {
	budget := suite.createTestBudget(models.BudgetCreate{})

	tests := []struct {
		name     string
		id       uuid.UUID
		response int
	}{
		{"Existing budget", budget.Data.ID, http.StatusOK},
		{"ID nil", uuid.Nil, http.StatusBadRequest},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			recorder := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("http://example.com/v1/budgets/%s", tt.id), "")

			var response controllers.BudgetResponse
			suite.decodeResponse(&recorder, &response)

			assert.Equal(t, tt.response, recorder.Code, "Wrong response code, Request ID: %s, Object: %v", recorder.Result().Header.Get("x-request-id"), response)
		})
	}
}

func (suite *TestSuiteStandard) TestGetBudgets() {
	_ = suite.createTestBudget(models.BudgetCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets", "")

	var response controllers.BudgetListResponse
	suite.decodeResponse(&recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, tolerance)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, tolerance)
}

func (suite *TestSuiteStandard) TestGetBudgetsFilter() {
	_ = suite.createTestBudget(models.BudgetCreate{
		Name:     "Exact String Match",
		Note:     "This is a specific note",
		Currency: "",
	})

	_ = suite.createTestBudget(models.BudgetCreate{
		Name:     "",
		Note:     "This is a specific note",
		Currency: "$",
	})

	_ = suite.createTestBudget(models.BudgetCreate{
		Name:     "Another String",
		Note:     "A different note",
		Currency: "€",
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Currency: €", "currency=€", 1},
		{"Currency: $", "currency=$", 1},
		{"Currency & Name", "currency=€&name=Another String", 1},
		{"Note", "note=This is a specific note", 2},
		{"Name", "name=Exact String Match", 1},
		{"Empty Name with Note", "name=&note=This is a specific note", 1},
		{"No currency", "currency=", 1},
		{"No name", "name=", 1},
		{"Search for 'stRing'", "search=stRing", 2},
		{"Search for 'Note'", "search=Note", 3},
	}

	var re controllers.BudgetListResponse
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v1/budgets?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)
			assert.Equal(t, tt.len, len(re.Data))
		})
	}
}

func (suite *TestSuiteStandard) TestNoBudgetNotFound() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd", "")

	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestBudgetInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/-56", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/notANumber", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/23", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/d19a622f-broken-uuid/2022-01", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	/*
	 * PATCH
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/budgets/-274", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/budgets/stringRandom", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	/*
	 * DELETE
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/-274", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/stringRandom", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budgetObject, savedObject controllers.BudgetResponse
	suite.decodeResponse(&recorder, &budgetObject)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, budgetObject.Data.Links.Self, "")
	suite.decodeResponse(&recorder, &savedObject)

	assert.Equal(suite.T(), savedObject, budgetObject)
}

func (suite *TestSuiteStandard) TestCreateBrokenBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "createdAt": "New Budget", "note": "More tests something something" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateBudgetNoBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestBudgetMonth() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	account := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, OnBudget: true})
	externalAccount := suite.createTestAccount(models.AccountCreate{BudgetID: budget.Data.ID, External: true})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(20.99),
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      types.NewMonth(2022, 2),
		Amount:     decimal.NewFromFloat(47.12),
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      types.NewMonth(2022, 3),
		Amount:     decimal.NewFromFloat(31.17),
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 3, 1, 7, 38, 17, 0, time.UTC),
		Amount:               decimal.NewFromFloat(1500),
		Note:                 "Income for march",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      externalAccount.Data.ID,
		DestinationAccountID: account.Data.ID,
		EnvelopeID:           nil,
	})

	tests := []struct {
		path     string
		response controllers.BudgetMonthResponse
	}{
		{
			fmt.Sprintf("%s/2022-01", budget.Data.Links.Self),
			controllers.BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month:  types.NewMonth(2022, 1),
					Income: decimal.NewFromFloat(0),
					Envelopes: []models.EnvelopeMonth{
						{
							Envelope: models.Envelope{
								EnvelopeCreate: models.EnvelopeCreate{
									Name: "Utilities",
								},
							},
							Month:      types.NewMonth(2022, 1),
							Spent:      decimal.NewFromFloat(-10),
							Balance:    decimal.NewFromFloat(10.99),
							Allocation: decimal.NewFromFloat(20.99),
						},
					},
				},
			},
		},
		{
			fmt.Sprintf("%s/2022-02", budget.Data.Links.Self),
			controllers.BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month:  types.NewMonth(2022, 2),
					Income: decimal.NewFromFloat(0),
					Envelopes: []models.EnvelopeMonth{
						{
							Envelope: models.Envelope{
								EnvelopeCreate: models.EnvelopeCreate{
									Name: "Utilities",
								},
							},
							Month:      types.NewMonth(2022, 2),
							Balance:    decimal.NewFromFloat(53.11),
							Spent:      decimal.NewFromFloat(-5),
							Allocation: decimal.NewFromFloat(47.12),
						},
					},
				},
			},
		},
		{
			fmt.Sprintf("%s/2022-03", budget.Data.Links.Self),
			controllers.BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month:  types.NewMonth(2022, 3),
					Income: decimal.NewFromFloat(1500),
					Envelopes: []models.EnvelopeMonth{
						{
							Envelope: models.Envelope{
								EnvelopeCreate: models.EnvelopeCreate{
									Name: "Utilities",
								},
							},
							Month:      types.NewMonth(2022, 3),
							Balance:    decimal.NewFromFloat(69.28),
							Spent:      decimal.NewFromFloat(-15),
							Allocation: decimal.NewFromFloat(31.17),
						},
					},
				},
			},
		},
	}

	var budgetMonth controllers.BudgetMonthResponse
	for _, tt := range tests {
		r := test.Request(suite.controller, suite.T(), http.MethodGet, tt.path, "")
		assertHTTPStatus(suite.T(), &r, http.StatusOK)
		suite.decodeResponse(&r, &budgetMonth)

		// Verify income calculation
		assert.True(suite.T(), budgetMonth.Data.Income.Equal(tt.response.Data.Income))

		if !assert.Len(suite.T(), budgetMonth.Data.Envelopes, 1) {
			assert.FailNow(suite.T(), "Response length does not match!", "Response does not have exactly 1 item")
		}

		expected := tt.response.Data.Envelopes[0]
		envelope := budgetMonth.Data.Envelopes[0]
		assert.True(suite.T(), envelope.Spent.Equal(expected.Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, expected.Spent, envelope.Spent, budgetMonth.Data)
		assert.True(suite.T(), envelope.Balance.Equal(expected.Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, expected.Balance, envelope.Balance, budgetMonth.Data)
		assert.True(suite.T(), envelope.Allocation.Equal(expected.Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, expected.Allocation, envelope.Allocation, budgetMonth.Data)
	}
}

func (suite *TestSuiteStandard) TestBudgetMonthBudgeted() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	envelopeZero := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Zero"})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelopeZero.Data.ID,
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(19.01),
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(20.99),
	})

	var budgetMonth controllers.BudgetMonthResponse

	r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/2022-01", budget.Data.Links.Self), "")
	assertHTTPStatus(suite.T(), &r, http.StatusOK)
	suite.decodeResponse(&r, &budgetMonth)

	assert.True(suite.T(), budgetMonth.Data.Budgeted.Equal(decimal.NewFromFloat(40)), "Calculation of budgeted sum for a month is off. Should be 40, is %s", budgetMonth.Data.Budgeted)
}

// TestBudgetMonthNonExistent verifies that month requests for non-existing budgets return a HTTP 404 Not Found.
func (suite *TestSuiteStandard) TestBudgetMonthNonExistent() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd/2022-01", "")
	assertHTTPStatus(suite.T(), &r, http.StatusNotFound)
}

// TestBudgetMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteStandard) TestBudgetMonthZero() {
	budget := suite.createTestBudget(models.BudgetCreate{})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/0001-01", budget.Data.Links.Self), "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

// TestBudgetMonthInvalid tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteStandard) TestBudgetMonthInvalid() {
	budget := suite.createTestBudget(models.BudgetCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/December-2020", budget.Data.Links.Self), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	suite.decodeResponse(&recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": "Updated new budget",
		"note": "",
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedBudget controllers.BudgetResponse
	suite.decodeResponse(&recorder, &updatedBudget)

	assert.Equal(suite.T(), "", updatedBudget.Data.Note)
	assert.Equal(suite.T(), "Updated new budget", updatedBudget.Data.Name)
}

func (suite *TestSuiteStandard) TestUpdateBudgetBrokenJSON() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	suite.decodeResponse(&recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodPatch, budget.Data.Links.Self, `{ "name": 2" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateBudgetInvalidType() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	suite.decodeResponse(&recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": 2,
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/budgets/a29bd123-beec-47de-a9cd-b6f7483fe00f", `{ "name": "2" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	suite.decodeResponse(&recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, budget.Data.Links.Self, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestDeleteNonExistingBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/c3d34346-609a-4734-9364-98f5b0100150", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteBudgetWithBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	suite.decodeResponse(&recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, budget.Data.Links.Self, `{ "name": "test name 23" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestDeleteAllocationsMonth() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	allocation1 := suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope2.Data.ID,
	})

	// Clear allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-01", 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify that allocations are deleted
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation1.Data.Links.Self, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation2.Data.Links.Self, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteAllocationsMonthFailures() {
	budgetAllocationsLink := suite.createTestBudget(models.BudgetCreate{}).Data.Links.MonthAllocations

	// Bad Request for invalid UUID
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/nouuid/2022-01/allocations", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, budgetAllocationsLink, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/059cdead-249f-4f94-8d29-16a80c6b4a09/2032-03/allocations", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestSetAllocationsMonthBudgeted() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	allocation1 := suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(40),
		EnvelopeID: envelope2.Data.ID,
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-02", 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthBudget})
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, requestString, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope1Month)
	suite.Assert().True(allocation1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", allocation1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope2Month)
	suite.Assert().True(allocation2.Data.Amount.Equal(envelope2Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", allocation2.Data.Amount, envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestSetAllocationsMonthSpend() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	cashAccount := suite.createTestAccount(models.AccountCreate{External: false, OnBudget: true})
	externalAccount := suite.createTestAccount(models.AccountCreate{External: true})
	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	_ = suite.createTestAllocation(models.AllocationCreate{
		Month:      types.NewMonth(2022, 1),
		Amount:     decimal.NewFromFloat(40),
		EnvelopeID: envelope2.Data.ID,
	})

	eID := &envelope1.Data.ID
	transaction1 := suite.createTestTransaction(models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 14, 43, 27, 0, time.UTC),
		EnvelopeID:           eID,
		BudgetID:             budget.Data.ID,
		SourceAccountID:      cashAccount.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		Amount:               decimal.NewFromFloat(15),
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-02", 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthSpend})
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, requestString, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope1Month)
	suite.Assert().True(transaction1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", transaction1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month controllers.EnvelopeMonthResponse
	suite.decodeResponse(&recorder, &envelope2Month)
	suite.Assert().True(envelope2Month.Data.Allocation.Equal(decimal.NewFromFloat(0)), "Expected: 0, got %s, Request ID: %s", envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestSetAllocationsMonthFailures() {
	budgetAllocationsLink := suite.createTestBudget(models.BudgetCreate{}).Data.Links.MonthAllocations

	// Bad Request for invalid UUID
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets/nouuid/2022-01/allocations", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, budgetAllocationsLink, "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets/059cdead-249f-4f94-8d29-16a80c6b4a09/2032-03/allocations", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)

	// Bad Request for invalid json in body
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": INVALID_JSON" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid mode
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": "UNKNOWN_MODE" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

// TestBudgetBalanceDoubleRegression verifies that the Budget balance is only added once.
func (suite *TestSuiteStandard) TestBudgetBalanceDoubleRegression() {
	shouldBalance := decimal.NewFromFloat(1000)

	budget := suite.createTestBudget(models.BudgetCreate{Name: "TestBudgetBalanceDoubleRegression"})

	internalAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.Data.ID,
		OnBudget: true,
		External: false,
	})

	externalAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.Data.ID,
		OnBudget: true,
		External: true,
	})

	category := suite.createTestCategory(models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{CategoryID: category.Data.ID})

	_ = suite.createTestTransaction(models.TransactionCreate{
		BudgetID:             budget.Data.ID,
		Amount:               shouldBalance,
		SourceAccountID:      externalAccount.Data.ID,
		DestinationAccountID: internalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
	})

	var budgetResponse controllers.BudgetResponse
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, budget.Data.Links.Self, "")
	suite.decodeResponse(&recorder, &budgetResponse)

	assert.True(suite.T(), budgetResponse.Data.Balance.Equal(shouldBalance), "Balance is %s, should be %s", budgetResponse.Data.Balance, shouldBalance)
}
