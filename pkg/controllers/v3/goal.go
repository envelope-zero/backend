package v3

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
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
// @Router			/v3/goals [options]
func OptionsGoals(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Goals
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/goals/{id} [options]
func OptionsGoalDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = getResourceByID[models.Goal](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
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
// @Router			/v3/goals [post]
func CreateGoals(c *gin.Context) {
	var goals []GoalEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &goals)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := GoalCreateResponse{}

	for _, create := range goals {
		goal := create.model()

		// Verify that the envelope exists. If not, append the error and move to the next goal
		_, err := getResourceByID[models.Envelope](c, create.EnvelopeID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		dbErr := models.DB.Create(&goal).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Goal](goal, c, dbErr)
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
// @Router			/v3/goals [get]
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
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, GoalListResponse{
			Error: &s,
		})
		return
	}

	q := models.DB.
		Order("date(month) ASC, name ASC").
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

	var goals []models.Goal
	err = query(c, q.Find(&goals))

	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, GoalListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalListResponse{
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
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/goals/{id} [get]
func GetGoal(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalResponse{
			Error: &e,
		})
		return
	}

	var goal models.Goal
	err = query(c, models.DB.First(&goal, id))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalResponse{
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
// @Param			id		path		string			true	"ID formatted as string"
// @Param			goal	body		GoalEditable	true	"Goal"
// @Router			/v3/goals/{id} [patch]
func UpdateGoal(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalResponse{
			Error: &e,
		})
		return
	}

	goal, err := getResourceByID[models.Goal](c, id)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalResponse{
			Error: &e,
		})
		return
	}

	// Get the fields that are set to be updated
	updateFields, err := httputil.GetBodyFields(c, GoalEditable{})
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalResponse{
			Error: &e,
		})
		return
	}

	// Bind the data for the patch
	var data GoalEditable
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalResponse{
			Error: &e,
		})
		return
	}

	// Check that the referenced envelope exists
	if slices.Contains(updateFields, "EnvelopeID") {
		_, err = getResourceByID[models.Envelope](c, data.EnvelopeID)
		if !err.Nil() {
			e := err.Error()
			c.JSON(err.Status, GoalResponse{
				Error: &e,
			})
			return
		}
	}

	err = query(c, models.DB.Model(&goal).Select("", updateFields...).Updates(data.model()))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, GoalResponse{
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
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/goals/{id} [delete]
func DeleteGoal(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	goal, err := getResourceByID[models.Goal](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, models.DB.Delete(&goal))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
