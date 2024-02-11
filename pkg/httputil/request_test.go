package httputil_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envelope-zero/backend/v5/pkg/httputil"
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

func TestUUIDFromString(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		status int // the expected http status
	}{
		{"Success", "https://example.com/?id=4e743e94-6a4b-44d6-aba5-d77c82103fa7", http.StatusOK},
		{"Invalid UUID", "https://example.com/?id=not-a-valid-uuid", http.StatusBadRequest},
		{"Empty UUID", "https://example.com/?id=", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.GET("/", func(_ *gin.Context) {
				var o struct {
					UUID string `form:"id"`
				}

				_ = c.Bind(&o)
				_, err := httputil.UUIDFromString(o.UUID)
				if err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
				}
				c.Status(http.StatusOK)
			})

			c.Request, _ = http.NewRequest(http.MethodGet, tt.url, bytes.NewBuffer([]byte("")))
			r.ServeHTTP(w, c.Request)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}
