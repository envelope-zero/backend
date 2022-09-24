package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

func OptionsGet(c *gin.Context) {
	c.Header("allow", "OPTIONS, GET")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsPost(c *gin.Context) {
	c.Header("allow", "OPTIONS, POST")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsGetPost(c *gin.Context) {
	c.Header("allow", "OPTIONS, GET, POST")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsGetDelete(c *gin.Context) {
	c.Header("allow", "OPTIONS, GET, DELETE")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsGetPatchDelete(c *gin.Context) {
	c.Header("allow", "OPTIONS, GET, PATCH, DELETE")
	c.Render(http.StatusNoContent, render.JSON{})
}

func OptionsDelete(c *gin.Context) {
	c.Header("allow", "OPTIONS, DELETE")
	c.Render(http.StatusNoContent, render.JSON{})
}
