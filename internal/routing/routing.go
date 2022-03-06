package routing

import (
	"net/http"

	"github.com/envelope-zero/backend/internal/controllers"
	"github.com/envelope-zero/backend/internal/database"
	"github.com/gin-gonic/gin"
)

// Router controls the routes for the API
func Router() *gin.Engine {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "hello world"})
	})

	database.ConnectDatabase()

	v1 := r.Group("/v1")
	{
		v1.GET("/budgets", controllers.GetBudgets)
		v1.GET("/budgets/:id", controllers.GetBudget)
		v1.POST("/budgets", controllers.CreateBudget)
		v1.PATCH("budgets/:id", controllers.UpdateBudget)
		v1.DELETE("budgets/:id", controllers.DeleteBudget)
	}

	return r
}
