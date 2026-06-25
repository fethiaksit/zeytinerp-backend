package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/services"
)

type ExchangeRateHandler struct{ DB *gorm.DB }

func NewExchangeRateHandler(db *gorm.DB) *ExchangeRateHandler {
	return &ExchangeRateHandler{DB: db}
}

func (h *ExchangeRateHandler) Latest(c *gin.Context) {
	if _, err := services.NormalizeCurrency(c.Query("currency")); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	rate, err := services.LatestRateToTRY(ctx, h.DB, c.Query("currency"))
	if err != nil {
		fail(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	ok(c, rate)
}
