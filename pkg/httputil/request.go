package httputil

import (
	"errors"
	"io"
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// BindData binds the data from the request to the struct passed in the interface.
func BindData(c *gin.Context, data interface{}) error {
	if err := c.ShouldBindJSON(&data); err != nil {
		if errors.Is(io.EOF, err) {
			e := errors.New("request body must not be empty")
			httperrors.New(c, http.StatusBadRequest, e.Error())
			return e
		}

		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		e := errors.New("the body of your request contains invalid or un-parseable data. Please check and try again")
		httperrors.New(c, http.StatusBadRequest, e.Error())
		return e
	}

	return nil
}

// This is needed because gin does not support form binding to uuid.UUID currently.
// Follow https://github.com/gin-gonic/gin/pull/3045 to see when this gets resolved.
func UUIDFromString(c *gin.Context, s string) (uuid.UUID, bool) {
	if s == "" {
		return uuid.Nil, true
	}

	u, err := uuid.Parse(s)
	if err != nil {
		httperrors.InvalidUUID(c)
		return uuid.Nil, false
	}

	return u, true
}
