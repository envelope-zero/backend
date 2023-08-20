package controllers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
)

type RenameRuleResponse struct {
	Data models.RenameRule `json:"data"` // Data for the rename rule
}

type RenameRuleListResponse struct {
	Data []models.RenameRule `json:"data"` // List of rename rules
}

type RenameRuleQueryFilter struct {
	Priority  uint   `form:"month"`   // By priority
	Match     string `form:"match"`   // By match
	AccountID string `form:"account"` // By ID of the account they map to
}

func (f RenameRuleQueryFilter) Parse(c *gin.Context) (models.RenameRuleCreate, bool) {
	envelopeID, ok := httputil.UUIDFromString(c, f.AccountID)
	if !ok {
		return models.RenameRuleCreate{}, false
	}

	var month QueryMonth
	if err := c.Bind(&month); err != nil {
		httperrors.Handler(c, err)
		return models.RenameRuleCreate{}, false
	}

	return models.RenameRuleCreate{
		Priority:  f.Priority,
		Match:     f.Match,
		AccountID: envelopeID,
	}, true
}

// RegisterRenameRuleRoutes registers the routes for renameRules with
// the RouterGroup that is passed.
func (co Controller) RegisterRenameRuleRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsRenameRuleList)
		r.GET("", co.GetRenameRules)
		r.POST("", co.CreateRenameRules)
	}

	// RenameRule with ID
	{
		r.OPTIONS("/:renameRuleId", co.OptionsRenameRuleDetail)
		r.GET("/:renameRuleId", co.GetRenameRule)
		r.PATCH("/:renameRuleId", co.UpdateRenameRule)
		r.DELETE("/:renameRuleId", co.DeleteRenameRule)
	}
}

// OptionsRenameRuleList returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			RenameRules
//	@Success		204
//	@Router			/v2/rename-rules [options]
func (co Controller) OptionsRenameRuleList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsRenameRuleDetail returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			RenameRules
//	@Success		204
//	@Param			renameRuleId	path	string	true	"ID formatted as string"
//	@Router			/v2/rename-rules/{renameRuleId} [options]
func (co Controller) OptionsRenameRuleDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("renameRuleId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.RenameRule](c, co, id)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// CreateRenameRulesV2 creates renameRules
//
//	@Summary		Create renameRules
//	@Description	Creates renameRules from the list of submitted renameRule data. The response code is the highest response code number that a single renameRule creation would have caused. If it is not equal to 201, at least one renameRule has an error.
//	@Tags			RenameRules
//	@Produce		json
//	@Success		201	{object}	[]ResponseRenameRule
//	@Failure		400	{object}	[]ResponseRenameRule
//	@Failure		404
//	@Failure		500			{object}	[]ResponseRenameRule
//	@Param			renameRules	body		[]models.RenameRuleCreate	true	"RenameRules"
//	@Router			/v2/rename-rules [post]
func (co Controller) CreateRenameRules(c *gin.Context) {
	var renameRules []models.RenameRule

	if err := httputil.BindData(c, &renameRules); err != nil {
		return
	}

	// The response list has the same length as the request list
	r := make([]ResponseRenameRule, 0, len(renameRules))

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated

	for _, o := range renameRules {
		o, err := co.createRenameRule(c, o)

		// Append the error or the successfully created transaction to the response list
		if !err.Nil() {
			r = append(r, ResponseRenameRule{Error: err.Error()})

			// The final status code is the highest HTTP status code number since this also
			// represents the priority we
			if err.Status > status {
				status = err.Status
			}
		} else {
			r = append(r, ResponseRenameRule{Data: o})
		}
	}

	c.JSON(status, r)
}

// GetRenameRules returns a list of renameRules matching the search parameters
//
//	@Summary		Get renameRules
//	@Description	Returns a list of renameRules
//	@Tags			RenameRules
//	@Produce		json
//	@Success		200	{object}	RenameRuleListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			priority	query		uint	false	"Filter by priority"
//	@Param			match		query		string	false	"Filter by match"
//	@Param			account		query		string	false	"Filter by account ID"
//	@Router			/v2/rename-rules [get]
func (co Controller) GetRenameRules(c *gin.Context) {
	var filter RenameRuleQueryFilter
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

	var renameRules []models.RenameRule
	if !queryWithRetry(c, co.DB.Where(&models.RenameRule{
		RenameRuleCreate: create,
	}, queryFields...).Find(&renameRules)) {
		return
	}

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	if len(renameRules) == 0 {
		renameRules = make([]models.RenameRule, 0)
	}

	c.JSON(http.StatusOK, RenameRuleListResponse{Data: renameRules})
}

// GetRenameRule returns data about a specific renameRule
//
//	@Summary		Get renameRule
//	@Description	Returns a specific renameRule
//	@Tags			RenameRules
//	@Produce		json
//	@Success		200	{object}	RenameRuleResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			renameRuleId	path		string	true	"ID formatted as string"
//	@Router			/v2/rename-rules/{renameRuleId} [get]
func (co Controller) GetRenameRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("renameRuleId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	renameRuleObject, ok := getResourceByIDAndHandleErrors[models.RenameRule](c, co, id)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, RenameRuleResponse{Data: renameRuleObject})
}

// UpdateRenameRule updates renameRule data
//
//	@Summary		Update renameRule
//	@Description	Update an renameRule. Only values to be updated need to be specified.
//	@Tags			RenameRules
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	RenameRuleResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			renameRuleId	path		string					true	"ID formatted as string"
//	@Param			renameRule		body		models.RenameRuleCreate	true	"RenameRule"
//	@Router			/v2/rename-rules/{renameRuleId} [patch]
func (co Controller) UpdateRenameRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("renameRuleId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	renameRule, ok := getResourceByIDAndHandleErrors[models.RenameRule](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.RenameRuleCreate{})
	if err != nil {
		return
	}

	var data models.RenameRule
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&renameRule).Select("", updateFields...).Updates(data)) {
		return
	}

	c.JSON(http.StatusOK, RenameRuleResponse{Data: renameRule})
}

// DeleteRenameRule deletes an renameRule
//
//	@Summary		Delete renameRule
//	@Description	Deletes an renameRule
//	@Tags			RenameRules
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			renameRuleId	path		string	true	"ID formatted as string"
//	@Router			/v2/rename-rules/{renameRuleId} [delete]
func (co Controller) DeleteRenameRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("renameRuleId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	renameRule, ok := getResourceByIDAndHandleErrors[models.RenameRule](c, co, id)
	if !ok {
		return
	}

	// RenameRules are hard deleted instantly to avoid conflicts for the UNIQUE(id,month)
	if !queryWithRetry(c, co.DB.Unscoped().Delete(&renameRule)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// createRenameRule creates a single renameRule after verifying it is a valid renameRule.
func (co Controller) createRenameRule(c *gin.Context, r models.RenameRule) (models.RenameRule, httperrors.ErrorStatus) {
	// Check that the referenced account exists
	_, err := getResourceByID[models.Account](c, co, r.AccountID)
	if !err.Nil() {
		return r, err
	}

	// Create the resource
	dbErr := co.DB.Create(&r).Error
	if dbErr != nil {
		return models.RenameRule{}, httperrors.GenericDBError[models.RenameRule](r, c, dbErr)
	}

	return r, httperrors.ErrorStatus{}
}
