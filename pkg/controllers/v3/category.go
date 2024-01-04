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

// CategoryCreate represents all user configurable parameters
type CategoryCreate struct {
	Name     string    `json:"name" gorm:"uniqueIndex:category_budget_name" example:"Saving" default:""`                        // Name of the category
	BudgetID uuid.UUID `json:"budgetId" gorm:"uniqueIndex:category_budget_name" example:"52d967d3-33f4-4b04-9ba7-772e5ab9d0ce"` // ID of the budget the category belongs to
	Note     string    `json:"note" example:"All envelopes for long-term saving" default:""`                                    // Notes about the category
	Archived bool      `json:"archived" example:"true" default:"false"`                                                         // Is the category archived?
}

// ToCreate transforms the API representation into the model representation
func (c CategoryCreate) ToCreate() models.CategoryCreate {
	return models.CategoryCreate{
		Name:     c.Name,
		BudgetID: c.BudgetID,
		Note:     c.Note,
		Archived: c.Archived,
	}
}

type Category struct {
	models.Category
	Envelopes []Envelope `json:"envelopes"` // Envelopes for the category

	Links struct {
		Self      string `json:"self" example:"https://example.com/api/v3/categories/3b1ea324-d438-4419-882a-2fc91d71772f"`              // The category itself
		Envelopes string `json:"envelopes" example:"https://example.com/api/v3/envelopes?category=3b1ea324-d438-4419-882a-2fc91d71772f"` // Envelopes for this category
	} `json:"links"`
}

func (c *Category) links(context *gin.Context) {
	url := context.GetString(string(models.DBContextURL))

	c.Links.Self = fmt.Sprintf("%s/v3/categories/%s", url, c.ID)
	c.Links.Envelopes = fmt.Sprintf("%s/v3/envelopes?category=%s", url, c.ID)
}

func getCategory(c *gin.Context, id uuid.UUID) (Category, httperrors.Error) {
	m, e := getResourceByID[models.Category](c, id)
	if !e.Nil() {
		return Category{}, e
	}

	cat := Category{
		Category: m,
	}

	eModels, err := m.Envelopes(models.DB)
	if err != nil {
		return Category{}, httperrors.Parse(c, err)
	}

	envelopes := make([]Envelope, 0)
	for _, e := range eModels {
		o, e := getEnvelope(c, e.ID)
		if !e.Nil() {
			return Category{}, e
		}
		envelopes = append(envelopes, o)
	}

	cat.Envelopes = envelopes
	cat.links(c)

	return cat, httperrors.Error{}
}

type CategoryListResponse struct {
	Data       []Category  `json:"data"`                                                          // List of Categories
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type CategoryCreateResponse struct {
	Data  []CategoryResponse `json:"data"`                                                          // List of the created Categories or their respective error
	Error *string            `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

func (c *CategoryCreateResponse) appendError(err httperrors.Error, status int) int {
	s := err.Error()
	c.Data = append(c.Data, CategoryResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	if err.Status > status {
		status = err.Status
	}

	return status
}

type CategoryResponse struct {
	Data  *Category `json:"data"`                                                          // Data for the Category
	Error *string   `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type CategoryQueryFilter struct {
	Name     string `form:"name" filterField:"false"`   // By name
	BudgetID string `form:"budget"`                     // By ID of the Budget
	Note     string `form:"note" filterField:"false"`   // By note
	Archived bool   `form:"archived"`                   // Is the Category archived?
	Search   string `form:"search" filterField:"false"` // By string in name or note
	Offset   uint   `form:"offset" filterField:"false"` // The offset of the first Category returned. Defaults to 0.
	Limit    int    `form:"limit" filterField:"false"`  // Maximum number of Categories to return. Defaults to 50.
}

func (f CategoryQueryFilter) ToCreate() (models.CategoryCreate, httperrors.Error) {
	budgetID, err := httputil.UUIDFromString(f.BudgetID)
	if !err.Nil() {
		return models.CategoryCreate{}, httperrors.Error{}
	}

	return models.CategoryCreate{
		BudgetID: budgetID,
		Archived: f.Archived,
	}, httperrors.Error{}
}

// RegisterCategoryRoutes registers the routes for categories with
// the RouterGroup that is passed.
func RegisterCategoryRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsCategoryList)
		r.GET("", GetCategories)
		r.POST("", CreateCategories)
	}

	// Category with ID
	{
		r.OPTIONS("/:id", OptionsCategoryDetail)
		r.GET("/:id", GetCategory)
		r.PATCH("/:id", UpdateCategory)
		r.DELETE("/:id", DeleteCategory)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Categories
// @Success		204
// @Router			/v3/categories [options]
func OptionsCategoryList(c *gin.Context) {
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
func OptionsCategoryDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = getCategory(c, id)
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
// @Success		201			{object}	CategoryCreateResponse
// @Failure		400			{object}	CategoryCreateResponse
// @Failure		404			{object}	CategoryCreateResponse
// @Failure		500			{object}	CategoryCreateResponse
// @Param			categories	body		[]CategoryCreate	true	"Categories"
// @Router			/v3/categories [post]
func CreateCategories(c *gin.Context) {
	var categories []CategoryCreate

	// Bind data and return error if not possible
	err := httputil.BindData(c, &categories)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, CategoryCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := CategoryCreateResponse{}

	for _, create := range categories {
		category := models.Category{
			CategoryCreate: create.ToCreate(),
		}

		// Verify that the budget exists. If not, append the error
		// and move to the next one.
		_, err := getResourceByID[models.Budget](c, create.BudgetID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		dbErr := models.DB.Create(&category).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Category](category, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		eObject, err := getCategory(c, category.ID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}
		r.Data = append(r.Data, CategoryResponse{Data: &eObject})
	}

	c.JSON(status, r)
}

// @Summary		Get categories
// @Description	Returns a list of categories
// @Tags			Categories
// @Produce		json
// @Success		200	{object}	CategoryListResponse
// @Failure		400	{object}	CategoryListResponse
// @Failure		500	{object}	CategoryListResponse
// @Router			/v3/categories [get]
// @Param			name		query	string	false	"Filter by name"
// @Param			note		query	string	false	"Filter by note"
// @Param			budget		query	string	false	"Filter by budget ID"
// @Param			archived	query	bool	false	"Is the category archived?"
// @Param			search		query	string	false	"Search for this text in name and note"
// @Param			offset		query	uint	false	"The offset of the first Category returned. Defaults to 0."
// @Param			limit		query	int		false	"Maximum number of Categories to return. Defaults to 50."
func GetCategories(c *gin.Context) {
	var filter CategoryQueryFilter

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we are filtering for
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate()
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryListResponse{
			Error: &s,
		})
		return
	}

	q := models.DB.
		Order("name ASC").
		Where(&models.Category{
			CategoryCreate: create,
		}, queryFields...)

	q = stringFilters(models.DB, q, setFields, filter.Name, filter.Note, filter.Search)

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
		c.JSON(err.Status, CategoryListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, CategoryListResponse{
			Error: &e,
		})
		return
	}

	r := make([]Category, 0)
	for _, category := range categories {
		o, err := getCategory(c, category.ID)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, CategoryListResponse{
				Error: &s,
			})
			return
		}
		r = append(r, o)
	}

	c.JSON(http.StatusOK, CategoryListResponse{
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
// @Success		200	{object}	CategoryResponse
// @Failure		400	{object}	CategoryResponse
// @Failure		404	{object}	CategoryResponse
// @Failure		500	{object}	CategoryResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/categories/{id} [get]
func GetCategory(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	r, err := getCategory(c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, CategoryResponse{Data: &r})
}

// @Summary		Update category
// @Description	Update an existing category. Only values to be updated need to be specified.
// @Tags			Categories
// @Accept			json
// @Produce		json
// @Success		200			{object}	CategoryResponse
// @Failure		400			{object}	CategoryResponse
// @Failure		404			{object}	CategoryResponse
// @Failure		500			{object}	CategoryResponse
// @Param			id			path		string			true	"ID formatted as string"
// @Param			category	body		CategoryCreate	true	"Category"
// @Router			/v3/categories/{id} [patch]
func UpdateCategory(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	category, err := getResourceByID[models.Category](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, CategoryCreate{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	var data CategoryCreate
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	// Transform the API representation to the model representation
	cat := models.Category{
		CategoryCreate: data.ToCreate(),
	}

	err = query(c, models.DB.Model(&category).Select("", updateFields...).Updates(cat))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	r, err := getCategory(c, category.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, CategoryResponse{Data: &r})
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
func DeleteCategory(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	category, err := getResourceByID[models.Category](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, models.DB.Delete(&category))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
