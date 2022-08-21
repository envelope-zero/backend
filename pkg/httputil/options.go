package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

func OptionsGet(c *gin.Context) {
	c.Header("allow", "GET")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsGetPost(c *gin.Context) {
	c.Header("allow", "GET, POST")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsGetDelete(c *gin.Context) {
	c.Header("allow", "GET, DELETE")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsGetPatchDelete(c *gin.Context) {
	c.Header("allow", "GET, PATCH, DELETE")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsDelete(c *gin.Context) {
	c.Header("allow", "DELETE")
	c.Render(http.StatusNoContent, render.JSON{})
}
