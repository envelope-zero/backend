package httputil_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v7/internal/httputil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestBindData verifies that BindData succeeds on valid data.
func TestBindData(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(_ *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		err := httputil.BindData(c, &o)
		assert.Nil(t, err)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte(`{ "name": "Drink more water!" }`)))
	r.ServeHTTP(w, c.Request)
}

// TestBindDataInvalidBody verifies that BindData returns the correct error on an invalid body.
func TestBindDataInvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(_ *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		err := httputil.BindData(c, &o)
		assert.ErrorIs(t, err, httputil.ErrInvalidBody)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte(`{ invalid json: "Drink more water! }`)))
	r.ServeHTTP(w, c.Request)
}

// TestBindDataEmptyBody verifies that BindData returns the correct error on an empty body.
func TestBindDataEmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(_ *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		err := httputil.BindData(c, &o)
		assert.ErrorIs(t, err, httputil.ErrRequestBodyEmpty)
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte("")))
	r.ServeHTTP(w, c.Request)
}

// TestBindDataJsonUnmarshalTypeError verifies that BindData returns the correct error on a type error
func TestBindDataJsonUnmarshalTypeError(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/", func(_ *gin.Context) {
		var o struct {
			Name string `json:"name"`
		}

		err := httputil.BindData(c, &o)
		assert.Equal(t, "json: cannot unmarshal number into Go struct field .name of type string", err.Error())
	})

	c.Request, _ = http.NewRequest(http.MethodGet, "https://example.com/", bytes.NewBuffer([]byte(`{ "name": 2 }`)))
	r.ServeHTTP(w, c.Request)
}
