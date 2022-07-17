package httputil_test

import (
	"net/url"
	"testing"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/stretchr/testify/assert"
)

func TestGetFields(t *testing.T) {
	url, _ := url.Parse("http://example.com/api/v1/accounts?budget=87645467-ad8a-4e16-ae7f-9d879b45f569&onBudget=false")

	queryFields := httputil.GetFields(url, controllers.AccountQueryFilter{})

	assert.Equal(t, []interface{}{"BudgetID", "OnBudget"}, queryFields)
}
