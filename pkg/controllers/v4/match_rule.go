package v4

import (
	"fmt"
	"net/http"

	"golang.org/x/exp/slices"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
)

// RegisterMatchRuleRoutes registers the routes for matchRules with
// the RouterGroup that is passed.
func RegisterMatchRuleRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsMatchRuleList)
		r.GET("", GetMatchRules)
		r.POST("", CreateMatchRules)
	}

	// MatchRule with ID
	{
		r.OPTIONS("/:id", OptionsMatchRuleDetail)
		r.GET("/:id", GetMatchRule)
		r.PATCH("/:id", UpdateMatchRule)
		r.DELETE("/:id", DeleteMatchRule)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			MatchRules
// @Success		204
// @Router			/v4/match-rules [options]
func OptionsMatchRuleList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			MatchRules
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/match-rules/{id} [options]
func OptionsMatchRuleDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
	}

	_, err = getModelByID[models.MatchRule](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// @Summary		Create matchRules
// @Description	Creates matchRules from the list of submitted matchRule data. The response code is the highest response code number that a single matchRule creation would have caused. If it is not equal to 201, at least one matchRule has an error.
// @Tags			MatchRules
// @Produce		json
// @Success		201			{object}	MatchRuleCreateResponse
// @Failure		400			{object}	MatchRuleCreateResponse
// @Failure		404			{object}	MatchRuleCreateResponse
// @Failure		500			{object}	MatchRuleCreateResponse
// @Param			matchRules	body		[]MatchRuleEditable	true	"MatchRules"
// @Router			/v4/match-rules [post]
func CreateMatchRules(c *gin.Context) {
	var matchRules []MatchRuleEditable

	err := httputil.BindData(c, &matchRules)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := MatchRuleCreateResponse{}

	for _, editable := range matchRules {
		model, err := createMatchRule(c, editable.model())

		// Append the error or the successfully created transaction to the response list
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		data := newMatchRule(c, model)
		r.Data = append(r.Data, MatchRuleResponse{Data: &data})
	}

	c.JSON(status, r)
}

// @Summary		Get matchRules
// @Description	Returns a list of matchRules
// @Tags			MatchRules
// @Produce		json
// @Success		200			{object}	MatchRuleListResponse
// @Failure		400			{object}	MatchRuleListResponse
// @Failure		500			{object}	MatchRuleListResponse
// @Param			priority	query		uint	false	"Filter by priority"
// @Param			match		query		string	false	"Filter by match"
// @Param			account		query		string	false	"Filter by account ID"
// @Param			offset		query		uint	false	"The offset of the first Match Rule returned. Defaults to 0."
// @Param			limit		query		int		false	"Maximum number of Match Rules to return. Defaults to 50.".
// @Router			/v4/match-rules [get]
func GetMatchRules(c *gin.Context) {
	var filter MatchRuleQueryFilter
	if err := c.Bind(&filter); err != nil {
		s := httperrors.ErrInvalidQueryString.Error()
		c.JSON(http.StatusBadRequest, MatchRuleListResponse{
			Error: &s,
		})
		return
	}

	// Get the parameters set in the query string
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	model, err := filter.model()
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleListResponse{Error: &e})
		return
	}

	q := models.DB.
		Order("priority ASC, match ASC").
		Where(&model, queryFields...)

	// Filter for match containing the query string or explicitly empty one
	if filter.Match != "" {
		q = q.Where("match LIKE ?", fmt.Sprintf("%%%s%%", filter.Match))
	} else if slices.Contains(setFields, "Match") {
		q = q.Where("match = ''")
	}

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to 50 Match Rules and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = filter.Limit
	}
	q = q.Limit(limit)

	// Execute the query
	var matchRules []models.MatchRule
	err = query(c, q.Find(&matchRules))

	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleListResponse{Error: &e})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleListResponse{
			Error: &e,
		})
		return
	}

	data := make([]MatchRule, 0)
	for _, matchRule := range matchRules {
		data = append(data, newMatchRule(c, matchRule))
	}

	c.JSON(http.StatusOK, MatchRuleListResponse{
		Data: data,
		Pagination: &Pagination{
			Count:  len(data),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Get matchRule
// @Description	Returns a specific matchRule
// @Tags			MatchRules
// @Produce		json
// @Success		200	{object}	MatchRuleResponse
// @Failure		400	{object}	MatchRuleResponse
// @Failure		404	{object}	MatchRuleResponse
// @Failure		500	{object}	MatchRuleResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/match-rules/{id} [get]
func GetMatchRule(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponse{
			Error: &e,
		})
		return
	}

	model, err := getModelByID[models.MatchRule](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}
	data := newMatchRule(c, model)

	c.JSON(http.StatusOK, MatchRuleResponse{
		Data: &data,
	})
}

// @Summary		Update matchRule
// @Description	Update a matchRule. Only values to be updated need to be specified.
// @Tags			MatchRules
// @Accept			json
// @Produce		json
// @Success		200			{object}	MatchRuleResponse
// @Failure		400			{object}	MatchRuleResponse
// @Failure		404			{object}	MatchRuleResponse
// @Failure		500			{object}	MatchRuleResponse
// @Param			id			path		string				true	"ID formatted as string"
// @Param			matchRule	body		MatchRuleEditable	true	"MatchRule"
// @Router			/v4/match-rules/{id} [patch]
func UpdateMatchRule(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponse{
			Error: &e,
		})
		return
	}

	matchRule, err := getModelByID[models.MatchRule](c, id)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponse{
			Error: &e,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, MatchRuleEditable{})
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponse{
			Error: &e,
		})
		return
	}

	var data MatchRuleEditable
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponse{
			Error: &e,
		})
		return
	}

	// Check that the referenced account exists
	if slices.Contains(updateFields, "AccountID") {
		_, err = getModelByID[models.Account](c, data.AccountID)
		if !err.Nil() {
			e := err.Error()
			c.JSON(err.Status, MatchRuleResponse{
				Error: &e,
			})
			return
		}
	}

	err = query(c, models.DB.Model(&matchRule).Select("", updateFields...).Updates(data.model()))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponse{
			Error: &e,
		})
		return
	}

	apiResource := newMatchRule(c, matchRule)
	c.JSON(http.StatusOK, MatchRuleResponse{
		Data: &apiResource,
	})
}

// @Summary		Delete matchRule
// @Description	Deletes an matchRule
// @Tags			MatchRules
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/match-rules/{id} [delete]
func DeleteMatchRule(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}
	matchRule, err := getModelByID[models.MatchRule](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}

	err = query(c, models.DB.Delete(&matchRule))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// createMatchRule creates a single matchRule after verifying it is a valid matchRule.
func createMatchRule(c *gin.Context, matchRule models.MatchRule) (models.MatchRule, httperrors.Error) {
	// Check that the referenced account exists
	_, err := getModelByID[models.Account](c, matchRule.AccountID)
	if !err.Nil() {
		return matchRule, err
	}

	// Create the resource
	dbErr := models.DB.Create(&matchRule).Error
	if dbErr != nil {
		return models.MatchRule{}, httperrors.GenericDBError[models.MatchRule](matchRule, c, dbErr)
	}

	return matchRule, httperrors.Error{}
}
