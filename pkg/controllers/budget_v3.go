package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

type BudgetQueryFilterV3 struct {
	Name     string `form:"name" filterField:"false"`   // By name
	Note     string `form:"note" filterField:"false"`   // By note
	Currency string `form:"currency"`                   // By currency
	Search   string `form:"search" filterField:"false"` // By string in name or note
	Offset   uint   `form:"offset" filterField:"false"` // The offset of the first Budget returned. Defaults to 0.
	Limit    int    `form:"limit" filterField:"false"`  // Maximum number of Budgets to return. Defaults to 50.
}

// Budget is the API v3 representation of a Budget.
type BudgetV3 struct {
	models.Budget
	Links struct {
		Self         string `json:"self" example:"https://example.com/api/v3/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf"`                      // The budget itself
		Accounts     string `json:"accounts" example:"https://example.com/api/v3/accounts?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`          // Accounts for this budget
		Categories   string `json:"categories" example:"https://example.com/api/v3/categories?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`      // Categories for this budget
		Envelopes    string `json:"envelopes" example:"https://example.com/api/v3/envelopes?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`        // Envelopes for this budget
		Transactions string `json:"transactions" example:"https://example.com/api/v3/transactions?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`  // Transactions for this budget
		Month        string `json:"month" example:"https://example.com/api/v3/months?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf&month=YYYY-MM"` // This uses 'YYYY-MM' for clients to replace with the actual year and month.
	} `json:"links"`
}

// links sets all links for the Budget.
func (b *BudgetV3) links(c *gin.Context) {
	url := c.GetString(string(database.ContextURL))

	b.Links.Self = fmt.Sprintf("%s/v3/budgets/%s", url, b.ID)
	b.Links.Accounts = fmt.Sprintf("%s/v3/accounts?budget=%s", url, b.ID)
	b.Links.Categories = fmt.Sprintf("%s/v3/categories?budget=%s", url, b.ID)
	b.Links.Envelopes = fmt.Sprintf("%s/v3/envelopes?budget=%s", url, b.ID)
	b.Links.Transactions = fmt.Sprintf("%s/v3/transactions?budget=%s", url, b.ID)
	b.Links.Month = fmt.Sprintf("%s/v3/months?budget=%s&month=YYYY-MM", url, b.ID)
}

// getBudgetV3 returns a budget with all fields set.
func (co Controller) getBudgetV3(c *gin.Context, id uuid.UUID) (BudgetV3, httperrors.Error) {
	m, err := getResourceByID[models.Budget](c, co, id)
	if !err.Nil() {
		return BudgetV3{}, err
	}

	b := BudgetV3{
		Budget: m,
	}

	b.links(c)

	return b, httperrors.Error{}
}

type BudgetListResponseV3 struct {
	Data       []BudgetV3  `json:"data"`                                                          // List of budgets
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type BudgetCreateResponseV3 struct {
	Error *string            `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []BudgetResponseV3 `json:"data"`                                                          // List of created Budgets
}

func (b *BudgetCreateResponseV3) appendError(err httperrors.Error, status int) int {
	s := err.Error()
	b.Data = append(b.Data, BudgetResponseV3{Error: &s})

	// The final status code is the highest HTTP status code number
	if err.Status > status {
		status = err.Status
	}

	return status
}

type BudgetResponseV3 struct {
	Data  *BudgetV3 `json:"data"`                                                          // Data for the budget
	Error *string   `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type BudgetMonthResponseV3 struct {
	Data  *models.BudgetMonth `json:"data"`                                                          // Data for the budget's month
	Error *string             `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

// RegisterBudgetRoutesV3 registers the routes for Budgets with
// the RouterGroup that is passed.
func (co Controller) RegisterBudgetRoutesV3(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsBudgetListV3)
		r.GET("", co.GetBudgetsV3)
		r.POST("", co.CreateBudgetsV3)
	}

	// Budget with ID
	{
		r.OPTIONS("/:id", co.OptionsBudgetDetailV3)
		r.GET("/:id", co.GetBudgetV3)
		r.PATCH("/:id", co.UpdateBudgetV3)
		r.DELETE("/:id", co.DeleteBudgetV3)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Budgets
// @Success		204
// @Router			/v3/budgets [options]
func (co Controller) OptionsBudgetListV3(c *gin.Context) {
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
// @Router			/v3/budgets/{id} [options]
func (co Controller) OptionsBudgetDetailV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = getResourceByID[models.Budget](c, co, id)
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
// @Success		201		{object}	BudgetCreateResponseV3
// @Failure		400		{object}	BudgetCreateResponseV3
// @Failure		500		{object}	BudgetCreateResponseV3
// @Param			budget	body		models.BudgetCreate	true	"Budget"
// @Router			/v3/budgets [post]
func (co Controller) CreateBudgetsV3(c *gin.Context) {
	var budgets []models.BudgetCreate

	// Bind data and return error if not possible
	err := httputil.BindData(c, &budgets)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, BudgetCreateResponseV3{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := BudgetCreateResponseV3{}

	for _, create := range budgets {
		b := models.Budget{
			BudgetCreate: create,
		}

		dbErr := co.DB.Create(&b).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Budget](b, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		// Append the budget
		bObject, err := co.getBudgetV3(c, b.ID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}
		r.Data = append(r.Data, BudgetResponseV3{Data: &bObject})
	}

	c.JSON(status, r)
}

// @Summary		List budgets
// @Description	Returns a list of budgets
// @Tags			Budgets
// @Produce		json
// @Success		200	{object}	BudgetListResponseV3
// @Failure		500	{object}	BudgetListResponseV3
// @Router			/v3/budgets [get]
// @Param			name		query	string	false	"Filter by name"
// @Param			note		query	string	false	"Filter by note"
// @Param			currency	query	string	false	"Filter by currency"
// @Param			search		query	string	false	"Search for this text in name and note"
// @Param			offset		query	uint	false	"The offset of the first Budget returned. Defaults to 0."
// @Param			limit		query	int		false	"Maximum number of Budgets to return. Defaults to 50."
func (co Controller) GetBudgetsV3(c *gin.Context) {
	var filter BudgetQueryFilterV3

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we're filtering for
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	var budgets []models.Budget

	// Always sort by name
	q := co.DB.
		Order("name ASC").
		Where(&models.Budget{
			BudgetCreate: models.BudgetCreate{
				Name:     filter.Name,
				Note:     filter.Note,
				Currency: filter.Currency,
			},
		},
			queryFields...)

	q = stringFilters(co.DB, q, setFields, filter.Name, filter.Note, filter.Search)

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
		c.JSON(err.Status, BudgetListResponseV3{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, BudgetListResponseV3{
			Error: &e,
		})
		return
	}

	budgetResources := make([]BudgetV3, 0)
	for _, budget := range budgets {
		r, err := co.getBudgetV3(c, budget.ID)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, BudgetListResponseV3{
				Error: &s,
			})
			return
		}
		budgetResources = append(budgetResources, r)
	}

	c.JSON(http.StatusOK, BudgetListResponseV3{
		Data: budgetResources,
		Pagination: &Pagination{
			Count:  len(budgetResources),
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
// @Success		200	{object}	BudgetResponseV3
// @Failure		400	{object}	BudgetResponseV3
// @Failure		404	{object}	BudgetResponseV3
// @Failure		500	{object}	BudgetResponseV3
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/budgets/{id} [get]
func (co Controller) GetBudgetV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	m, err := getResourceByID[models.Budget](c, co, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	r, err := co.getBudgetV3(c, m.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, BudgetResponseV3{Data: &r})
}

// @Summary		Update budget
// @Description	Update an existing budget. Only values to be updated need to be specified.
// @Tags			Budgets
// @Accept			json
// @Produce		json
// @Success		200		{object}	BudgetResponseV3
// @Failure		400		{object}	BudgetResponseV3
// @Failure		404		{object}	BudgetResponseV3
// @Failure		500		{object}	BudgetResponseV3
// @Param			id		path		string				true	"ID formatted as string"
// @Param			budget	body		models.BudgetCreate	true	"Budget"
// @Router			/v3/budgets/{id} [patch]
func (co Controller) UpdateBudgetV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	budget, err := getResourceByID[models.Budget](c, co, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.BudgetCreate{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	var data models.Budget
	err = httputil.BindData(c, &data.BudgetCreate)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	err = query(c, co.DB.Model(&budget).Select("", updateFields...).Updates(data))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	r, err := co.getBudgetV3(c, budget.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, BudgetResponseV3{Data: &r})
}

// @Summary		Delete budget
// @Description	Deletes a budget
// @Tags			Budgets
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/budgets/{id} [delete]
func (co Controller) DeleteBudgetV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	budget, err := getResourceByID[models.Budget](c, co, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, co.DB.Delete(&budget))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
