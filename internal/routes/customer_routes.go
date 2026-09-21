package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
	"market-erp-backend/internal/middleware"
)

func RegisterCustomerRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewCustomerHandler(db)

	api.GET("/customers", handler.List)
	api.GET("/customers/:id", handler.Get)
	api.GET("/customers/:id/balance", handler.Balance)

	admin := api.Group("/customers")
	admin.Use(middleware.RequireAdmin())
	{
		admin.POST("", handler.Create)
		admin.PUT("/:id", handler.Update)
		admin.DELETE("/:id", handler.Delete)
	}

	// Legacy / proxied routes: /api/admin/customers
	adminRoutes := api.Group("/admin/customers")
	{
		adminRoutes.GET("", handler.List)
		adminRoutes.GET("/:id", handler.Get)
		adminRoutes.GET("/:id/balance", handler.Balance)
	}
	adminRoutesAuth := api.Group("/admin/customers")
	adminRoutesAuth.Use(middleware.RequireAdmin())
	{
		adminRoutesAuth.POST("", handler.Create)
		adminRoutesAuth.PUT("/:id", handler.Update)
		adminRoutesAuth.DELETE("/:id", handler.Delete)
	}
}
