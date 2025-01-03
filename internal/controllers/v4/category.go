package v4

import (
	"net/http"

	"github.com/envelope-zero/backend/v5/internal/httputil"
	"github.com/envelope-zero/backend/v5/internal/models"
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
// @Router			/v4/categories [options]
func OptionsCategoryList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Categories
// @Success		204
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/categories/{id} [options]
func OptionsCategoryDetail(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.First(&models.Category{}, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
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
// @Router			/v4/categories [post]
func CreateCategories(c *gin.Context) {
	var editables []CategoryEditable

	// Bind data and return error if not possible
	err := httputil.BindData(c, &editables)
	if err != nil {
		e := err.Error()
		c.JSON(status(err), CategoryCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := CategoryCreateResponse{}

	for _, editable := range editables {
		category := editable.model()

		err = models.DB.Create(&category).Error
		if err != nil {
			status = r.appendError(err, status)
			continue
		}

		data, err := newCategory(c, models.DB, category)
		if err != nil {
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
// @Router			/v4/categories [get]
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
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryListResponse{
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
	err = q.Find(&categories).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = q.Limit(-1).Offset(-1).Count(&count).Error
	if err != nil {
		e := err.Error()
		c.JSON(status(err), CategoryListResponse{
			Error: &e,
		})
		return
	}

	data := make([]Category, 0)
	for _, category := range categories {
		apiResource, err := newCategory(c, models.DB, category)
		if err != nil {
			s := err.Error()
			c.JSON(status(err), CategoryListResponse{
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
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/categories/{id} [get]
func GetCategory(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
			Error: &s,
		})
		return
	}

	var category models.Category
	err = models.DB.First(&category, uri.ID).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
			Error: &s,
		})
		return
	}

	data, err := newCategory(c, models.DB, category)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
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
// @Param			id			path		URIID				true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Param			category	body		CategoryEditable	true	"Category"
// @Router			/v4/categories/{id} [patch]
func UpdateCategory(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
			Error: &s,
		})
		return
	}

	var category models.Category
	err = models.DB.First(&category, uri.ID).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, CategoryEditable{})
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
			Error: &s,
		})
		return
	}

	var data CategoryEditable
	err = httputil.BindData(c, &data)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
			Error: &s,
		})
		return
	}

	err = models.DB.Model(&category).Select("", updateFields...).Updates(data.model()).Error
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
			Error: &s,
		})
		return
	}

	r, err := newCategory(c, models.DB, category)
	if err != nil {
		s := err.Error()
		c.JSON(status(err), CategoryResponse{
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
// @Failure		400	{object}	httpError
// @Failure		404	{object}	httpError
// @Failure		500	{object}	httpError
// @Param			id	path		URIID	true	"ignored, but needed: https://github.com/swaggo/swag/issues/1014"
// @Router			/v4/categories/{id} [delete]
func DeleteCategory(c *gin.Context) {
	var uri URIID
	err := c.ShouldBindUri(&uri)
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	var category models.Category
	err = models.DB.First(&category, uri.ID).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	err = models.DB.Delete(&category).Error
	if err != nil {
		c.JSON(status(err), httpError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
