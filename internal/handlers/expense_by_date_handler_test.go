package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type expenseByDateTestResponse struct {
	Success  bool             `json:"success"`
	Date     string           `json:"date"`
	Total    decimal.Decimal  `json:"total"`
	Count    int              `json:"count"`
	Expenses []models.Expense `json:"expenses"`
	Error    string           `json:"error"`
}

func TestExpenseListByDate(t *testing.T) {
	db := newDateFilterTestDB(t)
	seedExpensesByDate(t, db)
	router := expenseByDateTestRouter(db)

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantError  string
	}{
		{name: "blank date", query: "?date=%20", wantStatus: http.StatusBadRequest, wantError: "date is required"},
		{name: "invalid date", query: "?date=2026-02-30", wantStatus: http.StatusBadRequest, wantError: "date is invalid; expected format is YYYY-MM-DD"},
		{name: "date with time", query: "?date=2026-07-01T10:00:00Z", wantStatus: http.StatusBadRequest, wantError: "date is invalid; expected format is YYYY-MM-DD"},
		{name: "valid date", query: "?date=2026-07-01", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/expenses/by-date"+tt.query, nil)
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
			}

			var body expenseByDateTestResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if tt.wantStatus != http.StatusOK {
				if body.Success || body.Error != tt.wantError {
					t.Fatalf("error response = {success:%v error:%q}, want {success:false error:%q}", body.Success, body.Error, tt.wantError)
				}
				return
			}

			if !body.Success || body.Date != "2026-07-01" || body.Count != 2 {
				t.Fatalf("response metadata = {success:%v date:%q count:%d}", body.Success, body.Date, body.Count)
			}
			if !body.Total.Equal(decimal.RequireFromString("30.5")) {
				t.Fatalf("total = %s, want 30.5", body.Total)
			}
			if strings.Contains(response.Body.String(), "created_at") || strings.Contains(response.Body.String(), "updated_at") {
				t.Fatalf("response exposes internal timestamp fields: %s", response.Body.String())
			}
			if body.Expenses[0].Note != "late" || body.Expenses[1].Note != "early" {
				t.Fatalf("expense order = [%q, %q], want [late, early]", body.Expenses[0].Note, body.Expenses[1].Note)
			}
		})
	}
}

func TestExpenseListByDateDefaultsToTodayInIstanbul(t *testing.T) {
	db := newDateFilterTestDB(t)
	seedExpensesByDate(t, db)

	gin.SetMode(gin.TestMode)
	handler := NewExpenseHandler(db)
	// June 30 in UTC, but July 1 in Istanbul.
	handler.Now = func() time.Time {
		return time.Date(2026, time.June, 30, 22, 30, 0, 0, time.UTC)
	}
	router := gin.New()
	router.GET("/expenses/by-date", handler.ListByDate)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses/by-date", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var body expenseByDateTestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Date != "2026-07-01" || body.Count != 2 || !body.Total.Equal(decimal.RequireFromString("30.5")) {
		t.Fatalf("response = {date:%q count:%d total:%s}, want {date:%q count:%d total:%s}", body.Date, body.Count, body.Total, "2026-07-01", 2, "30.5")
	}
}

func TestExpenseListByDateReturnsEmptyArray(t *testing.T) {
	router := expenseByDateTestRouter(newDateFilterTestDB(t))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses/by-date?date=2026-07-01", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"total":"0"`) || !strings.Contains(response.Body.String(), `"count":0`) || !strings.Contains(response.Body.String(), `"expenses":[]`) {
		t.Fatalf("unexpected empty response: %s", response.Body.String())
	}
}

func TestExpenseListByDateHandlesDatabaseError(t *testing.T) {
	db := newDateFilterTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	router := expenseByDateTestRouter(db)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/expenses/by-date?date=2026-07-01", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	var body expenseByDateTestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Success || body.Error == "" {
		t.Fatalf("unexpected database error response: %s", response.Body.String())
	}
}

func TestExpensesForDayQueryUsesIndexFriendlyRange(t *testing.T) {
	db := newDateFilterTestDB(t).Session(&gorm.Session{DryRun: true})
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	statement := expensesForDayQuery(db, start, start.AddDate(0, 0, 1)).Find(&[]models.Expense{}).Statement
	sql := strings.ToLower(statement.SQL.String())

	if strings.Contains(sql, "date(") {
		t.Fatalf("query wraps indexed column in DATE(): %s", sql)
	}
	if !strings.Contains(sql, "expense_date >= ?") || !strings.Contains(sql, "expense_date < ?") {
		t.Fatalf("query does not use half-open date range: %s", sql)
	}
}

func expenseByDateTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/expenses/by-date", NewExpenseHandler(db).ListByDate)
	return router
}

func seedExpensesByDate(t *testing.T, db *gorm.DB) {
	t.Helper()
	records := []models.Expense{
		{ExpenseDate: time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC), Category: "diger", Amount: decimal.NewFromInt(40), Note: "previous"},
		{ExpenseDate: time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC), Category: "diger", Amount: decimal.RequireFromString("10.5"), Note: "early"},
		{ExpenseDate: time.Date(2026, 7, 1, 18, 0, 0, 0, time.UTC), Category: "diger", Amount: decimal.NewFromInt(20), Note: "late"},
		{ExpenseDate: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC), Category: "diger", Amount: decimal.NewFromInt(80), Note: "next"},
	}
	for _, record := range records {
		if err := db.Create(&record).Error; err != nil {
			t.Fatalf("seed expense: %v", err)
		}
	}
}
