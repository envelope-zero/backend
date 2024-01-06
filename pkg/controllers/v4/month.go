package v4

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type MonthResponse struct {
	Data  *Month  `json:"data"`  // Data for the month
	Error *string `json:"error"` // The error, if any occurred
}

type Month struct {
	ID         uuid.UUID           `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Budget
	Name       string              `json:"name" example:"Zero budget"`                        // The name of the Budget
	Month      types.Month         `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // The month
	Income     decimal.Decimal     `json:"income" example:"2317.34"`                          // The total income for the month (sum of all incoming transactions without an Envelope)
	Available  decimal.Decimal     `json:"available" example:"217.34"`                        // The amount available to budget
	Balance    decimal.Decimal     `json:"balance" example:"5231.37"`                         // The sum of all envelope balances
	Spent      decimal.Decimal     `json:"spent" example:"133.70"`                            // The amount of money spent in this month
	Allocation decimal.Decimal     `json:"allocation" example:"1200.50"`                      // The sum of all allocations for this month
	Categories []CategoryEnvelopes `json:"categories"`                                        // A list of envelope month calculations grouped by category
}

type CategoryEnvelopes struct {
	Category
	Envelopes  []EnvelopeMonth `json:"envelopes"`                // Slice of all envelopes
	Balance    decimal.Decimal `json:"balance" example:"-10.13"` // Sum of the balances of the envelopes
	Allocation decimal.Decimal `json:"allocation" example:"90"`  // Sum of allocations for the envelopes
	Spent      decimal.Decimal `json:"spent" example:"100.13"`   // Sum spent for all envelopes
}

// EnvelopeMonth contains data about an Envelope for a specific month.
type EnvelopeMonth struct {
	Envelope
	Spent      decimal.Decimal `json:"spent" example:"73.12"`      // The amount spent over the whole month
	Balance    decimal.Decimal `json:"balance" example:"12.32"`    // The balance at the end of the monht
	Allocation decimal.Decimal `json:"allocation" example:"85.44"` // The amount of money allocated
}

// RegisterMonthRoutes registers the routes for months with
// the RouterGroup that is passed.
func RegisterMonthRoutes(r *gin.RouterGroup) {
	{
		r.OPTIONS("", OptionsMonth)
		r.GET("", GetMonth)
		r.POST("", SetAllocations)
		r.DELETE("", DeleteAllocations)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs.
// @Tags			Months
// @Success		204
// @Router			/v4/months [options]
func OptionsMonth(c *gin.Context) {
	httputil.OptionsGetPostDelete(c)
}

// @Summary		Get data about a month
// @Description	Returns data about a specific month.
// @Tags			Months
// @Produce		json
// @Success		200		{object}	MonthResponse
// @Failure		400		{object}	MonthResponse
// @Failure		404		{object}	MonthResponse
// @Failure		500		{object}	MonthResponse
// @Param			budget	query		string	true	"ID formatted as string"
// @Param			month	query		string	true	"The month in YYYY-MM format"
// @Router			/v4/months [get]
func GetMonth(c *gin.Context) {
	qMonth, b, e := parseMonthQuery(c)
	if !e.Nil() {
		s := e.Error()
		c.JSON(e.Status, MonthResponse{
			Error: &s,
		})
		return
	}

	month := qMonth

	result := Month{
		ID:    b.ID,
		Name:  b.Name,
		Month: month,
	}

	// Add allocated sum to response
	allocated, err := b.Allocated(models.DB, result.Month)
	if err != nil {
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, MonthResponse{
			Error: &s,
		})
		return
	}
	result.Allocation = allocated

	// Add income to response
	income, err := b.Income(models.DB, result.Month)
	if err != nil {
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, MonthResponse{
			Error: &s,
		})
		return
	}
	result.Income = income

	// Get all categories for the budget
	var categories []models.Category
	err = models.DB.
		Where(&models.Category{BudgetID: b.ID}).
		Order("name ASC").
		Find(&categories).
		Error

	if err != nil {
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, MonthResponse{
			Error: &s,
		})
		return
	}

	result.Categories = make([]CategoryEnvelopes, 0)
	result.Balance = decimal.Zero

	// Get envelopes for all categories
	for _, category := range categories {
		var categoryEnvelopes CategoryEnvelopes

		// Set the basic category values
		categoryResource, e := newCategory(c, models.DB, category)
		if !e.Nil() {
			s := e.Error()
			c.JSON(e.Status, MonthResponse{
				Error: &s,
			})
			return
		}

		categoryEnvelopes.Category = categoryResource
		categoryEnvelopes.Envelopes = make([]EnvelopeMonth, 0)

		var envelopes []models.Envelope

		err = models.DB.
			Where(&models.Envelope{
				CategoryID: category.ID,
			}).
			Order("name asc").
			Find(&envelopes).
			Error

		if err != nil {
			e := httperrors.Parse(c, err)
			s := e.Error()
			c.JSON(e.Status, MonthResponse{
				Error: &s,
			})
			return
		}

		for _, envelope := range envelopes {
			envelopeMonth, err := envelopeMonth(c, models.DB, envelope, result.Month)
			if err != nil {
				e := httperrors.Parse(c, err)
				s := e.Error()
				c.JSON(e.Status, MonthResponse{
					Error: &s,
				})
				return
			}

			// Update the month's summarized data
			result.Balance = result.Balance.Add(envelopeMonth.Balance)
			result.Spent = result.Spent.Add(envelopeMonth.Spent)

			// Update the category's summarized data
			categoryEnvelopes.Balance = categoryEnvelopes.Balance.Add(envelopeMonth.Balance)
			categoryEnvelopes.Spent = categoryEnvelopes.Spent.Add(envelopeMonth.Spent)
			categoryEnvelopes.Allocation = categoryEnvelopes.Allocation.Add(envelopeMonth.Allocation)
			categoryEnvelopes.Envelopes = append(categoryEnvelopes.Envelopes, envelopeMonth)
		}

		result.Categories = append(result.Categories, categoryEnvelopes)
	}

	// Available amount is the sum of balances of all on-budget accounts, then subtract the sum of all envelope balances
	result.Available = result.Balance.Neg()

	// Get all on budget accounts for the budget
	var accounts []models.Account
	err = models.DB.Where(&models.Account{BudgetID: b.ID, OnBudget: true}).Find(&accounts).Error
	if err != nil {
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, MonthResponse{
			Error: &s,
		})
		return
	}

	// Add all on-balance accounts to the available sum
	for _, a := range accounts {
		_, available, err := a.GetBalanceMonth(models.DB, month)
		if err != nil {
			e := httperrors.Parse(c, err)
			s := e.Error()
			c.JSON(e.Status, MonthResponse{
				Error: &s,
			})
			return
		}
		result.Available = result.Available.Add(available)
	}

	c.JSON(http.StatusOK, MonthResponse{Data: &result})
}

// @Summary		Delete allocations for a month
// @Description	Deletes all allocation for the specified month
// @Tags			Months
// @Success		204
// @Failure		400		{object}	httperrors.HTTPError
// @Failure		404		{object}	httperrors.HTTPError
// @Failure		500		{object}	httperrors.HTTPError
// @Param			budget	query		string	true	"ID formatted as string"
// @Param			month	query		string	true	"The month in YYYY-MM format"
// @Router			/v4/months [delete]
func DeleteAllocations(c *gin.Context) {
	month, budget, err := parseMonthQuery(c)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	var monthConfigs []models.MonthConfig

	err = query(c, models.DB.
		Joins("JOIN envelopes ON envelopes.id = month_configs.envelope_id").
		Joins("JOIN categories ON categories.id = envelopes.category_id").
		Joins("JOIN budgets on budgets.id = categories.budget_id").
		Where(models.MonthConfig{Month: month}).
		Where("budgets.id = ?", budget.ID).
		Where("month_configs.allocation > 0").
		Find(&monthConfigs))

	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	for _, monthConfig := range monthConfigs {
		monthConfig.Allocation = decimal.Zero
		err = query(c, models.DB.Updates(&monthConfig))
		if !err.Nil() {
			c.JSON(err.Status, httperrors.HTTPError{
				Error: err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// @Summary		Set allocations for a month
// @Description	Sets allocations for a month for all envelopes that do not have an allocation yet
// @Tags			Months
// @Success		204
// @Failure		400		{object}	httperrors.HTTPError
// @Failure		404		{object}	httperrors.HTTPError
// @Failure		500		{object}	httperrors.HTTPError
// @Param			budget	query		string					true	"ID formatted as string"
// @Param			month	query		string					true	"The month in YYYY-MM format"
// @Param			mode	body		BudgetAllocationMode	true	"Budget"
// @Router			/v4/months [post]
func SetAllocations(c *gin.Context) {
	month, _, err := parseMonthQuery(c)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	// Get the mode to set new allocations in
	var data BudgetAllocationMode
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	if data.Mode != AllocateLastMonthBudget && data.Mode != AllocateLastMonthSpend {
		httperrors.New(c, http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, httperrors.HTTPError{
			Error: fmt.Sprintf("The mode must be %s or %s", AllocateLastMonthBudget, AllocateLastMonthSpend),
		})
		return
	}

	pastMonth := month.AddDate(0, -1)
	queryCurrentMonth := models.DB.Select("*").Table("month_configs").Where("month_configs.envelope_id = envelopes.id AND month_configs.month = ? AND month_configs.allocation != 0", month)

	// Get all envelopes that do not have an allocation for the target month
	// but for the month before
	var envelopesAmount []struct {
		EnvelopeID uuid.UUID       `gorm:"column:id"`
		Amount     decimal.Decimal `gorm:"column:allocation"`
	}

	// Get all envelope IDs and allocation amounts where there is no allocation
	// for the request month, but one for the last month
	err = query(c, models.DB.
		Joins("JOIN month_configs ON month_configs.envelope_id = envelopes.id AND envelopes.archived IS FALSE AND month_configs.month = ? AND NOT EXISTS(?)", pastMonth, queryCurrentMonth).
		Select("envelopes.id, month_configs.allocation").
		Table("envelopes").
		Find(&envelopesAmount))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	// Create all new allocations
	for _, allocation := range envelopesAmount {
		// If the mode is the spend of last month, calculate and set it
		amount := allocation.Amount
		if data.Mode == AllocateLastMonthSpend {
			amount = models.Envelope{DefaultModel: models.DefaultModel{ID: allocation.EnvelopeID}}.Spent(models.DB, pastMonth).Neg()
		}

		// Find and update the correct MonthConfig.
		// If it does not exist, create it
		err = query(c, models.DB.Where(models.MonthConfig{
			Month:      month,
			EnvelopeID: allocation.EnvelopeID,
		}).Assign(models.MonthConfig{
			Allocation: amount,
		}).FirstOrCreate(&models.MonthConfig{}))
		if !err.Nil() {
			c.JSON(err.Status, httperrors.HTTPError{
				Error: err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// envelopeMonth calculates the month specific values for an envelope and returns an EnvelopeMonth with them
func envelopeMonth(c *gin.Context, db *gorm.DB, e models.Envelope, month types.Month) (EnvelopeMonth, error) {
	spent := e.Spent(db, month)

	envelopeMonth := EnvelopeMonth{
		Envelope:   newEnvelope(c, e),
		Spent:      spent,
		Balance:    decimal.NewFromFloat(0),
		Allocation: decimal.NewFromFloat(0),
	}

	var monthConfig models.MonthConfig
	err := db.First(&monthConfig, &models.MonthConfig{
		EnvelopeID: e.ID,
		Month:      month,
	}).Error

	// If an unexpected error occurs, return
	if err != nil && err != gorm.ErrRecordNotFound {
		return EnvelopeMonth{}, err
	}

	envelopeMonth.Balance, err = e.Balance(db, month)
	if err != nil {
		return EnvelopeMonth{}, err
	}

	envelopeMonth.Allocation = monthConfig.Allocation
	return envelopeMonth, nil
}

// parseMonthQuery takes in the context and parses the request
//
// It verifies that the requested budget exists and parses the ID to return
// the budget resource itself.
func parseMonthQuery(c *gin.Context) (types.Month, models.Budget, httperrors.Error) {
	var query struct {
		QueryMonth
		BudgetID string `form:"budget" example:"81b0c9ce-6fd3-4e1e-becc-106055898a2a"`
	}

	if err := c.BindQuery(&query); err != nil {
		return types.Month{}, models.Budget{}, httperrors.Parse(c, err)
	}

	if query.Month.IsZero() {
		return types.Month{}, models.Budget{}, httperrors.Error{
			Status: http.StatusBadRequest,
			Err:    httperrors.ErrMonthNotSetInQuery,
		}
	}

	budgetID, err := uuid.Parse(query.BudgetID)
	if err != nil {
		return types.Month{}, models.Budget{}, httperrors.Parse(c, err)
	}

	budget, e := getModelByID[models.Budget](c, budgetID)
	if !e.Nil() {
		return types.Month{}, models.Budget{}, e
	}

	return types.MonthOf(query.Month), budget, httperrors.Error{}
}
