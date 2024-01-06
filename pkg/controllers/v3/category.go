package v3

import (
	"net/http"

	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

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

	_, err = getModelByID[models.Category](c, id)
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
// @Param			categories	body		[]CategoryEditable	true	"Categories"
// @Router			/v3/categories [post]
func CreateCategories(c *gin.Context) {
	var editables []CategoryEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &editables)
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

	for _, editable := range editables {
		category := editable.model()

		// Verify that the budget exists. If not, append the error
		// and move to the next one.
		_, err := getModelByID[models.Budget](c, editable.BudgetID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		dbErr := models.DB.Create(&category).Error
		if dbErr != nil {
			err := httperrors.Parse(c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		data, err := newCategory(c, models.DB, category)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}
		r.Data = append(r.Data, CategoryResponse{Data: &data})
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
	filterModel, err := filter.model()
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryListResponse{
			Error: &s,
		})
		return
	}

	q := models.DB.
		Order("name ASC").
		Where(&filterModel, queryFields...)

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

	data := make([]Category, 0)
	for _, category := range categories {
		apiResource, err := newCategory(c, models.DB, category)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, CategoryListResponse{
				Error: &s,
			})
			return
		}
		data = append(data, apiResource)
	}

	c.JSON(http.StatusOK, CategoryListResponse{
		Data: data,
		Pagination: &Pagination{
			Count:  len(data),
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

	category, err := getModelByID[models.Category](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	data, err := newCategory(c, models.DB, category)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, CategoryResponse{Data: &data})
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
// @Param			id			path		string				true	"ID formatted as string"
// @Param			category	body		CategoryEditable	true	"Category"
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

	category, err := getModelByID[models.Category](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, CategoryEditable{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	var data CategoryEditable
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	err = query(c, models.DB.Model(&category).Select("", updateFields...).Updates(data.model()))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, CategoryResponse{
			Error: &s,
		})
		return
	}

	r, err := newCategory(c, models.DB, category)
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

	category, err := getModelByID[models.Category](c, id)
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
