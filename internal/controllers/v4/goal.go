package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v7/internal/httputil"
	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/envelope-zero/backend/v7/internal/types"
	ez_uuid "github.com/envelope-zero/backend/v7/internal/uuid"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

func RegisterGoalRoutes(r *gin.RouterGroup) {
	{
		r.OPTIONS("", OptionsGoals)
		r.GET("", GetGoals)
		r.POST("", CreateGoals)
	}
	{
		r.OPTIONS("/:id", OptionsGoalDetail)
		r.GET("/:id", GetGoal)
		r.PATCH("/:id", UpdateGoal)
		r.DELETE("/:id", DeleteGoal)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Goals
// @Success		204
// @Router			/v4/goals [options]
func OptionsGoals(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Goals
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/goals/{id} [options]
func OptionsGoalDetail(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.First(&models.Goal{}, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// @Summary		Create goals
// @Description	Creates new goals
// @Tags			Goals
// @Produce		json
// @Success		201		{object}	GoalCreateResponse
// @Failure		400		{object}	GoalCreateResponse
// @Failure		404		{object}	GoalCreateResponse
// @Failure		500		{object}	GoalCreateResponse
// @Param			goals	body		[]GoalEditable	true	"Goals"
// @Router			/v4/goals [post]
func CreateGoals(c *gin.Context) {
	var goals []GoalEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &goals)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := GoalCreateResponse{}

	for _, create := range goals {
		goal := create.model()
		err = models.DB.Create(&goal).Error
		if err != nil {
			status = r.appendError(err, status)
			continue
		}

		// Transform for the API and append
		apiResource := newGoal(c, goal)
		r.Data = append(r.Data, GoalResponse{Data: &apiResource})
	}

	c.JSON(status, r)
}

// @Summary		Get goals
// @Description	Returns a list of goals
// @Tags			Goals
// @Produce		json
// @Success		200	{object}	GoalListResponse
// @Failure		400	{object}	GoalListResponse
// @Failure		500	{object}	GoalListResponse
// @Router			/v4/goals [get]
// @Param			name				query	string	false	"Filter by name"
// @Param			note				query	string	false	"Filter by note"
// @Param			search				query	string	false	"Search for this text in name and note"
// @Param			archived			query	bool	false	"Is the goal archived?"
// @Param			envelope			query	string	false	"Filter by envelope ID"
// @Param			month				query	string	false	"Month of the goal. Ignores exact time, matches on the month of the RFC3339 timestamp provided."
// @Param			fromMonth			query	string	false	"Goals for this and later months. Ignores exact time, matches on the month of the RFC3339 timestamp provided."
// @Param			untilMonth			query	string	false	"Goals for this and earlier months. Ignores exact time, matches on the month of the RFC3339 timestamp provided."
// @Param			amount				query	string	false	"Filter by amount"
// @Param			amountLessOrEqual	query	string	false	"Amount less than or equal to this"
// @Param			amountMoreOrEqual	query	string	false	"Amount more than or equal to this"
// @Param			offset				query	uint	false	"The offset of the first goal returned. Defaults to 0."
// @Param			limit				query	int		false	"Maximum number of goal to return. Defaults to 50."
func GetGoals(c *gin.Context) {
	var filter GoalQueryFilter

	if err := c.Bind(&filter); err != nil {
		s := err.Error()
		c.JSON(http.StatusBadRequest, GoalListResponse{
			Error: &s,
		})
		return
	}

	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	where, err := filter.model()
	if err != nil {
		s := err.Error()
		c.JSON(status(err), GoalListResponse{
			Error: &s,
		})
		return
	}

	q := models.DB.
		Order("date(goals.month) ASC, goals.name ASC").
		Where(&where, queryFields...)

	q = stringFilters(models.DB, q, setFields, filter.Name, filter.Note, filter.Search)

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to 50 Accounts and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = filter.Limit
	}
	q = q.Limit(limit)

	if !where.Month.IsZero() {
		q = q.Where("goals.month >= date(?)", where.Month).Where("goals.month < date(?)", where.Month.AddDate(0, 1))
	}

	if filter.FromMonth != "" {
		fromMonth, e := types.ParseMonth(filter.FromMonth)
		if e != nil {
			s := e.Error()
			c.JSON(http.StatusBadRequest, GoalListResponse{
				Error: &s,
			})
		}
		q = q.Where("goals.month >= date(?)", fromMonth)
	}

	if filter.UntilMonth != "" {
		untilMonth, e := types.ParseMonth(filter.UntilMonth)
		if e != nil {
			s := e.Error()
			c.JSON(http.StatusBadRequest, GoalListResponse{
				Error: &s,
			})
		}
		q = q.Where("goals.month < date(?)", untilMonth.AddDate(0, 1))
	}

	if !filter.AmountLessOrEqual.IsZero() {
		q = q.Where("goals.amount <= ?", filter.AmountLessOrEqual)
	}

	if !filter.AmountMoreOrEqual.IsZero() {
		q = q.Where("goals.amount >= ?", filter.AmountMoreOrEqual)
	}

	if filter.CategoryID != ez_uuid.Nil {
		q = q.
			Joins("JOIN envelopes AS category_filter_envelopes on category_filter_envelopes.id = goals.envelope_id").
			Joins("JOIN categories AS category_filter_categories on category_filter_categories.id = category_filter_envelopes.category_id").
			Where("category_filter_categories.id = ?", filter.CategoryID.UUID)
	}

	if filter.BudgetID != ez_uuid.Nil {
		q = q.
			Joins("JOIN envelopes on envelopes.id = goals.envelope_id").
			Joins("JOIN categories on categories.id = envelopes.category_id").
			Joins("JOIN budgets on budgets.id = categories.budget_id").
			Where("budgets.id = ?", filter.BudgetID.UUID)
	}

	var goals []models.Goal
	err = q.Find(&goals).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), GoalListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = q.Limit(-1).Offset(-1).Count(&count).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalListResponse{
			Error: &e,
		})
		return
	}

	// Transform resources to their API representation
	data := make([]Goal, 0, len(goals))
	for _, goal := range goals {
		data = append(data, newGoal(c, goal))
	}

	c.JSON(http.StatusOK, GoalListResponse{
		Data: data,
		Pagination: &Pagination{
			Count:  len(data),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Get goal
// @Description	Returns a specific goal
// @Tags			Goals
// @Produce		json
// @Success		200	{object}	GoalResponse
// @Failure		400	{object}	GoalResponse
// @Failure		404	{object}	GoalResponse
// @Failure		500	{object}	GoalResponse
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/goals/{id} [get]
func GetGoal(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalResponse{
			Error: &e,
		})
		return
	}

	var goal models.Goal
	err = models.DB.First(&goal, uri.ID).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalResponse{
			Error: &e,
		})
		return
	}

	apiResource := newGoal(c, goal)
	c.JSON(http.StatusOK, GoalResponse{Data: &apiResource})
}

// @Summary		Update goal
// @Description	Updates an existing goal. Only values to be updated need to be specified.
// @Tags			Goals
// @Accept			json
// @Produce		json
// @Success		200		{object}	GoalResponse
// @Failure		400		{object}	GoalResponse
// @Failure		404		{object}	GoalResponse
// @Failure		500		{object}	GoalResponse
// @Param			id		path		URIID			true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			goal	body		GoalEditable	true	"Goal"
// @Router			/v4/goals/{id} [patch]
func UpdateGoal(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalResponse{
			Error: &e,
		})
		return
	}

	var goal models.Goal
	err = models.DB.First(&goal, uri.ID).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalResponse{
			Error: &e,
		})
		return
	}

	// Get the fields that are set to be updated
	updateFields, err := httputil.GetBodyFields(c, GoalEditable{})
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalResponse{
			Error: &e,
		})
		return
	}

	// Bind the data for the patch
	var data GoalEditable
	err = httputil.BindData(c, &data)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalResponse{
			Error: &e,
		})
		return
	}

	err = models.DB.Model(&goal).Select("", updateFields...).Updates(data.model()).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), GoalResponse{
			Error: &e,
		})
		return
	}

	apiResource := newGoal(c, goal)
	c.JSON(http.StatusOK, GoalResponse{Data: &apiResource})
}

// @Summary		Delete goal
// @Description	Deletes a goal
// @Tags			Goals
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/goals/{id} [delete]
func DeleteGoal(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var goal models.Goal
	err = models.DB.First(&goal, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.Delete(&goal).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
