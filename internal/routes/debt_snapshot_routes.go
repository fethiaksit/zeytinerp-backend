package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterDebtSnapshotRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewDebtSnapshotHandler(db)
	api.GET("/debt-snapshot", handler.Get)
}
