package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/services"
)

type DebtSnapshotHandler struct{ DB *gorm.DB }

func NewDebtSnapshotHandler(db *gorm.DB) *DebtSnapshotHandler {
	return &DebtSnapshotHandler{DB: db}
}

func (h *DebtSnapshotHandler) Get(c *gin.Context) {
	date, valid := debtSnapshotDate(c)
	if !valid {
		return
	}

	snapshot, err := services.DebtSnapshotAt(h.DB, date)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, snapshot)
}

func debtSnapshotDate(c *gin.Context) (time.Time, bool) {
	value := c.Query("date")
	if value == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), true
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		fail(c, http.StatusBadRequest, "Geçerli tarih giriniz")
		return time.Time{}, false
	}
	return date, true
}
