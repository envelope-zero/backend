package controllers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/envelope-zero/backend/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestFetchErrorHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	controllers.FetchErrorHandler(c, errors.New("Testing error"))

	test.AssertHTTPStatus(t, http.StatusInternalServerError, recorder)

	var apiResponse test.APIResponse
	err := json.NewDecoder(recorder.Body).Decode(&apiResponse)
	if err != nil {
		assert.Fail(t, "Unable to parse response from server %q into APIListResponse, '%v'", recorder.Body, err)
	}

	assert.Equal(t, "An error occured on the server during your request, please contact your server administrator. The request id is '', send this to your server administrator to help them finding the problem.", apiResponse.Error)
}
