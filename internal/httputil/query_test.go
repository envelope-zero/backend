package httputil_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/envelope-zero/backend/v7/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetURLFields(t *testing.T) {
	url, _ := url.Parse("http://example.com/api/v3/accounts?budget=87645467-ad8a-4e16-ae7f-9d879b45f569&onBudget=false&name=")

	queryFields, setFields := httputil.GetURLFields(url, struct {
		Name     string `form:"name" filterField:"false"`
		Note     string `form:"note" filterField:"false"`
		BudgetID string `form:"budget"`
		OnBudget bool   `form:"onBudget"`
	}{})

	assert.Equal(t, []interface{}{"BudgetID", "OnBudget"}, queryFields)
	assert.Equal(t, []string{"Name", "BudgetID", "OnBudget"}, setFields)
}

// TestGetBodyFields verifies that GetBodyFields parses correctly.
func TestGetBodyFields(t *testing.T) {
	tests := []struct {
		name       string                             // Name of the test
		body       string                             // The body to send to the PATCH request
		status     int                                // The expected status code
		assertFunc func(w *httptest.ResponseRecorder) // Additional assertions on the response. Can be nil
	}{
		{
			"Success",
			`{ "name": "test account" }`,
			http.StatusOK,
			nil,
		},
		{
			"Field is null",
			`{ "name": null }`,
			http.StatusOK,
			func(w *httptest.ResponseRecorder) {
				assert.Equal(t, `["Name"]`, w.Body.String(), `Fields are not parsed correctly, should be ["Name"]`)
			},
		},
		{
			"Unparseable",
			`{ "name": "test account }`,
			http.StatusBadRequest,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.PATCH("/", func(_ *gin.Context) {
				fields, err := httputil.GetBodyFields(c, struct {
					Name string `json:"name"`
				}{})
				if err != nil {
					c.JSON(http.StatusBadRequest, err.Error())
				}
				c.JSON(http.StatusOK, fields)
			})

			c.Request, _ = http.NewRequest(http.MethodPatch, "https://example.com/", bytes.NewBuffer([]byte(tt.body)))
			r.ServeHTTP(w, c.Request)
			assert.Equal(t, tt.status, w.Code, "Status is wrong, return body %#v", w.Body.String())

			// Execute additional assertions
			if tt.assertFunc != nil {
				tt.assertFunc(w)
			}
		})
	}
}
