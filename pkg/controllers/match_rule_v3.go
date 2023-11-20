package controllers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
)

// MatchRuleQueryFilter contains the fields that Match Rules can be filtered with.
type MatchRuleQueryFilterV3 struct {
	Priority  uint   `form:"priority"`                   // By priority
	Match     string `form:"match" filterField:"false"`  // By match
	AccountID string `form:"account"`                    // By ID of the Account they map to
	Offset    uint   `form:"offset" filterField:"false"` // The offset of the first Match Rule returned. Defaults to 0.
	Limit     int    `form:"limit" filterField:"false"`  // Maximum number of Match Rules to return. Defaults to 50.
}

// Parse returns a models.MatchRuleCreate struct that represents the MatchRuleQueryFilter.
func (f MatchRuleQueryFilterV3) Parse() (models.MatchRuleCreate, httperrors.Error) {
	envelopeID, err := httputil.UUIDFromString(f.AccountID)
	if !err.Nil() {
		return models.MatchRuleCreate{}, err
	}

	return models.MatchRuleCreate{
		Priority:  f.Priority,
		AccountID: envelopeID,
	}, httperrors.Error{}
}

type MatchRuleListResponseV3 struct {
	Data       []MatchRuleV3 `json:"data"`                                                          // List of Match Rules
	Error      *string       `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Pagination *Pagination   `json:"pagination"`                                                    // Pagination information
}

type MatchRuleCreateResponseV3 struct {
	Error *string               `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred
	Data  []MatchRuleResponseV3 `json:"data"`                                                          // List of created Match Rules
}

type MatchRuleResponseV3 struct {
	Error *string      `json:"error" example:"the specified resource ID is not a valid UUID"` // The error, if any occurred for this Match Rule
	Data  *MatchRuleV3 `json:"data"`                                                          // The Match Rule data, if creation was successful
}

// MatchRuleV3 is the API representation of a Match Rule.
type MatchRuleV3 struct {
	models.MatchRule
	Links struct {
		Self string `json:"self" example:"https://example.com/api/v3/match-rules/95685c82-53c6-455d-b235-f49960b73b21"` // The match rule itself
	} `json:"links"`
}

// links generates HATEOAS links for the Match Rule.
func (r *MatchRuleV3) links(c *gin.Context) {
	r.Links.Self = fmt.Sprintf("%s/v3/match-rules/%s", c.GetString(string(database.ContextURL)), r.ID)
}

func (co Controller) getMatchRuleV3(c *gin.Context, id uuid.UUID) (MatchRuleV3, httperrors.Error) {
	m, err := getResourceByID[models.MatchRule](c, co, id)
	if !err.Nil() {
		return MatchRuleV3{}, err
	}

	r := MatchRuleV3{
		MatchRule: m,
	}

	r.links(c)
	return r, httperrors.Error{}
}

// RegisterMatchRuleRoutesV3 registers the routes for matchRules with
// the RouterGroup that is passed.
func (co Controller) RegisterMatchRuleRoutesV3(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsMatchRuleListV3)
		r.GET("", co.GetMatchRulesV3)
		r.POST("", co.CreateMatchRulesV3)
	}

	// MatchRule with ID
	{
		r.OPTIONS("/:id", co.OptionsMatchRuleDetailV3)
		r.GET("/:id", co.GetMatchRuleV3)
		r.PATCH("/:id", co.UpdateMatchRuleV3)
		r.DELETE("/:id", co.DeleteMatchRuleV3)
	}
}

// OptionsMatchRuleListV3 returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			MatchRules
//	@Success		204
//	@Router			/v3/match-rules [options]
func (co Controller) OptionsMatchRuleListV3(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsMatchRuleDetaiV3 returns the allowed HTTP verbs
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			MatchRules
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v3/match-rules/{id} [options]
func (co Controller) OptionsMatchRuleDetailV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
	}

	_, err = getResourceByID[models.MatchRule](c, co, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// CreateMatchRulesV3 creates matchRules
//
//	@Summary		Create matchRules
//	@Description	Creates matchRules from the list of submitted matchRule data. The response code is the highest response code number that a single matchRule creation would have caused. If it is not equal to 201, at least one matchRule has an error.
//	@Tags			MatchRules
//	@Produce		json
//	@Success		201			{object}	MatchRuleCreateResponseV3
//	@Failure		400			{object}	MatchRuleCreateResponseV3
//	@Failure		404			{object}	MatchRuleCreateResponseV3
//	@Failure		500			{object}	MatchRuleCreateResponseV3
//	@Param			matchRules	body		[]models.MatchRuleCreate	true	"MatchRules"
//	@Router			/v3/match-rules [post]
func (co Controller) CreateMatchRulesV3(c *gin.Context) {
	var matchRules []models.MatchRuleCreate

	err := httputil.BindData(c, &matchRules)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleCreateResponseV3{
			Error: &e,
		})
		return
	}

	// The final http status. Will be modified when errors occur
	status := http.StatusCreated
	r := MatchRuleCreateResponseV3{}

	for _, create := range matchRules {
		m, err := co.createMatchRule(c, create)

		// Append the error or the successfully created transaction to the response list
		if !err.Nil() {
			e := err.Error()
			r.Data = append(r.Data, MatchRuleResponseV3{Error: &e})

			// The final status code is the highest HTTP status code number since this also
			// represents the priority we
			if err.Status > status {
				status = err.Status
			}
		} else {
			o, err := co.getMatchRuleV3(c, m.ID)
			if !err.Nil() {
				e := err.Error()
				c.JSON(err.Status, MatchRuleCreateResponseV3{
					Error: &e,
				})
				return
			}
			r.Data = append(r.Data, MatchRuleResponseV3{Data: &o})
		}
	}

	c.JSON(status, r)
}

// GetMatchRules returns a list of matchRules matching the search parameters
//
//	@Summary		Get matchRules
//	@Description	Returns a list of matchRules
//	@Tags			MatchRules
//	@Produce		json
//	@Success		200			{object}	MatchRuleListResponseV3
//	@Failure		400			{object}	MatchRuleListResponseV3
//	@Failure		500			{object}	MatchRuleListResponseV3
//	@Param			priority	query		uint	false	"Filter by priority"
//	@Param			match		query		string	false	"Filter by match"
//	@Param			account		query		string	false	"Filter by account ID"
//	@Param			offset		query		uint	false	"The offset of the first Match Rule returned. Defaults to 0."
//	@Param			limit		query		int		false	"Maximum number of Match Rules to return. Defaults to 50.".
//	@Router			/v3/match-rules [get]
func (co Controller) GetMatchRulesV3(c *gin.Context) {
	var filter MatchRuleQueryFilterV3
	if err := c.Bind(&filter); err != nil {
		s := httperrors.ErrInvalidQueryString.Error()
		c.JSON(http.StatusBadRequest, MatchRuleListResponseV3{
			Error: &s,
		})
		return
	}

	// Get the parameters set in the query string
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.Parse()
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleListResponseV3{Error: &e})
		return
	}

	q := co.DB.Where(&models.MatchRule{
		MatchRuleCreate: create,
	}, queryFields...)

	// Filter for match containing the query string or explicitly empty one
	if filter.Match != "" {
		q = q.Where("match LIKE ?", fmt.Sprintf("%%%s%%", filter.Match))
	} else if slices.Contains(setFields, "Match") {
		q = q.Where("match = ''")
	}

	// Set the offset. Does not need checking since the default is 0
	q = q.Offset(int(filter.Offset))

	// Default to 50 Match Rules and set the limit
	limit := 50
	if slices.Contains(setFields, "Limit") {
		limit = int(filter.Limit)
	}
	q = q.Limit(limit)

	// Execute the query
	var matchRules []models.MatchRule
	err = query(c, q.Find(&matchRules))

	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleListResponseV3{Error: &e})
		return
	}

	var count int64
	err = query(c, q.Limit(-1).Offset(-1).Count(&count))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleListResponseV3{
			Error: &e,
		})
		return
	}

	mrs := make([]MatchRuleV3, 0)
	for _, t := range matchRules {
		o, err := co.getMatchRuleV3(c, t.ID)
		if !err.Nil() {
			e := err.Error()
			c.JSON(err.Status, MatchRuleListResponseV3{
				Error: &e,
			})
			return
		}

		mrs = append(mrs, o)
	}

	c.JSON(http.StatusOK, MatchRuleListResponseV3{
		Data: mrs,
		Pagination: &Pagination{
			Count:  len(mrs),
			Total:  count,
			Offset: filter.Offset,
			Limit:  limit,
		},
	})
}

// GetMatchRule returns data about a specific matchRule
//
//	@Summary		Get matchRule
//	@Description	Returns a specific matchRule
//	@Tags			MatchRules
//	@Produce		json
//	@Success		200	{object}	MatchRuleResponseV3
//	@Failure		400	{object}	MatchRuleResponseV3
//	@Failure		404	{object}	MatchRuleResponseV3
//	@Failure		500	{object}	MatchRuleResponseV3
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v3/match-rules/{id} [get]
func (co Controller) GetMatchRuleV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	o, err := co.getMatchRuleV3(c, id)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	c.JSON(http.StatusOK, MatchRuleResponseV3{
		Data: &o,
	})
}

// UpdateMatchRule updates matchRule data
//
//	@Summary		Update matchRule
//	@Description	Update a matchRule. Only values to be updated need to be specified.
//	@Tags			MatchRules
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	MatchRuleResponseV3
//	@Failure		400			{object}	MatchRuleResponseV3
//	@Failure		404			{object}	MatchRuleResponseV3
//	@Failure		500			{object}	MatchRuleResponseV3
//	@Param			id			path		string					true	"ID formatted as string"
//	@Param			matchRule	body		models.MatchRuleCreate	true	"MatchRule"
//	@Router			/v3/match-rules/{id} [patch]
func (co Controller) UpdateMatchRuleV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	matchRule, err := getResourceByID[models.MatchRule](c, co, id)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.MatchRuleCreate{})
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	var data models.MatchRule
	err = httputil.BindData(c, &data.MatchRuleCreate)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	// Check that the referenced account exists
	if slices.Contains(updateFields, "AccountID") {
		_, err = getResourceByID[models.Account](c, co, data.AccountID)
		if !err.Nil() {
			e := err.Error()
			c.JSON(err.Status, MatchRuleResponseV3{
				Error: &e,
			})
			return
		}
	}

	err = query(c, co.DB.Model(&matchRule).Select("", updateFields...).Updates(data))
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	o, err := co.getMatchRuleV3(c, matchRule.ID)
	if !err.Nil() {
		e := err.Error()
		c.JSON(err.Status, MatchRuleResponseV3{
			Error: &e,
		})
		return
	}

	c.JSON(http.StatusOK, MatchRuleResponseV3{
		Data: &o,
	})
}

// DeleteMatchRule deletes an matchRule
//
//	@Summary		Delete matchRule
//	@Description	Deletes an matchRule
//	@Tags			MatchRules
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Param			id	path		string	true	"ID formatted as string"
//	@Router			/v3/match-rules/{id} [delete]
func (co Controller) DeleteMatchRuleV3(c *gin.Context) {
	id, err := httputil.UUIDFromString(c.Param("id"))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}
	matchRule, err := getResourceByID[models.MatchRule](c, co, id)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}

	err = query(c, co.DB.Delete(&matchRule))
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
