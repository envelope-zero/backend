package controllers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
)

// MatchRuleQueryFilter contains the fields that Match Rules can be filtered with.
type MatchRuleQueryFilter struct {
	Priority  uint   `form:"month"`   // By priority
	Match     string `form:"match"`   // By match
	AccountID string `form:"account"` // By ID of the account they map to
}

// Parse returns a models.MatchRuleCreate struct that represents the MatchRuleQueryFilter.
func (f MatchRuleQueryFilter) Parse(c *gin.Context) (models.MatchRuleCreate, httperrors.Error) {
	envelopeID, err := httputil.UUIDFromString(f.AccountID)
	if !err.Nil() {
		return models.MatchRuleCreate{}, err
	}

	var month QueryMonth
	if err := c.Bind(&month); err != nil {
		e := httperrors.Parse(c, err)
		return models.MatchRuleCreate{}, e
	}

	return models.MatchRuleCreate{
		Priority:  f.Priority,
		Match:     f.Match,
		AccountID: envelopeID,
	}, httperrors.Error{}
}

type ResponseMatchRule struct {
	Error string    `json:"error" example:"A human readable error message"` // This field contains a human readable error message
	Data  MatchRule `json:"data"`                                           // This field contains the MatchRule data
}

type MatchRule struct {
	models.MatchRule
	Links struct {
		Self string `json:"self" example:"https://example.com/api/v2/match-rules/95685c82-53c6-455d-b235-f49960b73b21"` // The match rule itself
	} `json:"links"`
}

func (r *MatchRule) links(c *gin.Context) {
	r.Links.Self = fmt.Sprintf("%s/v2/match-rules/%s", c.GetString(string(database.ContextURL)), r.ID)
}

func (co Controller) getMatchRule(c *gin.Context, id uuid.UUID) (MatchRule, bool) {
	m, ok := getResourceByIDAndHandleErrors[models.MatchRule](c, co, id)
	if !ok {
		return MatchRule{}, false
	}

	r := MatchRule{
		MatchRule: m,
	}

	r.links(c)
	return r, true
}

// RegisterMatchRuleRoutes registers the routes for matchRules with
// the RouterGroup that is passed.
func (co Controller) RegisterMatchRuleRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsMatchRuleList)
		r.GET("", co.GetMatchRules)
		r.POST("", co.CreateMatchRules)
	}

	// MatchRule with ID
	{
		r.OPTIONS("/:id", co.OptionsMatchRuleDetail)
		r.GET("/:id", co.GetMatchRule)
		r.PATCH("/:id", co.UpdateMatchRule)
		r.DELETE("/:id", co.DeleteMatchRule)
	}
}

// OptionsMatchRuleList returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			MatchRules
//	@Success		204
//	@Router			/v2/match-rules [options]
//	@Deprecated		true
func (co Controller) OptionsMatchRuleList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsMatchRuleDetail returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			MatchRules
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v2/match-rules/{id} [options]
//	@Deprecated		true
func (co Controller) OptionsMatchRuleDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.MatchRule](c, co, id)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// CreateMatchRulesV2 creates matchRules
//
//	@Summary		Create matchRules
//	@Description	Creates matchRules from the list of submitted matchRule data. The response code is the highest response code number that a single matchRule creation would have caused. If it is not equal to 201, at least one matchRule has an error.
//	@Tags			MatchRules
//	@Produce		json
//	@Success		201			{object}	[]ResponseMatchRule
//	@Failure		400			{object}	[]ResponseMatchRule
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	[]ResponseMatchRule
//	@Param			matchRules	body		[]models.MatchRuleCreate	true	"MatchRules"
//	@Router			/v2/match-rules [post]
//	@Deprecated		true
func (co Controller) CreateMatchRules(c *gin.Context) {
	var matchRules []models.MatchRuleCreate

	if err := httputil.BindDataHandleErrors(c, &matchRules); err != nil {
		return
	}

	// The response list has the same length as the request list
	r := make([]ResponseMatchRule, 0, len(matchRules))

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated

	for _, create := range matchRules {
		m, err := co.createMatchRule(c, create)

		// Append the error or the successfully created transaction to the response list
		if !err.Nil() {
			r = append(r, ResponseMatchRule{Error: err.Error()})

			// The final status code is the highest HTTP status code number since this also
			// represents the priority we
			if err.Status > status {
				status = err.Status
			}
		} else {
			o, ok := co.getMatchRule(c, m.ID)
			if !ok {
				return
			}
			r = append(r, ResponseMatchRule{Data: o})
		}
	}

	c.JSON(status, r)
}

// GetMatchRules returns a list of matchRules matching the search parameters
//
//	@Summary		Get matchRules
//	@Description	Returns a list of matchRules
//	@Tags			MatchRules
//	@Produce		json
//	@Success		200			{object}	[]models.MatchRule
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			priority	query		uint	false	"Filter by priority"
//	@Param			match		query		string	false	"Filter by match"
//	@Param			account		query		string	false	"Filter by account ID"
//	@Router			/v2/match-rules [get]
//	@Deprecated		true
func (co Controller) GetMatchRules(c *gin.Context) {
	var filter MatchRuleQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the parameters set in the query string
	queryFields, _ := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.Parse(c)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}

	var matchRules []models.MatchRule
	if !queryAndHandleErrors(c, co.DB.Where(&models.MatchRule{
		MatchRuleCreate: create,
	}, queryFields...).Find(&matchRules)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	if len(matchRules) == 0 {
		matchRules = make([]models.MatchRule, 0)
	}

	c.JSON(http.StatusOK, matchRules)
}

// GetMatchRule returns data about a specific matchRule
//
//	@Summary		Get matchRule
//	@Description	Returns a specific matchRule
//	@Tags			MatchRules
//	@Produce		json
//	@Success		200	{object}	models.MatchRule
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v2/match-rules/{id} [get]
//	@Deprecated		true
func (co Controller) GetMatchRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	matchRuleObject, ok := getResourceByIDAndHandleErrors[models.MatchRule](c, co, id)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, matchRuleObject)
}

// UpdateMatchRule updates matchRule data
//
//	@Summary		Update matchRule
//	@Description	Update an matchRule. Only values to be updated need to be specified.
//	@Tags			MatchRules
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	models.MatchRule
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			id			path		string					true	"ID formatted as string"
//	@Param			matchRule	body		models.MatchRuleCreate	true	"MatchRule"
//	@Router			/v2/match-rules/{id} [patch]
//	@Deprecated		true
func (co Controller) UpdateMatchRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	matchRule, ok := getResourceByIDAndHandleErrors[models.MatchRule](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFieldsHandleErrors(c, models.MatchRuleCreate{})
	if err != nil {
		return
	}

	var data models.MatchRule
	if err := httputil.BindDataHandleErrors(c, &data.MatchRuleCreate); err != nil {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Model(&matchRule).Select("", updateFields...).Updates(data)) {
		return
	}

	c.JSON(http.StatusOK, matchRule)
}

// DeleteMatchRule deletes an matchRule
//
//	@Summary		Delete matchRule
//	@Description	Deletes an matchRule
//	@Tags			MatchRules
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v2/match-rules/{id} [delete]
//	@Deprecated		true
func (co Controller) DeleteMatchRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	matchRule, ok := getResourceByIDAndHandleErrors[models.MatchRule](c, co, id)
	if !ok {
		return
	}

	// MatchRules are hard deleted instantly to avoid conflicts for the UNIQUE(id,month)
	if !queryAndHandleErrors(c, co.DB.Unscoped().Delete(&matchRule)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
