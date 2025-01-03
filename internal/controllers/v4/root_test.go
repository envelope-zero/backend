package v4_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	v4 "github.com/envelope-zero/backend/v5/internal/controllers/v4"
	"github.com/envelope-zero/backend/v5/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/v4", func(_ *gin.Context) {
		v4.Get(c)
	})

	// Test contexts cannot be injected any middleware, therefore
	// this only tests the path, not the host
	l := v4.Response{
		Links: v4.Links{
			Accounts:     "/v4/accounts",
			Budgets:      "/v4/budgets",
			Categories:   "/v4/categories",
			Envelopes:    "/v4/envelopes",
			Goals:        "/v4/goals",
			Import:       "/v4/import",
			MatchRules:   "/v4/match-rules",
			Months:       "/v4/months",
			Transactions: "/v4/transactions",
		},
	}

	var lr v4.Response

	c.Request, _ = http.NewRequest(http.MethodGet, "http://example.com/v4", nil)
	r.ServeHTTP(w, c.Request)

	test.DecodeResponse(t, w, &lr)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, l, lr)
}
