package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type dateFilteredListResponse struct {
	Success     bool              `json:"success"`
	Data        []json.RawMessage `json:"data"`
	TotalAmount decimal.Decimal   `json:"total_amount"`
	Error       string            `json:"error"`
}

func TestExpenseAndIncomeListDateFilters(t *testing.T) {
	db := newDateFilterTestDB(t)
	seedDateFilterRecords(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/expenses", NewExpenseHandler(db).List)
	router.GET("/income-entries", NewIncomeEntryHandler(db).List)

	tests := []struct {
		name       string
		query      url.Values
		wantIDs    []uint
		wantTotal  string
		wantStatus int
		wantError  string
	}{
		{
			name:       "start date only",
			query:      url.Values{"start_date": {"2026-06-01"}},
			wantIDs:    []uint{4, 3, 2},
			wantTotal:  "90",
			wantStatus: http.StatusOK,
		},
		{
			name:       "end date only",
			query:      url.Values{"end_date": {"2026-06-30"}},
			wantIDs:    []uint{3, 2, 1},
			wantTotal:  "60",
			wantStatus: http.StatusOK,
		},
		{
			name:       "inclusive date range",
			query:      url.Values{"start_date": {"2026-06-01"}, "end_date": {"2026-06-30"}},
			wantIDs:    []uint{3, 2},
			wantTotal:  "50",
			wantStatus: http.StatusOK,
		},
		{
			name:       "reversed date range",
			query:      url.Values{"start_date": {"2026-06-30"}, "end_date": {"2026-06-01"}},
			wantStatus: http.StatusBadRequest,
			wantError:  "start_date cannot be after end_date",
		},
	}

	for _, endpoint := range []string{"/expenses", "/income-entries"} {
		for _, tt := range tests {
			t.Run(endpoint+"/"+tt.name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, endpoint+"?"+tt.query.Encode(), nil)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)

				if response.Code != tt.wantStatus {
					t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.wantStatus, response.Body.String())
				}

				var body dateFilteredListResponse
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if tt.wantStatus == http.StatusBadRequest {
					if body.Success || body.Error != tt.wantError {
						t.Fatalf("error response = {success:%v error:%q}, want {success:false error:%q}", body.Success, body.Error, tt.wantError)
					}
					return
				}

				if !body.Success {
					t.Fatalf("success = false; error: %s", body.Error)
				}
				if !body.TotalAmount.Equal(decimal.RequireFromString(tt.wantTotal)) {
					t.Fatalf("total_amount = %s, want %s", body.TotalAmount, tt.wantTotal)
				}
				assertResponseIDs(t, body.Data, tt.wantIDs)
			})
		}
	}
}

func newDateFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Expense{}, &models.IncomeEntry{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func seedDateFilterRecords(t *testing.T, db *gorm.DB) {
	t.Helper()
	dates := []time.Time{
		mustTestDate(t, "2026-05-31"),
		mustTestDate(t, "2026-06-01"),
		mustTestDate(t, "2026-06-30"),
		mustTestDate(t, "2026-07-01"),
	}

	for index, date := range dates {
		amount := decimal.NewFromInt(int64((index + 1) * 10))
		expense := models.Expense{ExpenseDate: date, Category: "diger", Amount: amount}
		if err := db.Create(&expense).Error; err != nil {
			t.Fatalf("seed expense: %v", err)
		}
		income := models.IncomeEntry{IncomeDate: date, Category: "diger", Amount: amount}
		if err := db.Create(&income).Error; err != nil {
			t.Fatalf("seed income entry: %v", err)
		}
	}
}

func mustTestDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

func assertResponseIDs(t *testing.T, records []json.RawMessage, expected []uint) {
	t.Helper()
	if len(records) != len(expected) {
		t.Fatalf("record count = %d, want %d", len(records), len(expected))
	}
	for index, record := range records {
		var value struct {
			ID uint `json:"id"`
		}
		if err := json.Unmarshal(record, &value); err != nil {
			t.Fatalf("decode record %d: %v", index, err)
		}
		if value.ID != expected[index] {
			t.Fatalf("record %d id = %d, want %d", index, value.ID, expected[index])
		}
	}
}
