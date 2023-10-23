package controllers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestTransactionsCreateV2() {
	budget := suite.createTestBudget(models.BudgetCreate{})
	internalAccount := suite.createTestAccount(models.AccountCreate{External: false, BudgetID: budget.Data.ID, Name: "TestTransactionsCreate Internal"})
	externalAccount := suite.createTestAccount(models.AccountCreate{External: true, BudgetID: budget.Data.ID, Name: "TestTransactionsCreate External"})

	tests := []struct {
		name           string
		transactions   []models.TransactionCreate
		expectedStatus int
		expectedErrors []string
	}{
		{
			"One success, one fail",
			[]models.TransactionCreate{
				{
					BudgetID: uuid.New(),
					Amount:   decimal.NewFromFloat(17.23),
					Note:     "v2 non-existing budget ID",
				},
				{
					BudgetID:             budget.Data.ID,
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(57.01),
				},
			},
			http.StatusNotFound,
			[]string{
				"there is no Budget with this ID",
				"",
			},
		},
		{
			"Both succeed",
			[]models.TransactionCreate{
				{
					BudgetID:             budget.Data.ID,
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(17.23),
				},
				{
					BudgetID:             budget.Data.ID,
					SourceAccountID:      internalAccount.Data.ID,
					DestinationAccountID: externalAccount.Data.ID,
					Amount:               decimal.NewFromFloat(57.01),
				},
			},
			http.StatusCreated,
			[]string{
				"",
				"",
			},
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v2/transactions", tt.transactions)
			assertHTTPStatus(t, &r, tt.expectedStatus)

			var tr []controllers.ResponseTransactionV2
			suite.decodeResponse(&r, &tr)

			for i, transaction := range tr {
				assert.Equal(t, tt.expectedErrors[i], transaction.Error)

				if tt.expectedErrors[i] == "" {
					assert.Equal(t, fmt.Sprintf("http://example.com/v2/transactions/%s", transaction.Data.ID), transaction.Data.Links.Self)
				}
			}
		})
	}
}
