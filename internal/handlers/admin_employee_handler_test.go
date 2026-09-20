package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"market-erp-backend/internal/middleware"
	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

func newAdminEmployeeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func setupTestRouter(db *gorm.DB, jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authHandler := NewAuthHandler(db, jwtSecret)

	router.POST("/api/auth/login", authHandler.Login)

	api := router.Group("/api")
	api.Use(middleware.AuthRequired(jwtSecret))

	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())

	h := NewAdminEmployeeHandler(db)
	admin.GET("/employees", h.List)
	admin.POST("/employees", h.Create)
	admin.PUT("/employees/:id", h.Update)
	admin.PATCH("/employees/:id/status", h.UpdateStatus)
	admin.PATCH("/employees/:id/password", h.UpdatePassword)
	admin.DELETE("/employees/:id", h.Delete)

	return router
}

func TestAdminEmployeeCRUD(t *testing.T) {
	db := newAdminEmployeeTestDB(t)
	jwtSecret := "test_secret_key"
	router := setupTestRouter(db, jwtSecret)

	// 1. Seed an admin user to generate admin JWT token
	adminHash, _ := services.HashPassword("admin123")
	adminUser := models.User{
		Username:     "admin",
		PasswordHash: adminHash,
		Role:         "admin",
		IsActive:     true,
	}
	if err := db.Create(&adminUser).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	adminToken, err := services.GenerateJWT(jwtSecret, services.NewAuthClaims(adminUser.ID, adminUser.Username, adminUser.Role))
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	// 2. Test Unauthenticated GET /api/admin/employees -> 401 Unauthorized
	req := httptest.NewRequest(http.MethodGet, "/api/admin/employees", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", resp.Code)
	}

	// 3. Test Non-Admin Role -> 403 Forbidden
	cashierHash, _ := services.HashPassword("cashier123")
	cashierUser := models.User{
		Username:     "cashier1",
		PasswordHash: cashierHash,
		Role:         "cashier",
		IsActive:     true,
	}
	db.Create(&cashierUser)
	cashierToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(cashierUser.ID, cashierUser.Username, cashierUser.Role))

	req = httptest.NewRequest(http.MethodGet, "/api/admin/employees", nil)
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want 403", resp.Code)
	}

	// 4. Test POST /api/admin/employees (Create Employee)
	createPayload := []byte(`{
		"first_name": "Ahmet",
		"last_name": "Yılmaz",
		"phone": "05321112233",
		"username": "ahmet",
		"password": "password123",
		"confirm_password": "password123",
		"role": "cashier",
		"is_active": true
	}`)
	req = httptest.NewRequest(http.MethodPost, "/api/admin/employees", bytes.NewReader(createPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("create employee status = %d, want 201; body: %s", resp.Code, resp.Body.String())
	}

	var createRes struct {
		Success bool                  `json:"success"`
		Data    AdminEmployeeResponse `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &createRes); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	if createRes.Data.Username != "ahmet" || createRes.Data.Role != "cashier" || createRes.Data.FullName != "Ahmet Yılmaz" {
		t.Fatalf("unexpected created user: %#v", createRes.Data)
	}
	if createRes.Data.Phone == nil || *createRes.Data.Phone != "05321112233" {
		t.Fatalf("unexpected phone number: %v", createRes.Data.Phone)
	}

	empID := createRes.Data.ID

	// 5. Test GET /api/admin/employees (List Employees)
	req = httptest.NewRequest(http.MethodGet, "/api/admin/employees", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", resp.Code)
	}

	var listRes struct {
		Success bool                    `json:"success"`
		Data    []AdminEmployeeResponse `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &listRes)
	if len(listRes.Data) < 3 { // admin, cashier1, ahmet
		t.Fatalf("expected at least 3 users, got %d", len(listRes.Data))
	}

	// 6. Test PUT /api/admin/employees/:id (Update Employee)
	updatePayload := []byte(`{
		"first_name": "Ahmet Can",
		"last_name": "Yılmaz",
		"phone": "05321112233",
		"username": "ahmet_can",
		"role": "warehouse"
	}`)
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/employees/%d", empID), bytes.NewReader(updatePayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}

	// 7. Test PATCH /api/admin/employees/:id/status (Deactivate)
	statusPayload := []byte(`{"is_active": false}`)
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/admin/employees/%d/status", empID), bytes.NewReader(statusPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200", resp.Code)
	}

	// Verify inactive user cannot log in
	loginPayload := []byte(`{"username": "ahmet_can", "password": "password123"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("inactive login status = %d, want 401; body: %s", resp.Code, resp.Body.String())
	}

	// 8. Test PATCH /api/admin/employees/:id/password (Change Password & Reactivate)
	statusActivePayload := []byte(`{"is_active": true}`)
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/admin/employees/%d/status", empID), bytes.NewReader(statusActivePayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	passPayload := []byte(`{"password": "newpassword123", "confirm_password": "newpassword123"}`)
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/admin/employees/%d/password", empID), bytes.NewReader(passPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("update password status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}

	// Login with new password & check last_login_at updated
	newLoginPayload := []byte(`{"username": "ahmet_can", "password": "newpassword123"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(newLoginPayload))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("login with new password status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}

	var updatedUser models.User
	db.First(&updatedUser, empID)
	if updatedUser.LastLoginAt == nil || time.Since(*updatedUser.LastLoginAt) > 10*time.Second {
		t.Fatalf("last_login_at was not updated on login: %v", updatedUser.LastLoginAt)
	}

	// 9. Test DELETE /api/admin/employees/:id (Soft Deactivate)
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/employees/%d", empID), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", resp.Code)
	}

	db.First(&updatedUser, empID)
	if updatedUser.IsActive != false {
		t.Fatalf("expected user to be inactive after delete, got active=true")
	}
}
