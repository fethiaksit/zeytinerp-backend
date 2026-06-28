package handlers

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name       string
		query      url.Values
		wantStart  string
		wantEnd    string
		wantValid  bool
		wantStatus int
	}{
		{name: "empty range", query: url.Values{}, wantValid: true, wantStatus: 200},
		{name: "full range", query: url.Values{"start_date": {"2026-06-01"}, "end_date": {"2026-06-30"}}, wantStart: "2026-06-01", wantEnd: "2026-06-30", wantValid: true, wantStatus: 200},
		{name: "start only", query: url.Values{"start_date": {"2026-06-01"}}, wantStart: "2026-06-01", wantValid: true, wantStatus: 200},
		{name: "end only", query: url.Values{"end_date": {"2026-06-30"}}, wantEnd: "2026-06-30", wantValid: true, wantStatus: 200},
		{name: "invalid date", query: url.Values{"start_date": {"not-a-date"}}, wantValid: false, wantStatus: 400},
		{name: "reversed range", query: url.Values{"start_date": {"2026-06-30"}, "end_date": {"2026-06-01"}}, wantValid: false, wantStatus: 400},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest("GET", "/?"+tt.query.Encode(), nil)

			dateRange, valid := parseDateRange(context)
			if valid != tt.wantValid {
				t.Fatalf("valid = %v, want %v", valid, tt.wantValid)
			}
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			assertOptionalDate(t, "start", dateRange.start, tt.wantStart)
			assertOptionalDate(t, "end", dateRange.end, tt.wantEnd)
		})
	}
}

func TestApplyDateRangeUsesBetweenForFullRange(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	var expenses []models.Expense
	statement := applyDateRange(db.Model(&models.Expense{}), "expense_date", dateRange{
		start: &start,
		end:   &end,
	}).Find(&expenses).Statement

	if sql := statement.SQL.String(); !strings.Contains(sql, "expense_date BETWEEN $1 AND $2") {
		t.Fatalf("query does not use BETWEEN: %s", sql)
	}
}

func assertOptionalDate(t *testing.T, field string, actual *time.Time, expected string) {
	t.Helper()
	if expected == "" {
		if actual != nil {
			t.Fatalf("%s = %v, want nil", field, actual)
		}
		return
	}
	if actual == nil || actual.Format("2006-01-02") != expected {
		t.Fatalf("%s = %v, want %s", field, actual, expected)
	}
}
