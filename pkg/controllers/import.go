package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/importer"
	"github.com/envelope-zero/backend/pkg/importer/parser/ynab4"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ImportQuery struct {
	BudgetName string `form:"budgetName" binding:"required"`
}

// RegisterImportRoutes registers the routes for imports.
func (co Controller) RegisterImportRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsImport)
		r.POST("", co.Import)

		r.OPTIONS("/ynab4", co.OptionsImportYnab4)
		r.POST("/ynab4", co.ImportYnab4)
	}
}

// OptionsImport returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs. **Please use /v1/import/ynab4, which works exactly the same.**
//	@Tags			Import
//	@Success		204
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/import [options]
//	@Deprecated		true
func (co Controller) OptionsImport(c *gin.Context) {
	httputil.OptionsPost(c)
}

// OptionsImportYnab4 returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Import
//	@Success		204
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/import/ynab4 [options]
func (co Controller) OptionsImportYnab4(c *gin.Context) {
	httputil.OptionsPost(c)
}

// Import imports a YNAB 4 budget
//
//	@Summary		Import
//	@Description	Imports budgets from YNAB 4. **Please use /v1/import/ynab4, which works exactly the same.**
//	@Tags			Import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			file		formData	file	true	"File to import"
//	@Param			budgetName	query		string	false	"Name of the Budget to create"
//	@Router			/v1/import [post]
//	@Deprecated		true
func (co Controller) Import(c *gin.Context) {
	co.ImportYnab4(c)
}

// ImportYnab4 imports a YNAB 4 budget
//
//	@Summary		Import
//	@Description	Imports budgets from YNAB 4
//	@Tags			Import
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		201			{object}	BudgetResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			file		formData	file	true	"File to import"
//	@Param			budgetName	query		string	false	"Name of the Budget to create"
//	@Router			/v1/import/ynab4 [post]
func (co Controller) ImportYnab4(c *gin.Context) {
	var query ImportQuery
	if err := c.BindQuery(&query); err != nil {
		httperrors.New(c, http.StatusBadRequest, "The budgetName parameter must be set")
		return
	}

	// Verify if the budget does already exist. If yes, return an error
	// as we only allow imports to new budgets
	var budget models.Budget
	err := co.DB.Where(&models.Budget{
		BudgetCreate: models.BudgetCreate{
			Name: query.BudgetName,
		},
	}).First(&budget).Error

	if err == nil {
		httperrors.New(c, http.StatusBadRequest, "This budget name is already in use. Imports from YNAB 4 create a new budget, therefore the name needs to be unique.")
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		httperrors.Handler(c, err)
		return
	}

	formFile, err := c.FormFile("file")
	if formFile == nil {
		httperrors.New(c, http.StatusBadRequest, "You must send a file to this endpoint")
		return
	} else if err != nil && err.Error() == "unexpected EOF" {
		httperrors.New(c, http.StatusBadRequest, "The file you uploaded is empty. Did the file get deleted before you uploaded it?")
		return
	} else if err != nil {
		httperrors.Handler(c, err)
		return
	}

	if !strings.HasSuffix(formFile.Filename, ".yfull") {
		httperrors.New(c, http.StatusBadRequest, "Import currently only supports YNAB 4 budgets. If you tried to upload a YNAB 4 budget, make sure its file name ends with .yfull")
		return
	}

	f, err := formFile.Open()
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	// Parse the Budget.yfull
	resources, err := ynab4.Parse(f)
	if err != nil {
		httperrors.New(c, http.StatusBadRequest, err.Error())
		return
	}

	budget, err = importer.Create(co.DB, query.BudgetName, resources)
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	b, ok := co.getBudgetObject(c, budget.ID)
	if !ok {
		return
	}
	c.JSON(http.StatusCreated, BudgetResponse{Data: b})
}
