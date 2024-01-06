package v3

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/envelope-zero/backend/v4/pkg/httperrors"
	"github.com/envelope-zero/backend/v4/pkg/httputil"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MonthConfigFilter struct {
	EnvelopeID uuid.UUID
	Month      types.Month
}

type MonthConfigEditable struct {
	EnvelopeID uuid.UUID       `json:"envelopeId" gorm:"primaryKey" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"`                                      // ID of the envelope
	Month      types.Month     `json:"month" gorm:"primaryKey" example:"1969-06-01T00:00:00.000000Z"`                                                    // The month. This is always set to 00:00 UTC on the first of the month.
	Allocation decimal.Decimal `json:"allocation" gorm:"-" example:"22.01" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
	Note       string          `json:"note" example:"Added 200€ here because we replaced Tim's expensive vase" default:""`                               // A note for the month config
}

type MonthConfig struct {
	models.MonthConfig
	Links struct {
		Self     string `json:"self" example:"https://example.com/api/v3/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b/2017-10"` // The Month Config itself
		Envelope string `json:"envelope" example:"https://example.com/api/v3/envelopes/61027ebb-ab75-4a49-9e23-a104ddd9ba6b"`     // The Envelope this config belongs to
	} `json:"links"`
}

func (m *MonthConfig) links(c *gin.Context) {
	url := c.GetString(string(models.DBContextURL))

	m.Links.Self = fmt.Sprintf("%s/v3/envelopes/%s/%s", url, m.EnvelopeID, m.Month)
	m.Links.Envelope = fmt.Sprintf("%s/v3/envelopes/%s", url, m.EnvelopeID)
}

func getMonthConfig(c *gin.Context, id uuid.UUID, month types.Month) (MonthConfig, httperrors.Error) {
	m, err := getMonthConfigModel(c, id, month)
	if !err.Nil() {
		return MonthConfig{}, err
	}

	r := MonthConfig{
		MonthConfig: m,
	}

	r.links(c)
	return r, httperrors.Error{}
}

func getMonthConfigModel(c *gin.Context, id uuid.UUID, month types.Month) (models.MonthConfig, httperrors.Error) {
	var m models.MonthConfig

	err := query(c, models.DB.First(&m, &models.MonthConfig{
		EnvelopeID: id,
		Month:      month,
	}))

	if !err.Nil() {
		return models.MonthConfig{}, err
	}

	return m, httperrors.Error{}
}

type MonthConfigResponse struct {
	Data  *MonthConfig `json:"data"`                                                          // Config for the month
	Error *string      `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
}

type MonthConfigListResponse struct {
	Data       []MonthConfig `json:"data"`                                                          // List of Month Configs
	Error      *string       `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination   `json:"pagination"`                                                    // Pagination information
}

// RegisterMonthConfigRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func RegisterMonthConfigRoutes(r *gin.RouterGroup) {
	r.OPTIONS("/:id/:month", OptionsMonthConfigDetail)
	r.GET("/:id/:month", GetMonthConfig)
	r.PATCH("/:id/:month", UpdateMonthConfig)
}

// @Summary		Allowed HTTP verbs
// @Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags			Envelopes
// @Success		204
// @Failure		400		{object}	httperrors.HTTPError
// @Param			id		path		string	true	"ID of the Envelope"
// @Param			month	path		string	true	"The month in YYYY-MM format"
// @Router			/v3/envelopes/{id}/{month} [options]
func OptionsMonthConfigDetail(c *gin.Context) {
	_, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		e := httperrors.Parse(c, err)
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	httputil.OptionsGetPatch(c)
}

// @Summary		Get MonthConfig
// @Description	Returns configuration for a specific month
// @Tags			Envelopes
// @Produce		json
// @Success		200		{object}	MonthConfigResponse
// @Failure		400		{object}	MonthConfigResponse
// @Failure		404		{object}	MonthConfigResponse
// @Failure		500		{object}	MonthConfigResponse
// @Param			id		path		string	true	"ID of the Envelope"
// @Param			month	path		string	true	"The month in YYYY-MM format"
// @Router			/v3/envelopes/{id}/{month} [get]
func GetMonthConfig(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		e := httperrors.Parse(c, err)
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	_, err = getModelByID[models.Envelope](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	mConfig, err := getMonthConfig(c, id, types.MonthOf(month.Month))
	if !err.Nil() {
		// If there is no MonthConfig in the database, return one with the zero values
		if errors.Is(err.Err, httperrors.ErrNoResource) {
			mConfig := MonthConfig{
				MonthConfig: models.MonthConfig{
					EnvelopeID: id,
					Month:      types.MonthOf(month.Month),
				},
			}
			mConfig.links(c)

			c.JSON(http.StatusOK, MonthConfigResponse{Data: &mConfig})
			return
		}

		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponse{Data: &mConfig})
}

// MonthConfigCreate contains the fields relevant for MonthConfigs in APIv3.
type MonthConfigCreate struct {
	Note       string          `json:"note" example:"Added 200€ here because we replaced Tim's expensive vase" default:""`                                                // A note for the month config
	Allocation decimal.Decimal `json:"allocation" gorm:"type:DECIMAL(20,8)" example:"22.01" minimum:"0.00000001" maximum:"999999999999.99999999" multipleOf:"0.00000001"` // The maximum value is "999999999999.99999999", swagger unfortunately rounds this.
}

// ToCreate is used to transform the API representation into the model representation
func (m MonthConfigCreate) ToCreate() (create models.MonthConfigCreate) {
	create.Note = m.Note
	create.Allocation = m.Allocation
	return create
}

// @Summary		Update MonthConfig
// @Description	Changes configuration for a Month. If there is no configuration for the month yet, this endpoint transparently creates a configuration resource.
// @Tags			Envelopes
// @Produce		json
// @Success		201			{object}	MonthConfigResponse
// @Failure		400			{object}	MonthConfigResponse
// @Failure		404			{object}	MonthConfigResponse
// @Failure		500			{object}	MonthConfigResponse
// @Param			id			path		string				true	"ID of the Envelope"
// @Param			month		path		string				true	"The month in YYYY-MM format"
// @Param			monthConfig	body		MonthConfigCreate	true	"MonthConfig"
// @Router			/v3/envelopes/{id}/{month} [patch]
func UpdateMonthConfig(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		e := httperrors.Parse(c, err)
		s := e.Error()
		c.JSON(e.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	_, err = getModelByID[models.Envelope](c, id)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	m, err := getMonthConfigModel(c, id, types.MonthOf(month.Month))
	if !err.Nil() {
		// If no Month Config exists yet, create one
		// If the error is another error, return it to the user
		if errors.Is(err.Err, httperrors.ErrNoResource) {
			e := models.DB.Create(&models.MonthConfig{
				Month:      types.MonthOf(month.Month),
				EnvelopeID: id,
			}).Error

			if e != nil {
				err = httperrors.Parse(c, err)
				s := e.Error()
				c.JSON(err.Status, MonthConfigResponse{
					Error: &s,
				})
			}

			m, err = getMonthConfigModel(c, id, types.MonthOf(month.Month))
			if !err.Nil() {
				s := err.Error()
				c.JSON(err.Status, MonthConfigResponse{
					Error: &s,
				})
				return
			}
		} else {
			s := err.Error()
			c.JSON(err.Status, MonthConfigResponse{
				Error: &s,
			})
			return
		}
	}

	updateFields, err := httputil.GetBodyFields(c, MonthConfigCreate{})
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	var data MonthConfigCreate
	err = httputil.BindData(c, &data)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	create := data.ToCreate()

	err = query(c, models.DB.Model(&m).Select("", updateFields...).Updates(create))
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	o, err := getMonthConfig(c, m.EnvelopeID, m.Month)
	if !err.Nil() {
		s := err.Error()
		c.JSON(err.Status, MonthConfigResponse{
			Error: &s,
		})
		return
	}

	c.JSON(http.StatusOK, MonthConfigResponse{Data: &o})
}
