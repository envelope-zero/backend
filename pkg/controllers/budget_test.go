package controllers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/envelope-zero/backend/pkg/test"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func createTestBudget(t *testing.T, c models.BudgetCreate) controllers.BudgetResponse {
	r := test.Request(t, http.MethodPost, "http://example.com/v1/budgets", c)
	test.AssertHTTPStatus(t, http.StatusCreated, &r)

	var a controllers.BudgetResponse
	test.DecodeResponse(t, &r, &a)

	return a
}

func (suite *TestSuiteEnv) TestGetBudgets() {
	_ = createTestBudget(suite.T(), models.BudgetCreate{})

	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/budgets", "")

	var response controllers.BudgetListResponse
	test.DecodeResponse(suite.T(), &recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, test.TOLERANCE)
}

func (suite *TestSuiteEnv) TestGetBudgetsFilter() {
	_ = createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "Exact String Match",
		Note:     "This is a specific note",
		Currency: "€",
	})

	_ = createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "Another String",
		Note:     "This is a specific note",
		Currency: "$",
	})

	_ = createTestBudget(suite.T(), models.BudgetCreate{
		Name:     "Another String",
		Note:     "A different note",
		Currency: "€",
	})

	var re controllers.BudgetListResponse

	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets?currency=€", "")
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
	test.DecodeResponse(suite.T(), &r, &re)
	assert.Equal(suite.T(), 2, len(re.Data))

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets?currency=$", "")
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
	test.DecodeResponse(suite.T(), &r, &re)
	assert.Equal(suite.T(), 1, len(re.Data))

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets?currency=€&name=Another String", "")
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
	test.DecodeResponse(suite.T(), &r, &re)
	assert.Equal(suite.T(), 1, len(re.Data))

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets?note=This is a specific note", "")
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
	test.DecodeResponse(suite.T(), &r, &re)
	assert.Equal(suite.T(), 2, len(re.Data))

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets?name=Exact String Match", "")
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
	test.DecodeResponse(suite.T(), &r, &re)
	assert.Equal(suite.T(), 1, len(re.Data))
}

func (suite *TestSuiteEnv) TestNoBudgetNotFound() {
	recorder := test.Request(suite.T(), "GET", "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd", "")

	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestBudgetInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/-56", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/notANumber", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/23", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodGet, "http://example.com/v1/budgets/d19a622f-broken-uuid/2022-01", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * PATCH
	 */
	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/budgets/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodPatch, "http://example.com/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	/*
	 * DELETE
	 */
	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/budgets/-274", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)

	r = test.Request(suite.T(), http.MethodDelete, "http://example.com/v1/budgets/stringRandom", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

func (suite *TestSuiteEnv) TestCreateBudget() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budgetObject, savedObject controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budgetObject)

	recorder = test.Request(suite.T(), "GET", budgetObject.Data.Links.Self, "")
	test.DecodeResponse(suite.T(), &recorder, &savedObject)

	assert.Equal(suite.T(), savedObject, budgetObject)
}

func (suite *TestSuiteEnv) TestCreateBrokenBudget() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/budgets", `{ "createdAt": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestCreateBudgetNoBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/budgets", "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

// TestBudgetMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteEnv) TestBudgetMonth() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})
	category := createTestCategory(suite.T(), models.CategoryCreate{BudgetID: budget.Data.ID})
	envelope := createTestEnvelope(suite.T(), models.EnvelopeCreate{CategoryID: category.Data.ID, Name: "Utilities"})
	account := createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID})
	externalAccount := createTestAccount(suite.T(), models.AccountCreate{BudgetID: budget.Data.ID, External: true})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      1,
		Year:       2022,
		Amount:     decimal.NewFromFloat(20.99),
	})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      2,
		Year:       2022,
		Amount:     decimal.NewFromFloat(47.12),
	})

	_ = createTestAllocation(suite.T(), models.AllocationCreate{
		EnvelopeID: envelope.Data.ID,
		Month:      3,
		Year:       2022,
		Amount:     decimal.NewFromFloat(31.17),
	})

	_ = createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(10.0),
		Note:                 "Water bill for January",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           envelope.Data.ID,
		Reconciled:           true,
	})

	_ = createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 2, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(5.0),
		Note:                 "Water bill for February",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           envelope.Data.ID,
		Reconciled:           true,
	})

	_ = createTestTransaction(suite.T(), models.TransactionCreate{
		Date:                 time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Amount:               decimal.NewFromFloat(15.0),
		Note:                 "Water bill for March",
		BudgetID:             budget.Data.ID,
		SourceAccountID:      account.Data.ID,
		DestinationAccountID: externalAccount.Data.ID,
		EnvelopeID:           envelope.Data.ID,
		Reconciled:           true,
	})

	tests := []struct {
		path     string
		response controllers.BudgetMonthResponse
	}{
		{
			fmt.Sprintf("%s/2022-01", budget.Data.Links.Self),
			controllers.BudgetMonthResponse{
				Data: models.BudgetMonth{
					Month: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
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
					Month: time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
							Spent:      decimal.NewFromFloat(-5),
							Balance:    decimal.NewFromFloat(42.12),
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
					Month: time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
					Envelopes: []models.EnvelopeMonth{
						{
							Name:       "Utilities",
							Month:      time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
							Spent:      decimal.NewFromFloat(-15),
							Balance:    decimal.NewFromFloat(16.17),
							Allocation: decimal.NewFromFloat(31.17),
						},
					},
				},
			},
		},
	}

	var budgetMonth controllers.BudgetMonthResponse
	for _, tt := range tests {
		r := test.Request(suite.T(), "GET", tt.path, "")
		test.AssertHTTPStatus(suite.T(), http.StatusOK, &r)
		test.DecodeResponse(suite.T(), &r, &budgetMonth)

		// assert.FailNow(suite.T(), "BudgetMonth", budgetMonth)

		if !assert.Len(suite.T(), budgetMonth.Data.Envelopes, 1) {
			assert.FailNow(suite.T(), "Response length does not match!", "Response does not have exactly 1 item")
		}

		envelope := budgetMonth.Data.Envelopes[0]
		assert.True(suite.T(), envelope.Spent.Equal(tt.response.Data.Envelopes[0].Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[0].Spent, envelope.Spent, budgetMonth.Data)
		assert.True(suite.T(), envelope.Balance.Equal(tt.response.Data.Envelopes[0].Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[0].Balance, envelope.Balance, budgetMonth.Data)
		assert.True(suite.T(), envelope.Allocation.Equal(tt.response.Data.Envelopes[0].Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", budgetMonth.Data.Month, tt.response.Data.Envelopes[0].Allocation, envelope.Allocation, budgetMonth.Data)
	}
}

// TestBudgetMonthNonExistent verifies that month requests for non-existing budgets return a HTTP 404 Not Found.
func (suite *TestSuiteEnv) TestBudgetMonthNonExistent() {
	r := test.Request(suite.T(), "GET", "http://example.com/v1/budgets/65064e6f-04b4-46e0-8bbc-88c96c6b21bd/2022-01", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &r)
}

// TestBudgetMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteEnv) TestBudgetMonthZero() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})

	r := test.Request(suite.T(), "GET", fmt.Sprintf("%s/0001-01", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &r)
}

// TestBudgetMonthInvalid tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteEnv) TestBudgetMonthInvalid() {
	budget := createTestBudget(suite.T(), models.BudgetCreate{})

	recorder := test.Request(suite.T(), "GET", fmt.Sprintf("%s/December-2020", budget.Data.Links.Self), "")
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateBudget() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), "PATCH", budget.Data.Links.Self, `{ "name": "Updated new budget" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusOK, &recorder)

	var updatedBudget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &updatedBudget)

	assert.Equal(suite.T(), budget.Data.Note, updatedBudget.Data.Note)
	assert.Equal(suite.T(), "Updated new budget", updatedBudget.Data.Name)
}

func (suite *TestSuiteEnv) TestUpdateBudgetBroken() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/budgets", `{ "name": "New Budget", "note": "More tests something something" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), "PATCH", budget.Data.Links.Self, `{ "name": 2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusBadRequest, &recorder)
}

func (suite *TestSuiteEnv) TestUpdateNonExistingBudget() {
	recorder := test.Request(suite.T(), "PATCH", "http://example.com/v1/budgets/a29bd123-beec-47de-a9cd-b6f7483fe00f", `{ "name": "2" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteBudget() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), "DELETE", budget.Data.Links.Self, "")
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteNonExistingBudget() {
	recorder := test.Request(suite.T(), "DELETE", "http://example.com/v1/budgets/c3d34346-609a-4734-9364-98f5b0100150", "")
	test.AssertHTTPStatus(suite.T(), http.StatusNotFound, &recorder)
}

func (suite *TestSuiteEnv) TestDeleteBudgetWithBody() {
	recorder := test.Request(suite.T(), "POST", "http://example.com/v1/budgets", `{ "name": "Delete me now!" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusCreated, &recorder)

	var budget controllers.BudgetResponse
	test.DecodeResponse(suite.T(), &recorder, &budget)

	recorder = test.Request(suite.T(), "DELETE", budget.Data.Links.Self, `{ "name": "test name 23" }`)
	test.AssertHTTPStatus(suite.T(), http.StatusNoContent, &recorder)
}
