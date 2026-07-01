package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"market-erp-backend/internal/models"
)

type supplierResponse struct {
	Success bool              `json:"success"`
	Data    []models.Supplier `json:"data"`
	Error   string            `json:"error"`
}

func TestSupplierCreateNormalizesVisitDays(t *testing.T) {
	db := newSupplierHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/suppliers", NewSupplierHandler(db).Create)

	body := []byte(`{"name":"Test Firma","visit_days":["Perşembe","monday","Pazartesi"]}`)
	request := httptest.NewRequest(http.MethodPost, "/suppliers", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var result struct {
		Success bool            `json:"success"`
		Data    models.Supplier `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := []string{"monday", "thursday"}
	if !result.Success || !reflect.DeepEqual(result.Data.VisitDays, want) {
		t.Fatalf("visit_days = %#v, want %#v", result.Data.VisitDays, want)
	}
}

func TestSupplierUpdateReplacesVisitDays(t *testing.T) {
	db := newSupplierHandlerTestDB(t)
	supplier := models.Supplier{Name: "Test Firma", VisitDays: []string{"monday"}, IsActive: true}
	if err := db.Create(&supplier).Error; err != nil {
		t.Fatalf("create supplier: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/suppliers/:id", NewSupplierHandler(db).Update)
	body := []byte(`{"name":"Test Firma","visit_days":["wednesday","friday"]}`)
	request := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/suppliers/%d", supplier.ID), bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var updated models.Supplier
	if err := db.First(&updated, supplier.ID).Error; err != nil {
		t.Fatalf("reload supplier: %v", err)
	}
	want := []string{"wednesday", "friday"}
	if !reflect.DeepEqual(updated.VisitDays, want) {
		t.Fatalf("visit_days = %#v, want %#v", updated.VisitDays, want)
	}
}

func TestSupplierVisitsFiltersDayAndInactiveSuppliers(t *testing.T) {
	db := newSupplierHandlerTestDB(t)
	seedSupplierHandlerRecords(t, db)

	gin.SetMode(gin.TestMode)
	handler := NewSupplierHandler(db)
	router := gin.New()
	router.GET("/firms/visits", handler.Visits)

	request := httptest.NewRequest(http.MethodGet, "/firms/visits?day=pazartesi", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	result := decodeSupplierListResponse(t, response, http.StatusOK)
	if len(result.Data) != 1 || result.Data[0].Name != "Monday Active" {
		t.Fatalf("suppliers = %#v, want only Monday Active", result.Data)
	}
}

func TestSupplierTodayVisitsUsesIstanbulDate(t *testing.T) {
	db := newSupplierHandlerTestDB(t)
	seedSupplierHandlerRecords(t, db)

	gin.SetMode(gin.TestMode)
	handler := NewSupplierHandler(db)
	// Sunday in UTC, but Monday in Istanbul.
	handler.Now = func() time.Time {
		return time.Date(2026, time.July, 5, 22, 30, 0, 0, time.UTC)
	}
	router := gin.New()
	router.GET("/firms/today-visits", handler.TodayVisits)

	request := httptest.NewRequest(http.MethodGet, "/firms/today-visits", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	result := decodeSupplierListResponse(t, response, http.StatusOK)
	if len(result.Data) != 1 || result.Data[0].Name != "Monday Active" {
		t.Fatalf("suppliers = %#v, want only Monday Active", result.Data)
	}
}

func TestSupplierVisitsRejectsInvalidDay(t *testing.T) {
	db := newSupplierHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/firms/visits", NewSupplierHandler(db).Visits)

	request := httptest.NewRequest(http.MethodGet, "/firms/visits?day=funday", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	result := decodeSupplierListResponse(t, response, http.StatusBadRequest)
	if result.Success || result.Error != "day is invalid" {
		t.Fatalf("response = %#v, want day is invalid", result)
	}
}

func newSupplierHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Supplier{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

func seedSupplierHandlerRecords(t *testing.T, db *gorm.DB) {
	t.Helper()
	suppliers := []models.Supplier{
		{Name: "Monday Active", VisitDays: []string{"monday"}, IsActive: true},
		{Name: "Monday Inactive", VisitDays: []string{"monday"}, IsActive: false},
		{Name: "Tuesday Active", VisitDays: []string{"tuesday"}, IsActive: true},
	}
	if err := db.Create(&suppliers).Error; err != nil {
		t.Fatalf("seed suppliers: %v", err)
	}
	if err := db.Model(&models.Supplier{}).
		Where("name = ?", "Monday Inactive").
		Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate supplier: %v", err)
	}
}

func decodeSupplierListResponse(t *testing.T, response *httptest.ResponseRecorder, wantStatus int) supplierResponse {
	t.Helper()
	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, wantStatus, response.Body.String())
	}
	var result supplierResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}
