package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type dateRange struct {
	start *time.Time
	end   *time.Time
}

func parseDateRange(c *gin.Context) (dateRange, bool) {
	var result dateRange

	if value := c.Query("start_date"); value != "" {
		date, err := parseDate(value)
		if err != nil {
			fail(c, http.StatusBadRequest, "start_date is invalid")
			return dateRange{}, false
		}
		result.start = &date
	}

	if value := c.Query("end_date"); value != "" {
		date, err := parseDate(value)
		if err != nil {
			fail(c, http.StatusBadRequest, "end_date is invalid")
			return dateRange{}, false
		}
		result.end = &date
	}

	if result.start != nil && result.end != nil && result.start.After(*result.end) {
		fail(c, http.StatusBadRequest, "start_date cannot be after end_date")
		return dateRange{}, false
	}

	return result, true
}

func applyDateRange(query *gorm.DB, column string, dateRange dateRange) *gorm.DB {
	if dateRange.start != nil && dateRange.end != nil {
		return query.Where(column+" BETWEEN ? AND ?", *dateRange.start, *dateRange.end)
	}
	if dateRange.start != nil {
		return query.Where(column+" >= ?", *dateRange.start)
	}
	if dateRange.end != nil {
		return query.Where(column+" <= ?", *dateRange.end)
	}
	return query
}
