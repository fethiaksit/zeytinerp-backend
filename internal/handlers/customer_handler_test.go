package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"market-erp-backend/internal/middleware"
	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

func newCustomerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Customer{}, &models.CustomerTransaction{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func setupCustomerTestRouter(db *gorm.DB, jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mobileHandler := NewMobileHandler(db)
	mobile := router.Group("/api/mobile")
	mobile.GET("/customer/profile", mobileHandler.GetCustomerProfile)
	mobile.GET("/customer/transactions", mobileHandler.GetCustomerTransactions)

	api := router.Group("/api")
	api.Use(middleware.AuthRequired(jwtSecret))

	custHandler := NewCustomerHandler(db)
	api.GET("/customers", custHandler.List)
	api.GET("/customers/:id", custHandler.Get)

	admin := api.Group("/customers")
	admin.Use(middleware.RequireAdmin())
	admin.POST("", custHandler.Create)
	admin.PUT("/:id", custHandler.Update)
	admin.DELETE("/:id", custHandler.Delete)

	return router
}

func TestCustomerPermissionsAndMobileAccess(t *testing.T) {
	db := newCustomerTestDB(t)
	jwtSecret := "test_secret_key"
	router := setupCustomerTestRouter(db, jwtSecret)

	// Admin and Cashier tokens
	adminUser := models.User{Username: "admin", Role: "admin", IsActive: true}
	cashierUser := models.User{Username: "kasiyer", Role: "cashier", IsActive: true}
	db.Create(&adminUser)
	db.Create(&cashierUser)

	adminToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(adminUser.ID, adminUser.Username, adminUser.Role))
	cashierToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(cashierUser.ID, cashierUser.Username, cashierUser.Role))

	// 1. Cashier attempts to create customer -> 403 Forbidden
	createPayload := []byte(`{"name": "Test Müşteri", "phone": "05321112233"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(createPayload))
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("cashier create status = %d, want 403 Forbidden", resp.Code)
	}

	// 2. Admin creates customer -> 201 Created
	req = httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(createPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("admin create status = %d, want 201 Created; body: %s", resp.Code, resp.Body.String())
	}

	var createRes struct {
		Success bool             `json:"success"`
		Data    CustomerResponse `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &createRes)
	custID := createRes.Data.ID

	// 3. Admin creates an inactive customer
	inactivePayload := []byte(`{"name": "Pasif Müşteri", "phone": "05449998877", "is_active": false}`)
	req = httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(inactivePayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create inactive status = %d, want 201", resp.Code)
	}

	// 4. Cashier lists customers -> should only get active customers (1 active, 0 inactive)
	req = httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("cashier list status = %d, want 200", resp.Code)
	}

	var listRes struct {
		Success bool               `json:"success"`
		Data    []CustomerResponse `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &listRes)

	if len(listRes.Data) != 1 || listRes.Data[0].Name != "Test Müşteri" {
		t.Fatalf("expected 1 active customer for cashier, got %d", len(listRes.Data))
	}

	// 5. Admin lists customers -> can see all customers
	req = httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	json.Unmarshal(resp.Body.Bytes(), &listRes)
	if len(listRes.Data) != 2 {
		t.Fatalf("expected 2 customers for admin, got %d", len(listRes.Data))
	}

	// 6. Cashier attempts update/delete -> 403 Forbidden
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/customers/%d", custID), nil)
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("cashier delete status = %d, want 403", resp.Code)
	}

	// 7. Mobile Customer Check - matching phone -> has_customer = true
	req = httptest.NewRequest(http.MethodGet, "/api/mobile/customer/profile?phone=05321112233", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("mobile profile status = %d, want 200", resp.Code)
	}

	var mobileRes struct {
		HasCustomer bool `json:"has_customer"`
		Customer    *struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"customer"`
	}
	json.Unmarshal(resp.Body.Bytes(), &mobileRes)

	if !mobileRes.HasCustomer || mobileRes.Customer == nil || mobileRes.Customer.Name != "Test Müşteri" {
		t.Fatalf("unexpected mobile profile response: %#v", mobileRes)
	}

	// 8. Mobile Customer Check - non-matching phone -> has_customer = false
	req = httptest.NewRequest(http.MethodGet, "/api/mobile/customer/profile?phone=05000000000", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	json.Unmarshal(resp.Body.Bytes(), &mobileRes)
	if mobileRes.HasCustomer {
		t.Fatalf("expected has_customer = false for non-matching phone, got true")
	}
}
