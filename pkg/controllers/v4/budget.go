package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

// RegisterBudgetRoutes registers the routes for Budgets with
// the RouterGroup that is passed.
func RegisterBudgetRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsBudgetList)
		r.GET("", GetBudgets)
		r.POST("", CreateBudgets)
	}

	// Budget with ID
	{
		r.OPTIONS("/:id", OptionsBudgetDetail)
		r.GET("/:id", GetBudget)
		r.PATCH("/:id", UpdateBudget)
		r.DELETE("/:id", DeleteBudget)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Budgets
// @Success		204
// @Router			/v4/budgets [options]
func OptionsBudgetList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Budgets
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/budgets/{id} [options]
func OptionsBudgetDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = getModelByID[models.Budget](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// @Summary		Create budget
// @Description	Creates a new budget
// @Tags			Budgets
// @Accept			json
// @Produce		json
// @Success		201		{object}	BudgetCreateResponse
// @Failure		400		{object}	BudgetCreateResponse
// @Failure		500		{object}	BudgetCreateResponse
// @Param			budget	body		[]BudgetEditable	true	"Budget"
// @Router			/v4/budgets [post]
func CreateBudgets(c *gin.Context) {
	var budgets []BudgetEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &budgets)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, BudgetCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := BudgetCreateResponse{}

	for _, editable := range budgets {
		budget := editable.model()

		dbErr := models.DB.Create(&budget).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Budget](budget, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		data := newBudget(c, budget)
		r.Data = append(r.Data, BudgetResponse{Data: &data})
	}

	c.JSON(status, r)
}

// @Summary		List budgets
// @Description	Returns a list of budgets
// @Tags			Budgets
// @Produce		json
// @Success		200	{object}	BudgetListResponse
// @Failure		500	{object}	BudgetListResponse
// @Router			/v4/budgets [get]
// @Param			name		query	string	false	"Filter by name"
// @Param			note		query	string	false	"Filter by note"
// @Param			currency	query	string	false	"Filter by currency"
// @Param			search		query	string	false	"Search for this text in name and note"
// @Param			offset		query	uint	false	"The offset of the first Budget returned. Defaults to 0."
// @Param			limit		query	int		false	"Maximum number of Budgets to return. Defaults to 50."
func GetBudgets(c *gin.Context) {
	var filter BudgetQueryFilter

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we're filtering for
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	var budgets []models.Budget

	// Always sort by name
	q := models.DB.
		Order("name ASC").
		Where(filter.model(), queryFields...)

	q = stringFilters(models.DB, q, setFields, filter.Name, filter.Note, filter.Search)

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to all Budgets and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = filter.Limit
	}
	q = q.Limit(limit)

	err := query(c, q.Find(&budgets))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, BudgetListResponse{
			Error: &e,
		})
		return
	}

	apiResources := make([]Budget, 0)
	for _, budget := range budgets {
		apiResources = append(apiResources, newBudget(c, budget))
	}

	c.JSON(http.StatusOK, BudgetListResponse{
		Data: apiResources,
		Pagination: &Pagination{
			Count:  len(apiResources),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Get budget
// @Description	Returns a specific budget
// @Tags			Budgets
// @Produce		json
// @Success		200	{object}	BudgetResponse
// @Failure		400	{object}	BudgetResponse
// @Failure		404	{object}	BudgetResponse
// @Failure		500	{object}	BudgetResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/budgets/{id} [get]
func GetBudget(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	m, err := getModelByID[models.Budget](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	apiResource := newBudget(c, m)
	c.JSON(http.StatusOK, BudgetResponse{Data: &apiResource})
}

// @Summary		Update budget
// @Description	Update an existing budget. Only values to be updated need to be specified.
// @Tags			Budgets
// @Accept			json
// @Produce		json
// @Success		200		{object}	BudgetResponse
// @Failure		400		{object}	BudgetResponse
// @Failure		404		{object}	BudgetResponse
// @Failure		500		{object}	BudgetResponse
// @Param			id		path		string			true	"ID formatted as string"
// @Param			budget	body		BudgetEditable	true	"Budget"
// @Router			/v4/budgets/{id} [patch]
func UpdateBudget(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	budget, err := getModelByID[models.Budget](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, BudgetEditable{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	var data BudgetEditable
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	err = query(c, models.DB.Model(&budget).Select("", updateFields...).Updates(data))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	apiResource := newBudget(c, budget)
	c.JSON(http.StatusOK, BudgetResponse{Data: &apiResource})
}

// @Summary		Delete budget
// @Description	Deletes a budget
// @Tags			Budgets
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v4/budgets/{id} [delete]
func DeleteBudget(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	budget, err := getModelByID[models.Budget](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, models.DB.Delete(&budget))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
