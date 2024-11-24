package v5

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
	"github.com/gin-gonic/gin"
)

func RegisterRootRoutes(r *gin.RouterGroup) {
	r.GET("", Get)
	// r.DELETE("", Cleanup)
	r.OPTIONS("", Options)
}

type Response struct {
	Links Links `json:"links"` // Links for the v5 API
}

type Links struct {
	// Accounts     string `json:"accounts" example:"https://example.com/api/v5/accounts"`         // URL of Account collection endpoint
	Budgets string `json:"budgets" example:"https://example.com/api/v5/budgets"` // URL of Budget collection endpoint
	// Categories   string `json:"categories" example:"https://example.com/api/v5/categories"`     // URL of Category collection endpoint
	// Envelopes    string `json:"envelopes" example:"https://example.com/api/v5/envelopes"`       // URL of Envelope collection endpoint
	// Goals        string `json:"goals" example:"https://example.com/api/v5/goals"`               // URL of goal collection endpoint
	// Import       string `json:"import" example:"https://example.com/api/v5/import"`             // URL of import list endpoint
	// MatchRules   string `json:"matchRules" example:"https://example.com/api/v5/match-rules"`    // URL of Match Rule collection endpoint
	// Months       string `json:"months" example:"https://example.com/api/v5/months"`             // URL of Month endpoint
	// Transactions string `json:"transactions" example:"https://example.com/api/v5/transactions"` // URL of Transaction collection endpoint
}

// Get returns the link list for v5
//
//	@Summary		v5 API
//	@Description	Returns general information about the v5 API
//	@Tags			v5
//	@Success		200	{object}	Response
//	@Router			/v5 [get]
func Get(c *gin.Context) {
	url := c.GetString(string(models.DBContextURL))

	c.JSON(http.StatusOK, Response{
		Links: Links{
			// Accounts:     url + "/v5/accounts",
			Budgets: url + "/v5/budgets",
			// Categories:   url + "/v5/categories",
			// Envelopes:    url + "/v5/envelopes",
			// Goals:        url + "/v5/goals",
			// Import:       url + "/v5/import",
			// MatchRules:   url + "/v5/match-rules",
			// Months:       url + "/v5/months",
			// Transactions: url + "/v5/transactions",
		},
	})
}

// Options returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			v5
//	@Success		204
//	@Router			/v5 [options]
func Options(c *gin.Context) {
	httputil.OptionsGetDelete(c)
}
