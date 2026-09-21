package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
	"market-erp-backend/internal/middleware"
)

func RegisterStockMovementRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewStockMovementHandler(db)

	// Direct routes: /api/stock-movements
	api.GET("/stock-movements", handler.List)
	api.POST("/stock-movements", middleware.RequireRole("admin", "warehouse"), handler.Create)
	api.POST("/stock-movements/bulk", middleware.RequireRole("admin", "warehouse"), handler.BulkCreate)

	// Legacy / proxied routes: /api/admin/stock-movements
	admin := api.Group("/admin")
	{
		admin.GET("/stock-movements", handler.List)
		admin.POST("/stock-movements", middleware.RequireRole("admin", "warehouse"), handler.Create)
		admin.POST("/stock-movements/bulk", middleware.RequireRole("admin", "warehouse"), handler.BulkCreate)
	}
}
