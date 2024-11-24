package v5

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/internal/httperror"
	"github.com/envelope-zero/backend/v5/pkg/httputil"
	"github.com/envelope-zero/backend/v5/pkg/models"
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
// @Router			/v5/budgets [options]
func OptionsBudgetList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Budgets
// @Success		204
// @Failure		400	{object}	httperror.Error
// @Failure		404	{object}	httperror.Error
// @Failure		500	{object}	httperror.Error
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v5/budgets/{id} [options]
func OptionsBudgetDetail(c *gin.Context) {
	// var uri URIID
	// err := c.ShouldBindUri(&uri)
	// if err != nil {
	// 	c.JSON(httperror.Status(err), httperror.New(err))
	// 	return
	// }
	var uri URIID
	if c.BindUri(&uri) != nil {
		return
	}

	err := models.DB.First(&models.Budget{}, uri.ID).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
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
// @Failure		400		{object}	httperror.Error
// @Failure		500		{object}	httperror.Error
// @Param			budgets	body		[]BudgetEditable	true	"Budgets"
// @Router			/v5/budgets [post]
func CreateBudgets(c *gin.Context) {
	var budgets []BudgetEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &budgets)
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := BudgetCreateResponse{}

	for _, editable := range budgets {
		budget := editable.model()

		err := models.DB.Create(&budget).Error
		if err != nil {
			status = r.appendError(err, status)
			continue
		}

		data := newBudget(c, budget)
		r.Data = append(r.Data, BudgetCreatedResponse{Data: data})
	}

	c.JSON(status, r)
}

// @Summary		List budgets
// @Description	Returns a list of budgets
// @Tags			Budgets
// @Produce		json
// @Success		200	{object}	BudgetListResponse
// @Failure		500	{object}	httperror.Error
// @Router			/v5/budgets [get]
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

	err := q.Find(&budgets).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	var count int64
	err = q.Limit(-1).Offset(-1).Count(&count).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	apiResources := make([]Budget, 0)
	for _, budget := range budgets {
		apiResources = append(apiResources, newBudget(c, budget))
	}

	c.JSON(http.StatusOK, BudgetListResponse{
		Data: apiResources,
		Pagination: Pagination{
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
// @Failure		400	{object}	httperror.Error
// @Failure		404	{object}	httperror.Error
// @Failure		500	{object}	httperror.Error
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v5/budgets/{id} [get]
func GetBudget(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	var budget models.Budget
	err = models.DB.First(&budget, uri.ID).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	apiResource := newBudget(c, budget)
	c.JSON(http.StatusOK, BudgetResponse{Data: apiResource})
}

// @Summary		Update budget
// @Description	Update an existing budget. Only values to be updated need to be specified.
// @Tags			Budgets
// @Accept			json
// @Produce		json
// @Success		200		{object}	BudgetResponse
// @Failure		400		{object}	httperror.Error
// @Failure		404		{object}	httperror.Error
// @Failure		500		{object}	httperror.Error
// @Param			id		path		URIID			true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			budget	body		BudgetEditable	true	"Budget"
// @Router			/v5/budgets/{id} [patch]
func UpdateBudget(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	var budget models.Budget
	err = models.DB.First(&budget, uri.ID).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	updateFields, err := httputil.GetBodyFields(c, BudgetEditable{})
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	var data BudgetEditable
	err = httputil.BindData(c, &data)
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	err = models.DB.Model(&budget).Select("", updateFields...).Updates(data).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	apiResource := newBudget(c, budget)
	c.JSON(http.StatusOK, BudgetResponse{Data: apiResource})
}

// @Summary		Delete budget
// @Description	Deletes a budget
// @Tags			Budgets
// @Success		204
// @Failure		400	{object}	httperror.Error
// @Failure		404	{object}	httperror.Error
// @Failure		500	{object}	httperror.Error
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v5/budgets/{id} [delete]
func DeleteBudget(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	var budget models.Budget
	err = models.DB.First(&budget, uri.ID).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	err = models.DB.Delete(&budget).Error
	if err != nil {
		c.JSON(httperror.Status(err), httperror.New(err))
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
