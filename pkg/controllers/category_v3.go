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

// CategoryCreateV3 represents all user configurable parameters
type CategoryCreateV3 struct {
	Name     string    `json:"name" gorm:"uniqueIndex:category_budget_name" example:"Saving" default:""`                        // Name of the category
	BudgetID uuid.UUID `json:"budgetId" gorm:"uniqueIndex:category_budget_name" example:"52d967d3-33f4-4b04-9ba7-772e5ab9d0ce"` // ID of the budget the category belongs to
	Note     string    `json:"note" example:"All envelopes for long-term saving" default:""`                                    // Notes about the category
	Archived bool      `json:"archived" example:"true" default:"false"`                                                         // Is the category hidden?
}

// ToCreate transforms the API representation into the model representation
func (c CategoryCreateV3) ToCreate() models.CategoryCreate {
	return models.CategoryCreate{
		Name:     c.Name,
		BudgetID: c.BudgetID,
		Note:     c.Note,
		Hidden:   c.Archived,
	}
}

type CategoryV3 struct {
	models.Category
	Envelopes []EnvelopeV3 `json:"envelopes"`        // Envelopes for the category
	Hidden    bool         `json:"hidden,omitempty"` // Remove the hidden field

	Links struct {
		Self      string `json:"self" example:"https://example.com/api/v3/categories/3b1ea324-d438-4419-882a-2fc91d71772f"`              // The category itself
		Envelopes string `json:"envelopes" example:"https://example.com/api/v3/envelopes?category=3b1ea324-d438-4419-882a-2fc91d71772f"` // Envelopes for this category
	} `json:"links"`
}

func (c *CategoryV3) links(context *gin.Context) {
	url := context.GetString(string(database.ContextURL))

	c.Links.Self = fmt.Sprintf("%s/v3/categories/%s", url, c.ID)
	c.Links.Envelopes = fmt.Sprintf("%s/v3/envelopes?category=%s", url, c.ID)
}

func (co Controller) getCategoryV3(c *gin.Context, id uuid.UUID) (CategoryV3, httperrors.Error) {
	m, e := getResourceByID[models.Category](c, co, id)
	if !e.Nil() {
		return CategoryV3{}, e
	}

	cat := CategoryV3{
		Category: m,
	}

	eModels, err := m.Envelopes(co.DB)
	if err != nil {
		return CategoryV3{}, httperrors.Parse(c, err)
	}

	envelopes := make([]EnvelopeV3, 0)
	for _, e := range eModels {
		o, e := co.getEnvelopeV3(c, e.ID)
		if !e.Nil() {
			return CategoryV3{}, e
		}
		envelopes = append(envelopes, o)
	}

	cat.Envelopes = envelopes
	cat.links(c)

	return cat, httperrors.Error{}
}

type CategoryListResponseV3 struct {
	Data       []CategoryV3 `json:"data"`                                                          // List of Categories
	Error      *string      `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination  `json:"pagination"`                                                    // Pagination information
}

type CategoryCreateResponseV3 struct {
	Data  []CategoryResponseV3 `json:"data"`                                                          // List of the created Categories or their respective error
	Error *string              `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

func (c *CategoryCreateResponseV3) appendError(err httperrors.Error, status int) int {
	s := err.Error()
	c.Data = append(c.Data, CategoryResponseV3{Error: &s})

	// The final status code is the highest HTTP status code number
	if err.Status > status {
		status = err.Status
	}

	return status
}

type CategoryResponseV3 struct {
	Data  *CategoryV3 `json:"data"`                                                          // Data for the Category
	Error *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type CategoryQueryFilterV3 struct {
	Name     string `form:"name" filterField:"false"`     // By name
	BudgetID string `form:"budget"`                       // By ID of the Budget
	Note     string `form:"note" filterField:"false"`     // By note
	Archived bool   `form:"archived" filterField:"false"` // Is the Category archived?
	Search   string `form:"search" filterField:"false"`   // By string in name or note
	Offset   uint   `form:"offset" filterField:"false"`   // The offset of the first Category returned. Defaults to 0.
	Limit    int    `form:"limit" filterField:"false"`    // Maximum number of Categories to return. Defaults to 50.
}

func (f CategoryQueryFilterV3) ToCreate() (models.CategoryCreate, httperrors.Error) {
	budgetID, err := httputil.UUIDFromString(f.BudgetID)
	if !err.Nil() {
		return models.CategoryCreate{}, httperrors.Error{}
	}

	return models.CategoryCreate{
		BudgetID: budgetID,
		Hidden:   f.Archived,
	}, httperrors.Error{}
}

// RegisterCategoryRoutesV3 registers the routes for categories with
// the RouterGroup that is passed.
func (co Controller) RegisterCategoryRoutesV3(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsCategoryListV3)
		r.GET("", co.GetCategoriesV3)
		r.POST("", co.CreateCategoriesV3)
	}

	// Category with ID
	{
		r.OPTIONS("/:id", co.OptionsCategoryDetailV3)
		r.GET("/:id", co.GetCategoryV3)
		r.PATCH("/:id", co.UpdateCategoryV3)
		r.DELETE("/:id", co.DeleteCategoryV3)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Categories
// @Success		204
// @Router			/v3/categories [options]
func (co Controller) OptionsCategoryListV3(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Categories
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/categories/{id} [options]
func (co Controller) OptionsCategoryDetailV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = co.getCategoryV3(c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// @Summary		Create category
// @Description	Creates a new category
// @Tags			Categories
// @Produce		json
// @Success		201			{object}	CategoryCreateResponseV3
// @Failure		400			{object}	CategoryCreateResponseV3
// @Failure		404			{object}	CategoryCreateResponseV3
// @Failure		500			{object}	CategoryCreateResponseV3
// @Param			categories	body		[]CategoryCreateV3	true	"Categories"
// @Router			/v3/categories [post]
func (co Controller) CreateCategoriesV3(c *gin.Context) {
	var categories []CategoryCreateV3

	// Bind data and return error if not possible
	err := httputil.BindData(c, &categories)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, CategoryCreateResponseV3{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := CategoryCreateResponseV3{}

	for _, create := range categories {
		category := models.Category{
			CategoryCreate: create.ToCreate(),
		}

		// Verify that the budget exists. If not, append the error
		// and move to the next one.
		_, err := getResourceByID[models.Budget](c, co, create.BudgetID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		dbErr := co.DB.Create(&category).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Category](category, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		eObject, err := co.getCategoryV3(c, category.ID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}
		r.Data = append(r.Data, CategoryResponseV3{Data: &eObject})
	}

	c.JSON(status, r)
}

// @Summary		Get categories
// @Description	Returns a list of categories
// @Tags			Categories
// @Produce		json
// @Success		200	{object}	CategoryListResponseV3
// @Failure		400	{object}	CategoryListResponseV3
// @Failure		500	{object}	CategoryListResponseV3
// @Router			/v3/categories [get]
// @Param			name	query	string	false	"Filter by name"
// @Param			note	query	string	false	"Filter by note"
// @Param			budget	query	string	false	"Filter by budget ID"
// @Param			hidden	query	bool	false	"Is the category hidden?"
// @Param			search	query	string	false	"Search for this text in name and note"
// @Param			offset	query	uint	false	"The offset of the first Category returned. Defaults to 0."
// @Param			limit	query	int		false	"Maximum number of Categories to return. Defaults to 50."
func (co Controller) GetCategoriesV3(c *gin.Context) {
	var filter CategoryQueryFilterV3

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we are filtering for
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// If the archived parameter is set, add "Hidden" to the query fields
	// This is done since in v3, we're using the name "Archived", but the
	// field is not yet updated in the database, which will happen later
	if slices.Contains(setFields, "Archived") {
		queryFields = append(queryFields, "Hidden")
	}

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate()
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryListResponseV3{
			Error: &s,
		})
		return
	}

	q := co.DB.
		Order("name ASC").
		Where(&models.Category{
			CategoryCreate: create,
		}, queryFields...)

	q = stringFilters(co.DB, q, setFields, filter.Name, filter.Note, filter.Search)

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to 50 Accounts and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = filter.Limit
	}
	q = q.Limit(limit)

	var categories []models.Category
	err = query(c, q.Find(&categories))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryListResponseV3{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, CategoryListResponseV3{
			Error: &e,
		})
		return
	}

	r := make([]CategoryV3, 0)
	for _, category := range categories {
		o, err := co.getCategoryV3(c, category.ID)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, CategoryListResponseV3{
				Error: &s,
			})
			return
		}
		r = append(r, o)
	}

	c.JSON(http.StatusOK, CategoryListResponseV3{
		Data: r,
		Pagination: &Pagination{
			Count:  len(r),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Get category
// @Description	Returns a specific category
// @Tags			Categories
// @Produce		json
// @Success		200	{object}	CategoryResponseV3
// @Failure		400	{object}	CategoryResponseV3
// @Failure		404	{object}	CategoryResponseV3
// @Failure		500	{object}	CategoryResponseV3
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/categories/{id} [get]
func (co Controller) GetCategoryV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	r, err := co.getCategoryV3(c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, CategoryResponseV3{Data: &r})
}

// @Summary		Update category
// @Description	Update an existing category. Only values to be updated need to be specified.
// @Tags			Categories
// @Accept			json
// @Produce		json
// @Success		200			{object}	CategoryResponseV3
// @Failure		400			{object}	CategoryResponseV3
// @Failure		404			{object}	CategoryResponseV3
// @Failure		500			{object}	CategoryResponseV3
// @Param			id			path		string				true	"ID formatted as string"
// @Param			category	body		CategoryCreateV3	true	"Category"
// @Router			/v3/categories/{id} [patch]
func (co Controller) UpdateCategoryV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	category, err := getResourceByID[models.Category](c, co, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, CategoryCreateV3{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	var data CategoryCreateV3
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	// Transform the API representation to the model representation
	cat := models.Category{
		CategoryCreate: data.ToCreate(),
	}

	// If the archived parameter is set, add "Hidden" to the update fields
	// This is done since in v3, we're using the name "Archived", but the
	// field is not yet updated in the database, which will happen later
	if slices.Contains(updateFields, "Archived") {
		updateFields = append(updateFields, "Hidden")
	}

	err = query(c, co.DB.Model(&category).Select("", updateFields...).Updates(cat))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	r, err := co.getCategoryV3(c, category.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponseV3{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, CategoryResponseV3{Data: &r})
}

// @Summary		Delete category
// @Description	Deletes a category
// @Tags			Categories
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/categories/{id} [delete]
func (co Controller) DeleteCategoryV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	category, err := getResourceByID[models.Category](c, co, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, co.DB.Delete(&category))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
