package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Response struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message,omitempty"`
	Data        interface{}      `json:"data,omitempty"`
	TotalAmount *decimal.Decimal `json:"total_amount,omitempty"`
	Error       string           `json:"error,omitempty"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

func okWithTotalAmount(c *gin.Context, data interface{}, total decimal.Decimal) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data, TotalAmount: &total})
}

func created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Success: false, Error: message})
}

func EndpointNotFound(c *gin.Context) {
	fail(c, http.StatusNotFound, "endpoint not found")
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		fail(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return uint(id), true
}

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("date is required")
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, value)
}

func positiveDecimal(value decimal.Decimal, field string) error {
	if value.LessThanOrEqual(decimal.Zero) {
		return errors.New(field + " must be greater than zero")
	}
	return nil
}

func notNegativeDecimal(value decimal.Decimal, field string) error {
	if value.IsNegative() {
		return errors.New(field + " cannot be negative")
	}
	return nil
}

func validateType(value string, allowed map[string]bool) bool {
	return allowed[value]
}

func handleDBError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, "record not found")
		return
	}
	fail(c, http.StatusInternalServerError, err.Error())
}
