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
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"market-erp-backend/internal/middleware"
	"market-erp-backend/internal/models"
	"market-erp-backend/internal/services"
)

func newStockTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Product{}, &models.StockMovement{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	return db
}

func setupStockTestRouter(db *gorm.DB, jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	api := router.Group("/api")
	api.Use(middleware.AuthRequired(jwtSecret))

	handler := NewStockMovementHandler(db)

	api.GET("/stock-movements", handler.List)
	api.POST("/stock-movements", middleware.RequireRole("admin", "warehouse"), handler.Create)
	api.POST("/stock-movements/bulk", middleware.RequireRole("admin", "warehouse"), handler.BulkCreate)

	admin := api.Group("/admin")
	{
		admin.GET("/stock-movements", handler.List)
		admin.POST("/stock-movements", middleware.RequireRole("admin", "warehouse"), handler.Create)
		admin.POST("/stock-movements/bulk", middleware.RequireRole("admin", "warehouse"), handler.BulkCreate)
	}

	return router
}

func TestStockMovementEndpoints(t *testing.T) {
	db := newStockTestDB(t)
	jwtSecret := "test_secret_key"
	router := setupStockTestRouter(db, jwtSecret)

	// Seed product
	barcode := "8691234567890"
	product := models.Product{
		Name:          "Zeytin Yağı 1L",
		Barcode:       &barcode,
		Category:      "Gıda",
		PurchasePrice: decimal.NewFromFloat(100.0),
		SalePrice:     decimal.NewFromFloat(150.0),
		IsActive:      true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	// Create users & tokens
	adminUser := models.User{Username: "admin", Role: "admin", IsActive: true}
	warehouseUser := models.User{Username: "depo", Role: "warehouse", IsActive: true}
	cashierUser := models.User{Username: "kasiyer", Role: "cashier", IsActive: true}
	db.Create(&adminUser)
	db.Create(&warehouseUser)
	db.Create(&cashierUser)

	adminToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(adminUser.ID, adminUser.Username, adminUser.Role))
	warehouseToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(warehouseUser.ID, warehouseUser.Username, warehouseUser.Role))
	cashierToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(cashierUser.ID, cashierUser.Username, cashierUser.Role))

	// 1. Test Cashier POST /api/stock-movements -> 403 Forbidden
	payload := []byte(fmt.Sprintf(`{
		"product_id": %d,
		"quantity": 10,
		"type": "in",
		"note": "Kasiyer stok eklemeyi deniyor"
	}`, product.ID))

	req := httptest.NewRequest(http.MethodPost, "/api/stock-movements", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusForbidden {
		t.Fatalf("cashier status = %d, want 403", resp.Code)
	}

	// 2. Test Admin POST /api/stock-movements (snake_case)
	req = httptest.NewRequest(http.MethodPost, "/api/stock-movements", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("admin status = %d, want 201; body: %s", resp.Code, resp.Body.String())
	}

	var createRes struct {
		Success bool                 `json:"success"`
		Data    models.StockMovement `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &createRes); err != nil {
		t.Fatalf("unmarshal create res: %v", err)
	}
	if createRes.Data.ProductID != product.ID || createRes.Data.Type != "in" || createRes.Data.Product.Name != "Zeytin Yağı 1L" {
		t.Fatalf("unexpected movement output: %#v", createRes.Data)
	}

	// 3. Test Warehouse POST /api/stock-movements (camelCase & STOCK_IN normalization)
	camelPayload := []byte(fmt.Sprintf(`{
		"productId": %d,
		"quantity": 5,
		"movementType": "STOCK_IN",
		"note": "Depocu stok girişi"
	}`, product.ID))

	req = httptest.NewRequest(http.MethodPost, "/api/stock-movements", bytes.NewReader(camelPayload))
	req.Header.Set("Authorization", "Bearer "+warehouseToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("warehouse status = %d, want 201; body: %s", resp.Code, resp.Body.String())
	}

	// 4. Test POST /api/stock-movements/bulk
	bulkPayload := []byte(fmt.Sprintf(`{
		"entries": [
			{"productId": %d, "quantity": 15, "movementType": "STOCK_IN", "note": "Toplu 1"},
			{"product_id": %d, "quantity": 5, "type": "out", "note": "Toplu 2"}
		]
	}`, product.ID, product.ID))

	req = httptest.NewRequest(http.MethodPost, "/api/stock-movements/bulk", bytes.NewReader(bulkPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("bulk create status = %d, want 201; body: %s", resp.Code, resp.Body.String())
	}

	// 5. Test GET /api/stock-movements?product_id=...
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/stock-movements?product_id=%d", product.ID), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("get list status = %d, want 200", resp.Code)
	}

	var listRes struct {
		Success bool                   `json:"success"`
		Data    []models.StockMovement `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &listRes)
	if len(listRes.Data) != 4 {
		t.Fatalf("expected 4 stock movements, got %d", len(listRes.Data))
	}
}
