package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func OptionsGet(c *gin.Context) {
	c.Header("allow", "GET")
	c.Status(http.StatusNoContent)
}

func OptionsGetPost(c *gin.Context) {
	c.Header("allow", "GET, POST")
	c.Status(http.StatusNoContent)
}

func OptionsGetPatchDelete(c *gin.Context) {
	c.Header("allow", "GET, PATCH, DELETE")
	c.Status(http.StatusNoContent)
}
