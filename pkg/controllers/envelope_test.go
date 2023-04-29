package controllers_test

import (
	"fmt"
	"net/http"
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

func (suite *TestSuiteStandard) createTestEnvelope(c models.EnvelopeCreate, expectedStatus ...int) controllers.EnvelopeResponse {
	if c.CategoryID == uuid.Nil {
		c.CategoryID = suite.createTestCategory(models.CategoryCreate{}).Data.ID
	}

	if c.Name == "" {
		c.Name = uuid.NewString()
	}

	// Default to 200 OK as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/envelopes", c)
	assertHTTPStatus(suite.T(), &r, expectedStatus...)

	var e controllers.EnvelopeResponse
	suite.decodeResponse(&r, &e)

	return e
}

func (suite *TestSuiteStandard) TestEnvelopes() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/envelopes", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusInternalServerError)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}

func (suite *TestSuiteStandard) TestOptionsEnvelope() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v1/envelopes", uuid.New())
	recorder := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNotFound, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v1/envelopes/NotParseableAsUUID", "")
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))

	path = suite.createTestEnvelope(models.EnvelopeCreate{}).Data.Links.Self
	recorder = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assert.Equal(suite.T(), http.StatusNoContent, recorder.Code, "Request ID %s", recorder.Header().Get("x-request-id"))
}

func (suite *TestSuiteStandard) TestGetEnvelopes() {
	_ = suite.createTestEnvelope(models.EnvelopeCreate{})

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/envelopes", "")

	var response controllers.EnvelopeListResponse
	suite.decodeResponse(&recorder, &response)

	assert.Equal(suite.T(), 200, recorder.Code)
	assert.Len(suite.T(), response.Data, 1)

	diff := time.Since(response.Data[0].CreatedAt)
	assert.LessOrEqual(suite.T(), diff, tolerance)

	diff = time.Since(response.Data[0].UpdatedAt)
	assert.LessOrEqual(suite.T(), diff, tolerance)
}

func (suite *TestSuiteStandard) TestGetEnvelopesInvalidQuery() {
	tests := []string{
		"category=DefinitelyACat",
	}

	for _, tt := range tests {
		suite.T().Run(tt, func(t *testing.T) {
			recorder := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("http://example.com/v1/envelopes?%s", tt), "")
			assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
		})
	}
}

func (suite *TestSuiteStandard) TestGetEnvelopesFilter() {
	c1 := suite.createTestCategory(models.CategoryCreate{})
	c2 := suite.createTestCategory(models.CategoryCreate{})

	_ = suite.createTestEnvelope(models.EnvelopeCreate{
		Name:       "Groceries",
		Note:       "For the stuff bought in supermarkets",
		CategoryID: c1.Data.ID,
	})

	_ = suite.createTestEnvelope(models.EnvelopeCreate{
		Name:       "Hairdresser",
		Note:       "Because… Hair!",
		CategoryID: c2.Data.ID,
		Hidden:     true,
	})

	_ = suite.createTestEnvelope(models.EnvelopeCreate{
		Name:       "Stamps",
		Note:       "Because each stamp needs to go on an envelope. Hopefully it's not hairy",
		CategoryID: c2.Data.ID,
	})

	tests := []struct {
		name  string
		query string
		len   int
	}{
		{"Category 2", fmt.Sprintf("category=%s", c2.Data.ID), 2},
		{"Category Not Existing", "category=e0f9ff7a-9f07-463c-bbd2-0d72d09d3cc6", 0},
		{"Empty Note", "note=", 0},
		{"Empty Name", "name=", 0},
		{"Name & Note", "name=Groceries&note=For the stuff bought in supermarkets", 1},
		{"Fuzzy name", "name=es", 2},
		{"Fuzzy note", "note=Because", 2},
		{"Not hidden", "hidden=false", 2},
		{"Hidden", "hidden=true", 1},
		{"Search for 'hair'", "search=hair", 2},
		{"Search for 'st'", "search=st", 2},
		{"Search for 'STUFF'", "search=STUFF", 1},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var re controllers.EnvelopeListResponse
			r := test.Request(suite.controller, t, http.MethodGet, fmt.Sprintf("/v1/envelopes?%s", tt.query), "")
			assertHTTPStatus(suite.T(), &r, http.StatusOK)
			suite.decodeResponse(&r, &re)

			assert.Equal(t, tt.len, len(re.Data), "Request ID: %s", r.Result().Header.Get("x-request-id"))
		})
	}
}

func (suite *TestSuiteStandard) TestGetEnvelope() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, envelope.Data.Links.Self, "")

	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)
}

func (suite *TestSuiteStandard) TestNoEnvelopeNotFound() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/envelopes/828f2483-dabd-4267-a223-e34b5f171978", "")

	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestEnvelopeInvalidIDs() {
	/*
	 *  GET
	 */
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/envelopes/-56", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/envelopes/notANumber", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/envelopes/23", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/v1/envelopes/d19a622f-broken-uuid/2017-09", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	/*
	 * PATCH
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/envelopes/-274", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/envelopes/stringRandom", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	/*
	 * DELETE
	 */
	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/envelopes/-274", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	r = test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/envelopes/stringRandom", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateEnvelope() {
	_ = suite.createTestEnvelope(models.EnvelopeCreate{Name: "New envelope", Note: "More tests something something"})
}

func (suite *TestSuiteStandard) TestCreateEnvelopeNoCategory() {
	r := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/envelopes", models.Envelope{})
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateBrokenEnvelope() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/envelopes", `{ "createdAt": "New Envelope", "note": "More tests for envelopes to ensure less brokenness something" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateEnvelopeNonExistingCategory() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/envelopes", `{ "categoryId": "5f0cd7b9-9788-4871-96f8-c816c9ae338a" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestCreateEnvelopeNoBody() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/envelopes", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestCreateEnvelopeDuplicateName() {
	e := suite.createTestEnvelope(models.EnvelopeCreate{
		Name: "Unique Category Name",
	})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPost, "http://example.com/v1/envelopes", models.EnvelopeCreate{
		CategoryID: e.Data.CategoryID,
		Name:       e.Data.Name,
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

// TestEnvelopeMonth verifies that the monthly calculations are correct.
func (suite *TestSuiteStandard) TestEnvelopeMonth() {
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

	tests := []struct {
		path          string
		envelopeMonth models.EnvelopeMonth
	}{
		{
			fmt.Sprintf("%s/2022-01", envelope.Data.Links.Self),
			models.EnvelopeMonth{
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
		{
			fmt.Sprintf("%s/2022-02", envelope.Data.Links.Self),
			models.EnvelopeMonth{
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
		{
			fmt.Sprintf("%s/2022-03", envelope.Data.Links.Self),
			models.EnvelopeMonth{
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
		// This month should be all zeroes, but have otherwise correct settings
		{
			fmt.Sprintf("%s/1998-10", envelope.Data.Links.Self),
			models.EnvelopeMonth{
				Envelope: models.Envelope{
					EnvelopeCreate: models.EnvelopeCreate{
						Name: "Utilities",
					},
				},
				Month:      types.NewMonth(1998, 10),
				Spent:      decimal.NewFromFloat(-0),
				Balance:    decimal.NewFromFloat(0),
				Allocation: decimal.NewFromFloat(0),
			},
		},
	}

	// Sum alloc: 99.28

	var envelopeMonth controllers.EnvelopeMonthResponse
	for _, tt := range tests {
		r := test.Request(suite.controller, suite.T(), http.MethodGet, tt.path, "")
		assertHTTPStatus(suite.T(), &r, http.StatusOK)

		suite.decodeResponse(&r, &envelopeMonth)
		assert.Equal(suite.T(), tt.envelopeMonth.Name, envelopeMonth.Data.Name)
		assert.Equal(suite.T(), tt.envelopeMonth.Month, envelopeMonth.Data.Month)
		assert.True(suite.T(), envelopeMonth.Data.Spent.Equal(tt.envelopeMonth.Spent), "Monthly spent calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Spent, envelopeMonth.Data.Spent, envelopeMonth.Data)
		assert.True(suite.T(), envelopeMonth.Data.Balance.Equal(tt.envelopeMonth.Balance), "Monthly balance calculation for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Balance, envelopeMonth.Data.Balance, envelopeMonth.Data)
		assert.True(suite.T(), envelopeMonth.Data.Allocation.Equal(tt.envelopeMonth.Allocation), "Monthly allocation fetch for %v is wrong: should be %v, but is %v: %#v", envelopeMonth.Data.Month, tt.envelopeMonth.Allocation, envelopeMonth.Data.Allocation, envelopeMonth.Data)
	}
}

func (suite *TestSuiteStandard) TestEnvelopeMonthInvalid() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{})

	// Test that non-parseable requests produce an error
	r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/Stonks!", envelope.Data.Links.Self), "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestEnvelopeMonthNoEnvelope() {
	r := test.Request(suite.controller, suite.T(), http.MethodGet, "https://example.com/v1/envelopes/510ffa95-e445-43cc-8abc-da8e2c20ea5c/2022-04", "")
	assertHTTPStatus(suite.T(), &r, http.StatusNotFound)
}

// TestEnvelopeMonthZero tests that we return a HTTP Bad Request when requesting data for the zero timestamp.
func (suite *TestSuiteStandard) TestEnvelopeMonthZero() {
	e := suite.createTestEnvelope(models.EnvelopeCreate{})
	r := test.Request(suite.controller, suite.T(), http.MethodGet, fmt.Sprintf("%s/0001-01", e.Data.Links.Self), "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateEnvelope() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{Name: "New envelope", Note: "Keks is a cuddly cat"})

	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, envelope.Data.Links.Self, map[string]any{
		"name": "Updated new envelope for testing",
		"note": "",
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusOK)

	var updatedEnvelope controllers.EnvelopeResponse
	suite.decodeResponse(&recorder, &updatedEnvelope)

	assert.Equal(suite.T(), "", updatedEnvelope.Data.Note)
	assert.Equal(suite.T(), "Updated new envelope for testing", updatedEnvelope.Data.Name)
}

func (suite *TestSuiteStandard) TestUpdateEnvelopeBrokenJSON() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{Name: "New envelope", Note: "Keks is a cuddly cat"})
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, envelope.Data.Links.Self, `{ "name": 2" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateEnvelopeInvalidType() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{Name: "New envelope", Note: "Keks is a cuddly cat"})
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, envelope.Data.Links.Self, map[string]any{
		"name": 2,
	})
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateEnvelopeInvalidCategoryID() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{Name: "New envelope", Note: "Keks is a cuddly cat"})

	// Sets the CategoryID to uuid.Nil
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, envelope.Data.Links.Self, models.EnvelopeCreate{})
	assertHTTPStatus(suite.T(), &recorder, http.StatusBadRequest)
}

func (suite *TestSuiteStandard) TestUpdateNonExistingEnvelope() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodPatch, "http://example.com/v1/envelopes/dcf472ba-a64e-4f0f-900e-a789319e432c", `{ "name": "2" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteEnvelope() {
	e := suite.createTestEnvelope(models.EnvelopeCreate{Name: "Delete me!"})
	r := test.Request(suite.controller, suite.T(), http.MethodDelete, e.Data.Links.Self, "")
	assertHTTPStatus(suite.T(), &r, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestDeleteNonExistingEnvelope() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, "http://example.com/v1/envelopes/21a300da-d8b4-478d-8e85-95cb7982cbca", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNotFound)
}

func (suite *TestSuiteStandard) TestDeleteEnvelopeWithBody() {
	envelope := suite.createTestEnvelope(models.EnvelopeCreate{Name: "Delete this envelope"})
	recorder := test.Request(suite.controller, suite.T(), http.MethodDelete, envelope.Data.Links.Self, `{ "name": "test name 23" }`)
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
}
