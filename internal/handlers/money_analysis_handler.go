package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/services"
)

type MoneyAnalysisHandler struct{ DB *gorm.DB }

func NewMoneyAnalysisHandler(db *gorm.DB) *MoneyAnalysisHandler {
	return &MoneyAnalysisHandler{DB: db}
}

func (h *MoneyAnalysisHandler) Get(c *gin.Context) {
	monthStart, valid := moneyAnalysisMonth(c)
	if !valid {
		return
	}

	analysis, err := services.MoneyAnalysisForMonth(h.DB, monthStart)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, analysis)
}

func moneyAnalysisMonth(c *gin.Context) (time.Time, bool) {
	value := c.Query("month")
	if value == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), true
	}
	month, err := time.Parse("2006-01", value)
	if err != nil {
		fail(c, http.StatusBadRequest, "Geçerli ay giriniz")
		return time.Time{}, false
	}
	return time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.Local), true
}
