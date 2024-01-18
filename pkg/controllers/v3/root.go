package v3

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
)

func RegisterRootRoutes(r *gin.RouterGroup) {
	r.GET("", Get)
	r.DELETE("", Cleanup)
	r.OPTIONS("", Options)
}

type Response struct {
	Links Links `json:"links"` // Links for the v3 API
}

type Links struct {
	Accounts     string `json:"accounts" example:"https://example.com/api/v3/accounts"`         // URL of Account collection endpoint
	Budgets      string `json:"budgets" example:"https://example.com/api/v3/budgets"`           // URL of Budget collection endpoint
	Categories   string `json:"categories" example:"https://example.com/api/v3/categories"`     // URL of Category collection endpoint
	Envelopes    string `json:"envelopes" example:"https://example.com/api/v3/envelopes"`       // URL of Envelope collection endpoint
	Goals        string `json:"goals" example:"https://example.com/api/v3/goals"`               // URL of goal collection endpoint
	Import       string `json:"import" example:"https://example.com/api/v3/import"`             // URL of import list endpoint
	MatchRules   string `json:"matchRules" example:"https://example.com/api/v3/match-rules"`    // URL of Match Rule collection endpoint
	Months       string `json:"months" example:"https://example.com/api/v3/months"`             // URL of Month endpoint
	Transactions string `json:"transactions" example:"https://example.com/api/v3/transactions"` // URL of Transaction collection endpoint
}

// Get returns the link list for v3
//
//	@Summary		v3 API
//	@Description	Returns general information about the v3 API
//	@Tags			v3
//	@Success		200	{object}	Response
//	@Router			/v3 [get]
//
//	@Deprecated		true
func Get(c *gin.Context) {
	url := c.GetString(string(models.DBContextURL))

	c.JSON(http.StatusOK, Response{
		Links: Links{
			Accounts:     url + "/v3/accounts",
			Budgets:      url + "/v3/budgets",
			Categories:   url + "/v3/categories",
			Envelopes:    url + "/v3/envelopes",
			Goals:        url + "/v3/goals",
			Import:       url + "/v3/import",
			MatchRules:   url + "/v3/match-rules",
			Months:       url + "/v3/months",
			Transactions: url + "/v3/transactions",
		},
	})
}

// Options returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			v3
//	@Success		204
//	@Router			/v3 [options]
//
//	@Deprecated		true
func Options(c *gin.Context) {
	httputil.OptionsGetDelete(c)
}
