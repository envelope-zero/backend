package httputil

import (
	"net/url"
	"reflect"
)

func GetFields(url *url.URL, filter any) []any {
	var queryFields []any

	// Add all parameters set in the query string to the queryFields
	// This is used to determine which fields are queried in the database
	val := reflect.Indirect(reflect.ValueOf(filter))
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i).Name
		param := val.Type().Field(i).Tag.Get("form")

		if url.Query().Has(param) {
			queryFields = append(queryFields, field)
		}
	}
	return queryFields
}
