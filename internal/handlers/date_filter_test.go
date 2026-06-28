package handlers

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
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

func TestApplyDateRangeUsesIndependentBoundaries(t *testing.T) {
	db := newDryRunDB(t)
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		column    string
		dateRange dateRange
		wantStart bool
		wantEnd   bool
	}{
		{name: "expense start only", column: "expense_date", dateRange: dateRange{start: &start}, wantStart: true},
		{name: "expense end only", column: "expense_date", dateRange: dateRange{end: &end}, wantEnd: true},
		{name: "expense full range", column: "expense_date", dateRange: dateRange{start: &start, end: &end}, wantStart: true, wantEnd: true},
		{name: "income start only", column: "income_date", dateRange: dateRange{start: &start}, wantStart: true},
		{name: "income end only", column: "income_date", dateRange: dateRange{end: &end}, wantEnd: true},
		{name: "income full range", column: "income_date", dateRange: dateRange{start: &start, end: &end}, wantStart: true, wantEnd: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var expenses []models.Expense
			sql := applyDateRange(db.Model(&models.Expense{}), tt.column, tt.dateRange).
				Find(&expenses).Statement.SQL.String()

			if got := strings.Contains(sql, tt.column+" >="); got != tt.wantStart {
				t.Fatalf("start condition presence = %v, want %v: %s", got, tt.wantStart, sql)
			}
			if got := strings.Contains(sql, tt.column+" <="); got != tt.wantEnd {
				t.Fatalf("end condition presence = %v, want %v: %s", got, tt.wantEnd, sql)
			}
			if strings.Contains(sql, "BETWEEN") {
				t.Fatalf("query unexpectedly uses BETWEEN: %s", sql)
			}
		})
	}
}

func TestListAndTotalQueriesAreIndependent(t *testing.T) {
	db := newDryRunDB(t)
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	dateRange := dateRange{start: &start, end: &end}

	var expenses []models.Expense
	listStatement := applyDateRange(db.Model(&models.Expense{}), "expense_date", dateRange).
		Order("expense_date desc, id desc").
		Find(&expenses).Statement

	var total decimal.Decimal
	totalStatement := applyDateRange(db.Model(&models.Expense{}), "expense_date", dateRange).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Statement

	listSQL := listStatement.SQL.String()
	if strings.Contains(listSQL, "SUM(") {
		t.Fatalf("list query contains aggregate: %s", listSQL)
	}
	if !strings.Contains(listSQL, "ORDER BY expense_date desc, id desc") {
		t.Fatalf("list query lost its ordering: %s", listSQL)
	}

	totalSQL := totalStatement.SQL.String()
	if !strings.Contains(totalSQL, "SUM(amount)") {
		t.Fatalf("total query does not contain aggregate: %s", totalSQL)
	}
	if strings.Contains(totalSQL, "ORDER BY") {
		t.Fatalf("total query contains list ordering: %s", totalSQL)
	}

	for name, sql := range map[string]string{"list": listSQL, "total": totalSQL} {
		if !strings.Contains(sql, "expense_date >=") {
			t.Fatalf("%s query does not contain start condition: %s", name, sql)
		}
		if !strings.Contains(sql, "expense_date <=") {
			t.Fatalf("%s query does not contain end condition: %s", name, sql)
		}
	}
}

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
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
