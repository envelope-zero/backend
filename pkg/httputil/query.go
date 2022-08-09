package httputil

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"

	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func GetURLFields(url *url.URL, filter any) []any {
	var queryFields []any

	// Add all parameters set in the query string to the queryFields
	// This is used to determine which fields are queried in the database
	val := reflect.Indirect(reflect.ValueOf(filter))
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i).Name
		param := val.Type().Field(i).Tag.Get("form")

		// createField is a struct tag that allows to specify if the field is part
		// of the fields to filter for on the original struct
		createField := val.Type().Field(i).Tag.Get("createField")

		if url.Query().Has(param) && createField != "false" {
			queryFields = append(queryFields, field)
		}
	}
	return queryFields
}

// GetBodyFields returns a slice of strings with the field names
// of the resource passed in. Only names of fields which are set
// in the body are contained in that slice.
//
// This function reads and copies the reuqest body, it must always
// be called before any of gin's c.*Bind methods.
func GetBodyFields(c *gin.Context, resource any) ([]any, error) {
	// Copy the body to be able to use it multiple times
	body, _ := ioutil.ReadAll(c.Request.Body)
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	// Parse the body into a map to have all fields available
	var mapBody map[string]any

	if err := json.Unmarshal(body, &mapBody); err != nil {
		log.Error().Str("request-id", requestid.Get(c)).Msgf("%T: %v", err, err.Error())
		e := errors.New("the body of your request contains invalid or un-parseable data. Please check and try again")
		httperrors.New(c, http.StatusBadRequest, e.Error())
		return []any{}, e
	}

	var bodyFields []any
	// Add all parameters set in the body to the bodyFields
	// This is used to determine which fields are updated in the database
	val := reflect.Indirect(reflect.ValueOf(resource))
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i).Name
		param := val.Type().Field(i).Tag.Get("json")

		// If the request Body has the field, add it to the return value
		if _, ok := mapBody[param]; ok {
			bodyFields = append(bodyFields, field)
		}
	}
	return bodyFields, nil
}
