package controllers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TODO: migrate all createTest* methods to functions with *testing.T as first argument.
func (suite *TestSuiteStandard) createTestMatchRule(t *testing.T, c models.MatchRuleCreate, expectedStatus ...int) controllers.MatchRule {
	// Default to 201 Creted as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	rules := []models.MatchRuleCreate{c}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v2/match-rules", rules)
	assertHTTPStatus(t, &r, expectedStatus...)

	var responseRules []controllers.ResponseMatchRule
	suite.decodeResponse(&r, &responseRules)

	return responseRules[0].Data
}

func (suite *TestSuiteStandard) TestOptionsMatchRule() {
	path := fmt.Sprintf("%s/%s", "http://example.com/v2/match-rules", uuid.New())
	r := test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assertHTTPStatus(suite.T(), &r, http.StatusNotFound)

	r = test.Request(suite.controller, suite.T(), http.MethodOptions, "http://example.com/v2/match-rules/NotParseableAsUUID", "")
	assertHTTPStatus(suite.T(), &r, http.StatusBadRequest)

	path = suite.createTestMatchRule(suite.T(), models.MatchRuleCreate{
		AccountID: suite.createTestAccount(models.AccountCreate{}).Data.ID,
	},
	).Links.Self

	r = test.Request(suite.controller, suite.T(), http.MethodOptions, path, "")
	assertHTTPStatus(suite.T(), &r, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestMatchRuleCreate() {
	a := suite.createTestAccount(models.AccountCreate{Name: "TestMatchRuleCreate"})

	tests := []struct {
		name           string
		create         []models.MatchRuleCreate
		expectedErrors []string
		expectedStatus int
	}{
		{
			"All successful",
			[]models.MatchRuleCreate{
				{
					Priority:  10,
					Match:     "Some Match*",
					AccountID: a.Data.ID,
				},
				{
					Priority:  10,
					Match:     "Bank*",
					AccountID: a.Data.ID,
				},
			},
			[]string{
				"",
				"",
			},
			http.StatusCreated,
		},
		{
			"Second fails",
			[]models.MatchRuleCreate{
				{
					Priority:  10,
					Match:     "Bank*",
					AccountID: a.Data.ID,
				},
				{
					Priority:  10,
					Match:     "Bank*",
					AccountID: uuid.New(),
				},
			},
			[]string{
				"",
				"there is no Account with this ID",
			},
			http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v2/match-rules", tt.create)
			assertHTTPStatus(t, &r, tt.expectedStatus)

			var tr []controllers.ResponseMatchRule
			suite.decodeResponse(&r, &tr)

			for i, r := range tr {
				assert.Equal(t, tt.expectedErrors[i], r.Error)

				if tt.expectedErrors[i] == "" {
					assert.Equal(t, fmt.Sprintf("http://example.com/v2/match-rules/%s", r.Data.ID), r.Data.Links.Self)
				}
			}
		})
	}
}
