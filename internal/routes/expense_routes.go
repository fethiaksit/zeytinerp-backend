package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterExpenseRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewExpenseHandler(db)
	api.POST("/expenses", handler.Create)
	api.GET("/expenses", handler.List)
	api.GET("/expenses/by-date", handler.ListByDate)
	api.PUT("/expenses/:id", handler.Update)
	api.DELETE("/expenses/:id", handler.Delete)
}
