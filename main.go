package main

import (
	"os"

	"github.com/envelope-zero/backend/internal/routing"
	"github.com/gin-gonic/gin"
)

func main() {
	// gin uses debug as the default mode, we use release for
	// security reasons
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode("release")
	}

	r := routing.Router()
	r.Run()
}
