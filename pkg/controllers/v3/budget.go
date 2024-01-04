package v3

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

type BudgetQueryFilter struct {
	Name     string `form:"name" filterField:"false"`   // By name
	Note     string `form:"note" filterField:"false"`   // By note
	Currency string `form:"currency"`                   // By currency
	Search   string `form:"search" filterField:"false"` // By string in name or note
	Offset   uint   `form:"offset" filterField:"false"` // The offset of the first Budget returned. Defaults to 0.
	Limit    int    `form:"limit" filterField:"false"`  // Maximum number of Budgets to return. Defaults to 50.
}

// Budget is the API v3 representation of a Budget.
type Budget struct {
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
func (b *Budget) links(c *gin.Context) {
	url := c.GetString(string(models.DBContextURL))

	b.Links.Self = fmt.Sprintf("%s/v3/budgets/%s", url, b.ID)
	b.Links.Accounts = fmt.Sprintf("%s/v3/accounts?budget=%s", url, b.ID)
	b.Links.Categories = fmt.Sprintf("%s/v3/categories?budget=%s", url, b.ID)
	b.Links.Envelopes = fmt.Sprintf("%s/v3/envelopes?budget=%s", url, b.ID)
	b.Links.Transactions = fmt.Sprintf("%s/v3/transactions?budget=%s", url, b.ID)
	b.Links.Month = fmt.Sprintf("%s/v3/months?budget=%s&month=YYYY-MM", url, b.ID)
}

// getBudget returns a budget with all fields set.
func getBudget(c *gin.Context, id uuid.UUID) (Budget, httperrors.Error) {
	m, err := getResourceByID[models.Budget](c, id)
	if !err.Nil() {
		return Budget{}, err
	}

	b := Budget{
		Budget: m,
	}

	b.links(c)

	return b, httperrors.Error{}
}

type BudgetListResponse struct {
	Data       []Budget    `json:"data"`                                                          // List of budgets
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type BudgetCreateResponse struct {
	Error *string          `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []BudgetResponse `json:"data"`                                                          // List of created Budgets
}

func (b *BudgetCreateResponse) appendError(err httperrors.Error, status int) int {
	s := err.Error()
	b.Data = append(b.Data, BudgetResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	if err.Status > status {
		status = err.Status
	}

	return status
}

type BudgetResponse struct {
	Data  *Budget `json:"data"`                                                          // Data for the budget
	Error *string `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

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
// @Router			/v3/budgets [options]
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
// @Router			/v3/budgets/{id} [options]
func OptionsBudgetDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = getResourceByID[models.Budget](c, id)
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
// @Param			budget	body		models.BudgetCreate	true	"Budget"
// @Router			/v3/budgets [post]
func CreateBudgets(c *gin.Context) {
	var budgets []models.BudgetCreate

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

	for _, create := range budgets {
		b := models.Budget{
			BudgetCreate: create,
		}

		dbErr := models.DB.Create(&b).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Budget](b, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		// Append the budget
		bObject, err := getBudget(c, b.ID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}
		r.Data = append(r.Data, BudgetResponse{Data: &bObject})
	}

	c.JSON(status, r)
}

// @Summary		List budgets
// @Description	Returns a list of budgets
// @Tags			Budgets
// @Produce		json
// @Success		200	{object}	BudgetListResponse
// @Failure		500	{object}	BudgetListResponse
// @Router			/v3/budgets [get]
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
		Where(&models.Budget{
			BudgetCreate: models.BudgetCreate{
				Name:     filter.Name,
				Note:     filter.Note,
				Currency: filter.Currency,
			},
		},
			queryFields...)

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

	budgetResources := make([]Budget, 0)
	for _, budget := range budgets {
		r, err := getBudget(c, budget.ID)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, BudgetListResponse{
				Error: &s,
			})
			return
		}
		budgetResources = append(budgetResources, r)
	}

	c.JSON(http.StatusOK, BudgetListResponse{
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
// @Success		200	{object}	BudgetResponse
// @Failure		400	{object}	BudgetResponse
// @Failure		404	{object}	BudgetResponse
// @Failure		500	{object}	BudgetResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/budgets/{id} [get]
func GetBudget(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	m, err := getResourceByID[models.Budget](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	r, err := getBudget(c, m.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, BudgetResponse{Data: &r})
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
// @Param			id		path		string				true	"ID formatted as string"
// @Param			budget	body		models.BudgetCreate	true	"Budget"
// @Router			/v3/budgets/{id} [patch]
func UpdateBudget(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	budget, err := getResourceByID[models.Budget](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.BudgetCreate{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	var data models.Budget
	err = httputil.BindData(c, &data.BudgetCreate)
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

	r, err := getBudget(c, budget.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, BudgetResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, BudgetResponse{Data: &r})
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
func DeleteBudget(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	budget, err := getResourceByID[models.Budget](c, id)
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
