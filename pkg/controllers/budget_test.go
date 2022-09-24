package controllers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) createTestBudget(t *testing.T, c models.BudgetCreate, expectedStatus ...int) controllers.BudgetResponse {
	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v1/budgets", c)
	test.AssertHTTPStatus(t, &r, expectedStatus...)

	var a controllers.BudgetResponse
	test.DecodeResponse(t, &r, &a)

	return a
}

func (suite *TestSuiteStandard) TestCreateBudgetNoDB() {
	suite.CloseDB()
	suite.createTestBudget(suite.T(), models.BudgetCreate{}, http.StatusInternalServerError)
}

func (suite *TestSuiteStandard) TestBudgets() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusInternalServerError)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteStandard) TestOptionsBudget() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/budgets", uuid.New())
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.Self
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestOptionsBudgetMonth() {
	budgetLink := suite.createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.Month

	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, strings.Replace(budgetLink, "YYYY-MM", "1970-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
	assert.Equal(suite.T(), recorder.Header().Get("allow"), "OPTIONS, GET")

	// Bad Request for invalid UUID
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/nouuid/2022-01", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid month
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, budgetLink, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/5b95e1a9-522d-4a36-9074-32f7c2ff0513/1980-06", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestOptionsBudgetMonthAllocations() {
	budgetAllocationsLink := suite.createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.MonthAllocations

	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, strings.Replace(budgetAllocationsLink, "YYYY-MM", "1970-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
	assert.Equal(suite.T(), recorder.Header().Get("allow"), "OPTIONS, DELETE")

	// Bad Request for invalid UUID
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/nouuid/2022-01/allocations", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, budgetAllocationsLink, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/budgets/059cdead-249f-4f94-8d29-16a80c6b4a09/2032-03/allocations", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestGetBudgets() {
	_ = suite.createTestBudget(suite.T(), models.BudgetCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets", "")

	var response controllers.BudgetListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
}

func (suite *TestSuiteStandard) TestGetBudgetsFilter() {
	_ = suite.createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "Exact String Match",
		Note:     "This is a specific note",
		Currency: "",
	})

	_ = suite.createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "",
		Note:     "This is a specific note",
		Currency: "$",
	})

	_ = suite.createTestBudget(suite.T(), models.BudgetCreate{
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
	}

	var re controllers.BudgetListResponse
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v1/budgets?%s", tt.query), "")
			test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
			test.DecodeResponse(suite.T(), &r, &re)
			assert.Equal(suite.T(), tt.len, len(re.Data))
		})
	}
}

func (suite *TestSuiteStandard) TestNoBudgetNotFound() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd", "")

	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestBudgetInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/-56", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/notANumber", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/23", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/d19a622f-broken-uuid/2022-01", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	/*
	 * PATCH
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/budgets/-274", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	/*
	 * DELETE
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/-274", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budgetObject, savedObject controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budgetObject)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, budgetObject.Data.Links.Self, "")
	test.DecodeResponse(suite.T(), &recorder, &savedObject)

	assert.Equal(suite.T(), savedObject, budgetObject)
}

func (suite *TestSuiteStandard) TestCreateBrokenBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "createdAt": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateBudgetNoBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestBudgetMonth() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	account := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})
	externalAccount := suite.createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID, External: true})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(20.99),
	})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(47.12),
	})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(31.17),
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
		Reconciled:           true,
	})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
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
					Month:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
					Income: decimal.NewFromFloat(0),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
							Spent:      decimal.NewFromFloat(10),
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
					Month:  time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
					Income: decimal.NewFromFloat(0),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
							Balance:    decimal.NewFromFloat(53.11),
							Spent:      decimal.NewFromFloat(5),
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
					Month:  time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
					Income: decimal.NewFromFloat(1500),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
							Balance:    decimal.NewFromFloat(69.28),
							Spent:      decimal.NewFromFloat(15),
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
		test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
		test.DecodeResponse(suite.T(), &r, &budgetMonth)

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
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	envelopeZero := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Zero"})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelopeZero.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(19.01),
	})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(20.99),
	})

	var budgetMonth controllers.BudgetMonthResponse

	r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/2022-01", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusOK)
	test.DecodeResponse(suite.T(), &r, &budgetMonth)

	assert.True(suite.T(), budgetMonth.Data.Budgeted.Equal(decimal.NewFromFloat(40)), "Calculation of budgeted sum for a month is off. Should be 40, is %s", budgetMonth.Data.Budgeted)
}

// TestBudgetMonthNonExistent verifies that month requests for non-existing budgets return a HTTP 404 Not Found.
func (suite *TestSuiteStandard) TestBudgetMonthNonExistent() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd/2022-01", "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusNotFound)
}

// TestBudgetMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteStandard) TestBudgetMonthZero() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})

	r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/0001-01", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

// TestBudgetMonthInvalid tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteStandard) TestBudgetMonthInvalid() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/December-2020", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": "Updated new budget",
		"note": "",
	})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedBudget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedBudget)

	assert.Equal(suite.T(), "", updatedBudget.Data.Note)
	assert.Equal(suite.T(), "Updated new budget", updatedBudget.Data.Name)
}

func (suite *TestSuiteStandard) TestUpdateBudgetBrokenJSON() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodPatch, budget.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateBudgetInvalidType() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodPatch, budget.Data.Links.Self, map[string]any{
		"name": 2,
	})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/budgets/a29bd123-beec-47de-a9cd-b6f7483fe00f", `{ "name": "2" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, budget.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestDeleteNonExistingBudget() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/c3d34346-609a-4734-9364-98f5b0100150", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteBudgetWithBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusCreated)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, budget.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestDeleteAllocationsMonth() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})

	allocation1 := suite.createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(15.42),
		EnvelopeID: envelope2.Data.ID,
	})

	// Clear allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-01", 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify that allocations are deleted
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation1.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)

	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, allocation2.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteAllocationsMonthFailures() {
	budgetAllocationsLink := suite.createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.MonthAllocations

	// Bad Request for invalid UUID
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/nouuid/2022-01/allocations", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, budgetAllocationsLink, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/budgets/059cdead-249f-4f94-8d29-16a80c6b4a09/2032-03/allocations", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestSetAllocationsMonthBudgeted() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})

	allocation1 := suite.createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	allocation2 := suite.createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(40),
		EnvelopeID: envelope2.Data.ID,
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-02", 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthBudget})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, requestString, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month controllers.EnvelopeMonthResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope1Month)
	suite.Assert().True(allocation1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", allocation1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month controllers.EnvelopeMonthResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope2Month)
	suite.Assert().True(allocation2.Data.Amount.Equal(envelope2Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", allocation2.Data.Amount, envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestSetAllocationsMonthSpend() {
	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{})
	cashAccount := suite.createTestAccount(suite.T(), models.AccountCreate{External: false})
	externalAccount := suite.createTestAccount(suite.T(), models.AccountCreate{External: true})
	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope1 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})
	envelope2 := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(30),
		EnvelopeID: envelope1.Data.ID,
	})

	_ = suite.createTestAllocation(suite.T(), models.AllocationCreate{
		Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		Amount:     decimal.NewFromFloat(40),
		EnvelopeID: envelope2.Data.ID,
	})

	eID := &envelope1.Data.ID
	transaction1 := suite.createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 14, 43, 27, 0, time.UTC),
		EnvelopeID:           eID,
		BudgetID:             budget.Data.ID,
		SourceAccountID:      cashAccount.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		Amount:               decimal.NewFromFloat(15),
	})

	// Update in budgeted mode allocations
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budget.Data.Links.MonthAllocations, "YYYY-MM", "2022-02", 1), controllers.BudgetAllocationMode{Mode: controllers.AllocateLastMonthSpend})
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)

	// Verify the allocation for the first envelope
	requestString := strings.Replace(envelope1.Data.Links.Month, "YYYY-MM", "2022-02", 1)
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, requestString, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope1Month controllers.EnvelopeMonthResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope1Month)
	suite.Assert().True(transaction1.Data.Amount.Equal(envelope1Month.Data.Allocation), "Expected: %s, got %s, Request ID: %s", transaction1.Data.Amount, envelope1Month.Data.Allocation, recorder.Header().Get("x-request-id"))

	// Verify the allocation for the second envelope
	recorder = test.Request(suite.controller, suite.T(), http.MethodGet, strings.Replace(envelope2.Data.Links.Month, "YYYY-MM", "2022-02", 1), "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusOK)
	var envelope2Month controllers.EnvelopeMonthResponse
	test.DecodeResponse(suite.T(), &recorder, &envelope2Month)
	suite.Assert().True(envelope2Month.Data.Allocation.Equal(decimal.NewFromFloat(0)), "Expected: 0, got %s, Request ID: %s", envelope2Month.Data.Allocation, recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestSetAllocationsMonthFailures() {
	budgetAllocationsLink := suite.createTestBudget(suite.T(), models.BudgetCreate{}).Data.Links.MonthAllocations

	// Bad Request for invalid UUID
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets/nouuid/2022-01/allocations", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid months
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, budgetAllocationsLink, "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Not found for non-existing budget
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/budgets/059cdead-249f-4f94-8d29-16a80c6b4a09/2032-03/allocations", "")
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)

	// Bad Request for invalid json in body
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": INVALID_JSON" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)

	// Bad Request for invalid mode
	recorder = test.Request(suite.controller, suite.T(), http.MethodPost, strings.Replace(budgetAllocationsLink, "YYYY-MM", "2022-01", 1), `{ "mode": "UNKNOWN_MODE" }`)
	test.AssertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

// TestBudgetBalanceDoubleRegression verifies that the Budget balance is only added once.
func (suite *TestSuiteStandard) TestBudgetBalanceDoubleRegression() {
	shouldBalance := decimal.NewFromFloat(1000)

	budget := suite.createTestBudget(suite.T(), models.BudgetCreate{Name: "TestBudgetBalanceDoubleRegression"})

	internalAccount := suite.createTestAccount(suite.T(), models.AccountCreate{
		BudgetID: budget.Data.ID,
		OnBudget: true,
		External: false,
	})

	externalAccount := suite.createTestAccount(suite.T(), models.AccountCreate{
		BudgetID: budget.Data.ID,
		OnBudget: true,
		External: true,
	})

	category := suite.createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := suite.createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID})

	_ = suite.createTestTransaction(suite.T(), models.TransactionCreate{
		BudgetID:             budget.Data.ID,
		Amount:               shouldBalance,
		SourceAccountID:      externalAccount.Data.ID,
		DestinationAccountID: internalAccount.Data.ID,
		EnvelopeID:           &envelope.Data.ID,
	})

	var budgetResponse controllers.BudgetResponse
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, budget.Data.Links.Self, "")
	test.DecodeResponse(suite.T(), &recorder, &budgetResponse)

	assert.True(suite.T(), budgetResponse.Data.Balance.Equal(shouldBalance), "Balance is %s, should be %s", budgetResponse.Data.Balance, shouldBalance)
}
