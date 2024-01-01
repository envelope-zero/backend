package controllers_test

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/test"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestHealthzSuccess() {
	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/healthz", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusNoContent)
}

func (suite *TestSuiteStandard) TestHealthzFail() {
	suite.CloseDB()

	recorder := test.Request(suite.controller, suite.T(), http.MethodGet, "http://example.com/healthz", "")
	assertHTTPStatus(suite.T(), &recorder, http.StatusInternalServerError)
	assert.Contains(suite.T(), test.DecodeError(suite.T(), recorder.Body.Bytes()), "There is a problem with the database connection")
}
