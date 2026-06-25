package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
)

func RegisterMoneyAnalysisRoutes(api *gin.RouterGroup, db *gorm.DB) {
	handler := handlers.NewMoneyAnalysisHandler(db)
	api.GET("/money-analysis", handler.Get)
}
