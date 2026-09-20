package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
	"market-erp-backend/internal/middleware"
)

func RegisterAdminEmployeeRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewAdminEmployeeHandler(db)

	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("/employees", handler.List)
		admin.POST("/employees", handler.Create)
		admin.PUT("/employees/:id", handler.Update)
		admin.PATCH("/employees/:id/status", handler.UpdateStatus)
		admin.PATCH("/employees/:id/password", handler.UpdatePassword)
		admin.DELETE("/employees/:id", handler.Delete)
	}
}
