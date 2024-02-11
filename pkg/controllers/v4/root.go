package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
)

func RegisterRootRoutes(r *gin.RouterGroup) {
	r.GET("", Get)
	r.DELETE("", Cleanup)
	r.OPTIONS("", Options)
}

type Response struct {
	Links Links `json:"links"` // Links for the v4 API
}

type Links struct {
	Accounts     string `json:"accounts" example:"https://example.com/api/v4/accounts"`         // URL of Account collection endpoint
	Budgets      string `json:"budgets" example:"https://example.com/api/v4/budgets"`           // URL of Budget collection endpoint
	Categories   string `json:"categories" example:"https://example.com/api/v4/categories"`     // URL of Category collection endpoint
	Envelopes    string `json:"envelopes" example:"https://example.com/api/v4/envelopes"`       // URL of Envelope collection endpoint
	Goals        string `json:"goals" example:"https://example.com/api/v4/goals"`               // URL of goal collection endpoint
	Import       string `json:"import" example:"https://example.com/api/v4/import"`             // URL of import list endpoint
	MatchRules   string `json:"matchRules" example:"https://example.com/api/v4/match-rules"`    // URL of Match Rule collection endpoint
	Months       string `json:"months" example:"https://example.com/api/v4/months"`             // URL of Month endpoint
	Transactions string `json:"transactions" example:"https://example.com/api/v4/transactions"` // URL of Transaction collection endpoint
}

// Get returns the link list for v4
//
//	@Summary		v4 API
//	@Description	Returns general information about the v4 API
//	@Tags			v4
//	@Success		200	{object}	Response
//	@Router			/v4 [get]
func Get(c *gin.Context) {
	url := c.GetString(string(models.DBContextURL))

	c.JSON(http.StatusOK, Response{
		Links: Links{
			Accounts:     url + "/v4/accounts",
			Budgets:      url + "/v4/budgets",
			Categories:   url + "/v4/categories",
			Envelopes:    url + "/v4/envelopes",
			Goals:        url + "/v4/goals",
			Import:       url + "/v4/import",
			MatchRules:   url + "/v4/match-rules",
			Months:       url + "/v4/months",
			Transactions: url + "/v4/transactions",
		},
	})
}

// Options returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			v4
//	@Success		204
//	@Router			/v4 [options]
func Options(c *gin.Context) {
	httputil.OptionsGetDelete(c)
}
