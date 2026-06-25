package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterSupplierTransactionRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewSupplierTransactionHandler(db)
	api.POST("/supplier-transactions", handler.Create)
	api.GET("/supplier-transactions", handler.List)
	api.GET("/supplier-transactions/:id", handler.Get)
	api.PUT("/supplier-transactions/:id", handler.Update)
	api.DELETE("/supplier-transactions/:id", handler.Delete)
	api.DELETE("/invoices", handler.DeleteInvoice)
	api.DELETE("/invoices/:id", handler.DeleteInvoice)
	api.POST("/supplier-transactions/:id/files", handler.UploadFiles)
	api.GET("/supplier-transactions/:id/files", handler.ListFiles)
	api.DELETE("/supplier-transaction-files/:file_id", handler.DeleteFile)
}
