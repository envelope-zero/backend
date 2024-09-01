package httputil

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// BindData binds the data from the request to the struct passed in the interface.
func BindData(c *gin.Context, data interface{}) error {
	if err := c.ShouldBindJSON(&data); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrRequestBodyEmpty
		}

		var jsonUnmarshalTypeError *json.UnmarshalTypeError
		if errors.As(err, &jsonUnmarshalTypeError) {
			return err
		}

		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		return ErrInvalidBody
	}

	return nil
}

// UUIDFromString binds a string to a UUID
//
// This is needed because gin does not support form binding to uuid.UUID currently.
// Follow https://github.com/gin-gonic/gin/pull/3045 to see when this gets resolved.
func UUIDFromString(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}

	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, ErrInvalidUUID
	}

	return u, nil
}
