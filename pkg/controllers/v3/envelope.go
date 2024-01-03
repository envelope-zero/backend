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

// EnvelopeCreate represents all user configurable parameters
type EnvelopeCreate struct {
	Name       string    `json:"name" gorm:"uniqueIndex:envelope_category_name" example:"Groceries" default:""`                       // Name of the envelope
	CategoryID uuid.UUID `json:"categoryId" gorm:"uniqueIndex:envelope_category_name" example:"878c831f-af99-4a71-b3ca-80deb7d793c1"` // ID of the category the envelope belongs to
	Note       string    `json:"note" example:"For stuff bought at supermarkets and drugstores" default:""`                           // Notes about the envelope
	Archived   bool      `json:"archived" example:"true" default:"false"`                                                             // Is the envelope archived?
}

// ToCreate transforms the API representation into the model representation
func (e EnvelopeCreate) ToCreate() models.EnvelopeCreate {
	return models.EnvelopeCreate{
		Name:       e.Name,
		CategoryID: e.CategoryID,
		Note:       e.Note,
		Archived:   e.Archived,
	}
}

type EnvelopeLinks struct {
	Self         string `json:"self" example:"https://example.com/api/v3/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166"`                     // The envelope itself
	Transactions string `json:"transactions" example:"https://example.com/api/v3/transactions?envelope=45b6b5b9-f746-4ae9-b77b-7688b91f8166"` // The envelope's transactions
	Month        string `json:"month" example:"https://example.com/api/v3/envelopes/45b6b5b9-f746-4ae9-b77b-7688b91f8166/YYYY-MM"`            // The MonthConfig for the envelope
}

func (l *EnvelopeLinks) links(c *gin.Context, e models.Envelope) {
	url := c.GetString(string(models.DBContextURL))
	self := fmt.Sprintf("%s/v3/envelopes/%s", url, e.ID)

	l.Self = self
	l.Transactions = fmt.Sprintf("%s/v3/transactions?envelope=%s", url, e.ID)
	l.Month = fmt.Sprintf("%s/v3/envelopes/%s/YYYY-MM", url, e.ID)
}

type Envelope struct {
	models.Envelope
	Links EnvelopeLinks `json:"links"` // Links to related resources
}

func getEnvelope(c *gin.Context, id uuid.UUID) (Envelope, httperrors.Error) {
	m, err := getResourceByID[models.Envelope](c, id)
	if !err.Nil() {
		return Envelope{}, httperrors.Error{}
	}

	r := Envelope{
		Envelope: m,
	}

	r.Links.links(c, m)
	return r, httperrors.Error{}
}

type EnvelopeListResponse struct {
	Data       []Envelope  `json:"data"`                                                          // List of Envelopes
	Error      *string     `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination `json:"pagination"`                                                    // Pagination information
}

type EnvelopeCreateResponse struct {
	Data  []EnvelopeResponse `json:"data"`                                                          // Data for the Envelope
	Error *string            `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

// appendError appends an EnvelopeResponse with the error and returns the updated HTTP status
func (e *EnvelopeCreateResponse) appendError(err httperrors.Error, status int) int {
	s := err.Error()
	e.Data = append(e.Data, EnvelopeResponse{Error: &s})

	// The final status code is the highest HTTP status code number
	if err.Status > status {
		status = err.Status
	}

	return status
}

type EnvelopeResponse struct {
	Data  *Envelope `json:"data"`                                                          // Data for the Envelope
	Error *string   `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type EnvelopeQueryFilter struct {
	Name       string `form:"name" filterField:"false"`   // By name
	CategoryID string `form:"category"`                   // By the ID of the category
	Note       string `form:"note" filterField:"false"`   // By the note
	Archived   bool   `form:"archived"`                   // Is the envelope archived?
	Search     string `form:"search" filterField:"false"` // By string in name or note
	Offset     uint   `form:"offset" filterField:"false"` // The offset of the first Envelope returned. Defaults to 0.
	Limit      int    `form:"limit" filterField:"false"`  // Maximum number of Envelopes to return. Defaults to 50.
}

func (f EnvelopeQueryFilter) ToCreate() (models.EnvelopeCreate, httperrors.Error) {
	categoryID, err := httputil.UUIDFromString(f.CategoryID)
	if !err.Nil() {
		return models.EnvelopeCreate{}, err
	}

	return models.EnvelopeCreate{
		CategoryID: categoryID,
		Archived:   f.Archived,
	}, httperrors.Error{}
}

// RegisterEnvelopeRoutes registers the routes for envelopes with
// the RouterGroup that is passed.
func RegisterEnvelopeRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsEnvelopeList)
		r.GET("", GetEnvelopes)
		r.POST("", CreateEnvelopes)
	}

	// Envelope with ID
	{
		r.OPTIONS("/:id", OptionsEnvelopeDetail)
		r.GET("/:id", GetEnvelope)
		r.PATCH("/:id", UpdateEnvelope)
		r.DELETE("/:id", DeleteEnvelope)
	}
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Envelopes
// @Success		204
// @Router			/v3/envelopes [options]
func OptionsEnvelopeList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Envelopes
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/envelopes/{id} [options]
func OptionsEnvelopeDetail(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	_, err = getResourceByID[models.Envelope](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// @Summary		Create envelope
// @Description	Creates a new envelope
// @Tags			Envelopes
// @Produce		json
// @Success		201			{object}	EnvelopeCreateResponse
// @Failure		400			{object}	EnvelopeCreateResponse
// @Failure		404			{object}	EnvelopeCreateResponse
// @Failure		500			{object}	EnvelopeCreateResponse
// @Param			envelope	body		[]v3.EnvelopeCreate	true	"Envelopes"
// @Router			/v3/envelopes [post]
func CreateEnvelopes(c *gin.Context) {
	var envelopes []EnvelopeCreate

	// Bind data and return error if not possible
	err := httputil.BindData(c, &envelopes)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, EnvelopeCreateResponse{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := EnvelopeCreateResponse{}

	for _, create := range envelopes {
		e := models.Envelope{
			EnvelopeCreate: create.ToCreate(),
		}

		// Verify that the category exists. If not, append the error
		// and move to the next envelope
		_, err := getResourceByID[models.Category](c, create.CategoryID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		dbErr := models.DB.Create(&e).Error
		if dbErr != nil {
			err := httperrors.GenericDBError[models.Envelope](e, c, dbErr)
			status = r.appendError(err, status)
			continue
		}

		eObject, err := getEnvelope(c, e.ID)
		if !err.Nil() {
			status = r.appendError(err, status)
			continue
		}

		r.Data = append(r.Data, EnvelopeResponse{Data: &eObject})
	}

	c.JSON(status, r)
}

// @Summary		Get envelopes
// @Description	Returns a list of envelopes
// @Tags			Envelopes
// @Produce		json
// @Success		200	{object}	EnvelopeListResponse
// @Failure		400	{object}	EnvelopeListResponse
// @Failure		500	{object}	EnvelopeListResponse
// @Router			/v3/envelopes [get]
// @Param			name		query	string	false	"Filter by name"
// @Param			note		query	string	false	"Filter by note"
// @Param			category	query	string	false	"Filter by category ID"
// @Param			archived	query	bool	false	"Is the envelope archived?"
// @Param			search		query	string	false	"Search for this text in name and note"
// @Param			offset		query	uint	false	"The offset of the first Envelope returned. Defaults to 0."
// @Param			limit		query	int		false	"Maximum number of Envelopes to return. Defaults to 50."
func GetEnvelopes(c *gin.Context) {
	var filter EnvelopeQueryFilter

	// The filters contain only strings, so this will always succeed
	_ = c.Bind(&filter)

	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate()
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeListResponse{
			Error: &s,
		})
		return
	}

	q := models.DB.
		Order("name ASC").
		Where(&models.Envelope{
			EnvelopeCreate: create,
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

	var envelopes []models.Envelope
	err = query(c, q.Find(&envelopes))

	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeListResponse{
			Error: &s,
		})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, EnvelopeListResponse{
			Error: &e,
		})
		return
	}

	r := make([]Envelope, 0, len(envelopes))
	for _, e := range envelopes {
		o, err := getEnvelope(c, e.ID)
		if !err.Nil() {
			s := err.Error()
			c.JSON(err.Status, EnvelopeListResponse{
				Error: &s,
			})
			return
		}

		r = append(r, o)
	}

	c.JSON(http.StatusOK, EnvelopeListResponse{
		Data: r,
		Pagination: &Pagination{
			Count:  len(r),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// @Summary		Get Envelope
// @Description	Returns a specific Envelope
// @Tags			Envelopes
// @Produce		json
// @Success		200	{object}	EnvelopeResponse
// @Failure		400	{object}	EnvelopeResponse
// @Failure		404	{object}	EnvelopeResponse
// @Failure		500	{object}	EnvelopeResponse
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/envelopes/{id} [get]
func GetEnvelope(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	m, err := getResourceByID[models.Envelope](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	r, err := getEnvelope(c, m.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, EnvelopeResponse{Data: &r})
}

// @Summary		Update envelope
// @Description	Updates an existing envelope. Only values to be updated need to be specified.
// @Tags			Envelopes
// @Accept			json
// @Produce		json
// @Success		200			{object}	EnvelopeResponse
// @Failure		400			{object}	EnvelopeResponse
// @Failure		404			{object}	EnvelopeResponse
// @Failure		500			{object}	EnvelopeResponse
// @Param			id			path		string				true	"ID formatted as string"
// @Param			envelope	body		v3.EnvelopeCreate	true	"Envelope"
// @Router			/v3/envelopes/{id} [patch]
func UpdateEnvelope(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	envelope, err := getResourceByID[models.Envelope](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, EnvelopeCreate{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	var data EnvelopeCreate
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	// Transform the API representation to the model representation
	e := models.Envelope{
		EnvelopeCreate: data.ToCreate(),
	}

	err = query(c, models.DB.Model(&envelope).Select("", updateFields...).Updates(e))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	r, err := getEnvelope(c, envelope.ID)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, EnvelopeResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, EnvelopeResponse{Data: &r})
}

// @Summary		Delete envelope
// @Description	Deletes an envelope
// @Tags			Envelopes
// @Success		204
// @Failure		400	{object}	httperrors.HTTPError
// @Failure		404	{object}	httperrors.HTTPError
// @Failure		500	{object}	httperrors.HTTPError
// @Param			id	path		string	true	"ID formatted as string"
// @Router			/v3/envelopes/{id} [delete]
func DeleteEnvelope(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	envelope, err := getResourceByID[models.Envelope](c, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	err = query(c, models.DB.Delete(&envelope))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
