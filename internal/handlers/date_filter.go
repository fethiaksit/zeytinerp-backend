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

func parseDateRange(c *gin.Context, startDate, endDate string) (dateRange, bool) {
	var result dateRange

	if startDate != "" {
		date, err := parseDate(startDate)
		if err != nil {
			fail(c, http.StatusBadRequest, "start_date is invalid")
			return dateRange{}, false
		}
		result.start = &date
	}

	if endDate != "" {
		date, err := parseDate(endDate)
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

func applyExpenseDateRange(query *gorm.DB, dateRange dateRange) *gorm.DB {
	return applyDateRange(query, "expenses.expense_date", dateRange)
}

func applyIncomeDateRange(query *gorm.DB, dateRange dateRange) *gorm.DB {
	return applyDateRange(query, "income_entries.income_date", dateRange)
}

func applyDateRange(query *gorm.DB, column string, dateRange dateRange) *gorm.DB {
	if dateRange.start != nil {
		query = query.Where(column+" >= ?", *dateRange.start)
	}
	if dateRange.end != nil {
		query = query.Where(column+" <= ?", *dateRange.end)
	}
	return query
}
