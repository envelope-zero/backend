package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/envelope-zero/backend/internal/routing"
)

func main() {
	// gin uses debug as the default mode, we use release for
	// security reasons
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode("release")
	}

	r, err := routing.Router()
	if err != nil {
		log.Fatal(err)
	}

	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}
