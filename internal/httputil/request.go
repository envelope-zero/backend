package httputil

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
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
