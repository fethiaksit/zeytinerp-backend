package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
	"market-erp-backend/internal/middleware"
)

func RegisterCustomerTransactionRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewCustomerTransactionHandler(db)

	// Standard transactions endpoints under /api/customers/:id/transactions
	api.GET("/customers/:id/transactions", handler.List)
	api.POST("/customers/:id/transactions", handler.Create)

	// Legacy / proxied transactions endpoints
	api.POST("/customer-transactions", handler.Create)
	api.GET("/customer-transactions", handler.List)
	api.DELETE("/customer-transactions/:id", middleware.RequireAdmin(), handler.Delete)

	admin := api.Group("/admin")
	{
		admin.GET("/customers/:id/transactions", handler.List)
		admin.POST("/customers/:id/transactions", handler.Create)
		admin.POST("/customer-transactions", handler.Create)
		admin.GET("/customer-transactions", handler.List)
		admin.DELETE("/customer-transactions/:id", middleware.RequireAdmin(), handler.Delete)
	}
}
