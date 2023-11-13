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

// BindDataHandleErrors binds the data from the request to the struct passed in the interface.
//
// This function is deprecated. Use BindData(*gin.Context, any) httperrors.Error.
func BindDataHandleErrors(c *gin.Context, data interface{}) error {
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

// BindData binds the data from the request to the struct passed in the interface.
func BindData(c *gin.Context, data interface{}) httperrors.Error {
	if err := c.ShouldBindJSON(&data); err != nil {
		if errors.Is(io.EOF, err) {
			return httperrors.Error{
				Status: http.StatusBadRequest,
				Err:    httperrors.ErrRequestBodyEmpty,
			}
		}

		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		return httperrors.Error{
			Status: http.StatusBadRequest,
			Err:    httperrors.ErrInvalidBody,
		}
	}

	return httperrors.Error{}
}

// This is needed because gin does not support form binding to uuid.UUID currently.
// Follow https://github.com/gin-gonic/gin/pull/3045 to see when this gets resolved.
//
// This method is deprecated. Use UUIDFromString and handle errors in the calling method.
func UUIDFromStringHandleErrors(c *gin.Context, s string) (uuid.UUID, bool) {
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

// UUIDFromString binds a string to a UUID
//
// This is needed because gin does not support form binding to uuid.UUID currently.
// Follow https://github.com/gin-gonic/gin/pull/3045 to see when this gets resolved.
func UUIDFromString(s string) (uuid.UUID, httperrors.Error) {
	if s == "" {
		return uuid.Nil, httperrors.Error{}
	}

	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, httperrors.Error{
			Status: http.StatusBadRequest,
			Err:    httperrors.ErrInvalidUUID,
		}
	}

	return u, httperrors.Error{}
}
