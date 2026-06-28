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
	"gorm.io/gorm/logger"

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

			dateRange, valid := parseDateRange(context, context.Query("start_date"), context.Query("end_date"))
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
		buildSQL  func(dateRange) string
		dateRange dateRange
		wantStart bool
		wantEnd   bool
	}{
		{name: "expense start only", column: "expenses.expense_date", buildSQL: func(filter dateRange) string { return expenseListSQL(db, filter) }, dateRange: dateRange{start: &start}, wantStart: true},
		{name: "expense end only", column: "expenses.expense_date", buildSQL: func(filter dateRange) string { return expenseListSQL(db, filter) }, dateRange: dateRange{end: &end}, wantEnd: true},
		{name: "expense full range", column: "expenses.expense_date", buildSQL: func(filter dateRange) string { return expenseListSQL(db, filter) }, dateRange: dateRange{start: &start, end: &end}, wantStart: true, wantEnd: true},
		{name: "income start only", column: "income_entries.income_date", buildSQL: func(filter dateRange) string { return incomeListSQL(db, filter) }, dateRange: dateRange{start: &start}, wantStart: true},
		{name: "income end only", column: "income_entries.income_date", buildSQL: func(filter dateRange) string { return incomeListSQL(db, filter) }, dateRange: dateRange{end: &end}, wantEnd: true},
		{name: "income full range", column: "income_entries.income_date", buildSQL: func(filter dateRange) string { return incomeListSQL(db, filter) }, dateRange: dateRange{start: &start, end: &end}, wantStart: true, wantEnd: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.buildSQL(tt.dateRange)

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

	tests := []struct {
		name     string
		column   string
		listSQL  string
		totalSQL string
	}{
		{name: "expenses", column: "expenses.expense_date", listSQL: expenseListSQL(db, dateRange), totalSQL: expenseTotalSQL(db, dateRange)},
		{name: "incomes", column: "income_entries.income_date", listSQL: incomeListSQL(db, dateRange), totalSQL: incomeTotalSQL(db, dateRange)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("list SQL: %s", tt.listSQL)
			t.Logf("total SQL: %s", tt.totalSQL)

			if strings.Contains(tt.listSQL, "SUM(") {
				t.Fatalf("list query contains aggregate: %s", tt.listSQL)
			}
			if !strings.Contains(tt.listSQL, "ORDER BY") {
				t.Fatalf("list query lost its ordering: %s", tt.listSQL)
			}
			if !strings.Contains(tt.totalSQL, "SUM(amount)") {
				t.Fatalf("total query does not contain aggregate: %s", tt.totalSQL)
			}
			if strings.Contains(tt.totalSQL, "ORDER BY") {
				t.Fatalf("total query contains list ordering: %s", tt.totalSQL)
			}

			for queryName, sql := range map[string]string{"list": tt.listSQL, "total": tt.totalSQL} {
				if !strings.Contains(sql, tt.column+" >=") {
					t.Fatalf("%s query does not contain start condition: %s", queryName, sql)
				}
				if !strings.Contains(sql, tt.column+" <=") {
					t.Fatalf("%s query does not contain end condition: %s", queryName, sql)
				}
			}
		})
	}
}

func TestDateFilterArgumentOrder(t *testing.T) {
	db := newDryRunDB(t)
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	filter := dateRange{start: &start, end: &end}

	queries := []struct {
		name      string
		statement *gorm.Statement
	}{
		{name: "expenses", statement: applyExpenseDateRange(db.Model(&models.Expense{}), filter).Find(&[]models.Expense{}).Statement},
		{name: "incomes", statement: applyIncomeDateRange(db.Model(&models.IncomeEntry{}), filter).Find(&[]models.IncomeEntry{}).Statement},
	}

	for _, query := range queries {
		t.Run(query.name, func(t *testing.T) {
			if len(query.statement.Vars) != 2 {
				t.Fatalf("argument count = %d, want 2", len(query.statement.Vars))
			}
			if query.statement.Vars[0] != start {
				t.Fatalf("first argument = %v, want start date %v", query.statement.Vars[0], start)
			}
			if query.statement.Vars[1] != end {
				t.Fatalf("second argument = %v, want end date %v", query.statement.Vars[1], end)
			}
		})
	}
}

func expenseListSQL(db *gorm.DB, dateRange dateRange) string {
	var expenses []models.Expense
	return applyExpenseDateRange(db.Model(&models.Expense{}), dateRange).
		Order("expense_date desc, id desc").
		Find(&expenses).Statement.SQL.String()
}

func expenseTotalSQL(db *gorm.DB, dateRange dateRange) string {
	var total decimal.Decimal
	return applyExpenseDateRange(db.Model(&models.Expense{}), dateRange).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Statement.SQL.String()
}

func incomeListSQL(db *gorm.DB, dateRange dateRange) string {
	var entries []models.IncomeEntry
	return applyIncomeDateRange(db.Model(&models.IncomeEntry{}), dateRange).
		Order("income_date desc, id desc").
		Find(&entries).Statement.SQL.String()
}

func incomeTotalSQL(db *gorm.DB, dateRange dateRange) string {
	var total decimal.Decimal
	return applyIncomeDateRange(db.Model(&models.IncomeEntry{}), dateRange).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Statement.SQL.String()
}

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
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
