package v5_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	v5 "github.com/envelope-zero/backend/v5/pkg/controllers/v5"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/v5", func(_ *gin.Context) {
		v5.Get(c)
	})

	// Test contexts cannot be injected any middleware, therefore
	// this only tests the path, not the host
	l := v5.Response{
		Links: v5.Links{
			// Accounts:     "/v5/accounts",
			Budgets: "/v5/budgets",
			// Categories:   "/v5/categories",
			// Envelopes:    "/v5/envelopes",
			// Goals:        "/v5/goals",
			// Import:       "/v5/import",
			// MatchRules:   "/v5/match-rules",
			// Months:       "/v5/months",
			// Transactions: "/v5/transactions",
		},
	}

	var lr v5.Response

	c.Request, _ = http.NewRequest(http.MethodGet, "http://example.com/v5", nil)
	r.ServeHTTP(w, c.Request)

	test.DecodeResponse(t, w, &lr)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, l, lr)
}
