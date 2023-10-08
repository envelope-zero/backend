package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/controllers"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/envelope-zero/backend/v3/test"
)

// TODO: migrate all createTest* methods to functions with *testing.T as first argument.
func (suite *TestSuiteStandard) createTestMatchRule(t *testing.T, c models.MatchRuleCreate, expectedStatus ...int) models.MatchRule {
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
