package controllers_test

import (
	"net/http"
	"testing"

	"github.com/envelope-zero/backend/v2/pkg/controllers"
	"github.com/envelope-zero/backend/v2/pkg/models"
	"github.com/envelope-zero/backend/v2/test"
)

// TODO: migrate all createTest* methods to functions with *testing.T as first argument.
func (suite *TestSuiteStandard) createTestRenameRule(t *testing.T, c models.RenameRuleCreate, expectedStatus ...int) controllers.RenameRuleResponse {
	// Default to 201 Creted as expected status
	if len(expectedStatus) == 0 {
		expectedStatus = append(expectedStatus, http.StatusCreated)
	}

	rules := []models.RenameRuleCreate{c}

	r := test.Request(suite.controller, t, http.MethodPost, "http://example.com/v2/rename-rules", rules)
	assertHTTPStatus(t, &r, expectedStatus...)

	var responseRules []controllers.RenameRuleResponse
	suite.decodeResponse(&r, &responseRules)

	return responseRules[0]
}
