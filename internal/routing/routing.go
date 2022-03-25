package routing

import (
	"log"
	"net/http"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// Router controls the routes for the API.
func Router() *gin.Engine {
	r := gin.Default()

	err := models.ConnectDatabase()
	if err != nil {
		log.Printf("Database migration failed with: %s", err.Error())
	}

	// The root path lists the available versions
	r.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"v1": "/v1",
		})
	})

	// Options lists the allowed HTTP verbs
	r.OPTIONS("", func(c *gin.Context) {
		c.Header("allow", "GET")
	})

	// API v1 setup
	v1 := r.Group("/v1")
	{
		v1.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"budgets": "/budgets",
			})
		})

		v1.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET")
		})
	}

	budgets := v1.Group("/budgets")
	controllers.RegisterBudgetRoutes(budgets)

	return r
}
