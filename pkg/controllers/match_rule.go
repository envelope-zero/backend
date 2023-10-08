package controllers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
)

type MatchRuleQueryFilter struct {
	Priority  uint   `form:"month"`   // By priority
	Match     string `form:"match"`   // By match
	AccountID string `form:"account"` // By ID of the account they map to
}

func (f MatchRuleQueryFilter) Parse(c *gin.Context) (models.MatchRuleCreate, bool) {
	envelopeID, ok := httputil.UUIDFromString(c, f.AccountID)
	if !ok {
		return models.MatchRuleCreate{}, false
	}

	var month QueryMonth
	if err := c.Bind(&month); err != nil {
		httperrors.Handler(c, err)
		return models.MatchRuleCreate{}, false
	}

	return models.MatchRuleCreate{
		Priority:  f.Priority,
		Match:     f.Match,
		AccountID: envelopeID,
	}, true
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
		r.OPTIONS("/:matchRuleId", co.OptionsMatchRuleDetail)
		r.GET("/:matchRuleId", co.GetMatchRule)
		r.PATCH("/:matchRuleId", co.UpdateMatchRule)
		r.DELETE("/:matchRuleId", co.DeleteMatchRule)
	}
}

// OptionsMatchRuleList returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			MatchRules
//	@Success		204
//	@Router			/v2/match-rules [options]
func (co Controller) OptionsMatchRuleList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsMatchRuleDetail returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			MatchRules
//	@Success		204
//	@Param			matchRuleId	path	string	true	"ID formatted as string"
//	@Router			/v2/match-rules/{matchRuleId} [options]
func (co Controller) OptionsMatchRuleDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("matchRuleId"))
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
//	@Success		201	{object}	[]ResponseMatchRule
//	@Failure		400	{object}	[]ResponseMatchRule
//	@Failure		404
//	@Failure		500			{object}	[]ResponseMatchRule
//	@Param			matchRules	body		[]models.MatchRuleCreate	true	"MatchRules"
//	@Router			/v2/match-rules [post]
func (co Controller) CreateMatchRules(c *gin.Context) {
	var matchRules []models.MatchRule

	if err := httputil.BindData(c, &matchRules); err != nil {
		return
	}

	// The response list has the same length as the request list
	r := make([]ResponseMatchRule, 0, len(matchRules))

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated

	for _, o := range matchRules {
		o, err := co.createMatchRule(c, o)

		// Append the error or the successfully created transaction to the response list
		if !err.Nil() {
			r = append(r, ResponseMatchRule{Error: err.Error()})

			// The final status code is the highest HTTP status code number since this also
			// represents the priority we
			if err.Status > status {
				status = err.Status
			}
		} else {
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
//	@Success		200	{object}	[]models.MatchRule
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			priority	query		uint	false	"Filter by priority"
//	@Param			match		query		string	false	"Filter by match"
//	@Param			account		query		string	false	"Filter by account ID"
//	@Router			/v2/match-rules [get]
func (co Controller) GetMatchRules(c *gin.Context) {
	var filter MatchRuleQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the parameters set in the query string
	queryFields, _ := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, ok := filter.Parse(c)
	if !ok {
		return
	}

	var matchRules []models.MatchRule
	if !queryWithRetry(c, co.DB.Where(&models.MatchRule{
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
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			matchRuleId	path		string	true	"ID formatted as string"
//	@Router			/v2/match-rules/{matchRuleId} [get]
func (co Controller) GetMatchRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("matchRuleId"))
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
//	@Success		200	{object}	models.MatchRule
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			matchRuleId	path		string					true	"ID formatted as string"
//	@Param			matchRule	body		models.MatchRuleCreate	true	"MatchRule"
//	@Router			/v2/match-rules/{matchRuleId} [patch]
func (co Controller) UpdateMatchRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("matchRuleId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	matchRule, ok := getResourceByIDAndHandleErrors[models.MatchRule](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.MatchRuleCreate{})
	if err != nil {
		return
	}

	var data models.MatchRule
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&matchRule).Select("", updateFields...).Updates(data)) {
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
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			matchRuleId	path		string	true	"ID formatted as string"
//	@Router			/v2/match-rules/{matchRuleId} [delete]
func (co Controller) DeleteMatchRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("matchRuleId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	matchRule, ok := getResourceByIDAndHandleErrors[models.MatchRule](c, co, id)
	if !ok {
		return
	}

	// MatchRules are hard deleted instantly to avoid conflicts for the UNIQUE(id,month)
	if !queryWithRetry(c, co.DB.Unscoped().Delete(&matchRule)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// createMatchRule creates a single matchRule after verifying it is a valid matchRule.
func (co Controller) createMatchRule(c *gin.Context, r models.MatchRule) (models.MatchRule, httperrors.ErrorStatus) {
	// Check that the referenced account exists
	_, err := getResourceByID[models.Account](c, co, r.AccountID)
	if !err.Nil() {
		return r, err
	}

	// Create the resource
	dbErr := co.DB.Create(&r).Error
	if dbErr != nil {
		return models.MatchRule{}, httperrors.GenericDBError[models.MatchRule](r, c, dbErr)
	}

	return r, httperrors.ErrorStatus{}
}
