package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/services"
)

type FinanceCenterHandler struct{ DB *gorm.DB }

func NewFinanceCenterHandler(db *gorm.DB) *FinanceCenterHandler {
	return &FinanceCenterHandler{DB: db}
}

func (h *FinanceCenterHandler) Summary(c *gin.Context) {
	data, err := services.FinanceCenterSummaryData(h.DB)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, data)
}

func (h *FinanceCenterHandler) History(c *gin.Context) {
	date, valid := financeCenterDate(c, "date")
	if !valid {
		return
	}
	data, err := services.FinanceCenterHistoryAt(h.DB, date)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, data)
}

func (h *FinanceCenterHandler) MoneyFlow(c *gin.Context) {
	start, end, valid := financeCenterDateRange(c)
	if !valid {
		return
	}
	data, err := services.FinanceCenterMoneyFlowData(h.DB, start, end)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, data)
}

func (h *FinanceCenterHandler) DebtDistribution(c *gin.Context) {
	data, err := services.FinanceCenterDebtDistributionData(h.DB)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, data)
}

func (h *FinanceCenterHandler) Cashflow(c *gin.Context) {
	data, err := services.FinanceCenterCashflowData(h.DB, time.Now(), 12)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, data)
}

func (h *FinanceCenterHandler) Alerts(c *gin.Context) {
	data, err := services.FinanceCenterAlertsData(h.DB)
	if err != nil {
		handleDBError(c, err)
		return
	}
	ok(c, data)
}

func financeCenterDate(c *gin.Context, key string) (time.Time, bool) {
	value := c.Query(key)
	if value == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), true
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		fail(c, http.StatusBadRequest, "Geçerli tarih giriniz")
		return time.Time{}, false
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local), true
}

func financeCenterDateRange(c *gin.Context) (time.Time, time.Time, bool) {
	now := time.Now()
	defaultStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	start := defaultStart
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if value := c.Query("start_date"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			fail(c, http.StatusBadRequest, "Geçerli tarih giriniz")
			return time.Time{}, time.Time{}, false
		}
		start = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local)
	}
	if value := c.Query("end_date"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			fail(c, http.StatusBadRequest, "Geçerli tarih giriniz")
			return time.Time{}, time.Time{}, false
		}
		end = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.Local)
	}
	if end.Before(start) {
		fail(c, http.StatusBadRequest, "Bitiş tarihi başlangıç tarihinden önce olamaz")
		return time.Time{}, time.Time{}, false
	}
	return start, end.AddDate(0, 0, 1), true
}
