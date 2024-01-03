package v3_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	v3 "github.com/envelope-zero/backend/v4/pkg/controllers/v3"
	"github.com/envelope-zero/backend/v4/test"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/v3", func(ctx *gin.Context) {
		v3.Get(c)
	})

	// Test contexts cannot be injected any middleware, therefore
	// this only tests the path, not the host
	l := v3.Response{
		Links: v3.Links{
			Accounts:     "/v3/accounts",
			Budgets:      "/v3/budgets",
			Categories:   "/v3/categories",
			Envelopes:    "/v3/envelopes",
			Goals:        "/v3/goals",
			Import:       "/v3/import",
			MatchRules:   "/v3/match-rules",
			Months:       "/v3/months",
			Transactions: "/v3/transactions",
		},
	}

	var lr v3.Response

	c.Request, _ = http.NewRequest(http.MethodGet, "http://example.com/v3", nil)
	r.ServeHTTP(w, c.Request)

	test.DecodeResponse(t, w, &lr)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, l, lr)
}
