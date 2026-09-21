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
	txHandler := NewCustomerTransactionHandler(db)

	api.GET("/customers", custHandler.List)
	api.GET("/customers/:id", custHandler.Get)
	api.GET("/customers/:id/balance", custHandler.Balance)

	api.GET("/customers/:id/transactions", txHandler.List)
	api.POST("/customers/:id/transactions", txHandler.Create)

	admin := api.Group("/customers")
	admin.Use(middleware.RequireAdmin())
	admin.POST("", custHandler.Create)
	admin.PUT("/:id", custHandler.Update)
	admin.DELETE("/:id", custHandler.Delete)

	return router
}

func TestCariCustomerFullRules(t *testing.T) {
	db := newCustomerTestDB(t)
	jwtSecret := "test_secret_key"
	router := setupCustomerTestRouter(db, jwtSecret)

	adminUser := models.User{Username: "admin", Role: "admin", IsActive: true}
	cashierUser := models.User{Username: "kasiyer", Role: "cashier", IsActive: true}
	db.Create(&adminUser)
	db.Create(&cashierUser)

	adminToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(adminUser.ID, adminUser.Username, adminUser.Role))
	cashierToken, _ := services.GenerateJWT(jwtSecret, services.NewAuthClaims(cashierUser.ID, cashierUser.Username, cashierUser.Role))

	// 1. Admin cari müşteri oluşturabilir (customer_type = "cari")
	createPayload := []byte(`{"name": "Ahmet Yılmaz", "phone": "05321112233", "credit_limit": 5000.00}`)
	req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(createPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("1. admin create status = %d, want 201; body: %s", resp.Code, resp.Body.String())
	}

	var createRes struct {
		Success bool             `json:"success"`
		Data    CustomerResponse `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &createRes)
	custID := createRes.Data.ID

	if createRes.Data.CustomerType != "cari" {
		t.Fatalf("1. expected customer_type = 'cari', got %q", createRes.Data.CustomerType)
	}

	// 2. Cashier cari müşteri oluşturamaz -> 403
	req = httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(createPayload))
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("2. cashier create status = %d, want 403", resp.Code)
	}

	// 3. Aynı telefon ikinci kez eklenemez (farklı format ile dene: +905321112233)
	duplicatePayload := []byte(`{"name": "Ahmet Klon", "phone": "+905321112233"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(duplicatePayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("3. duplicate phone create status = %d, want 400", resp.Code)
	}

	// 4. Negatif credit_limit reddedilir
	negativeLimitPayload := []byte(`{"name": "Mehmet", "phone": "05339998877", "credit_limit": -100}`)
	req = httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(negativeLimitPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("4. negative credit_limit status = %d, want 400", resp.Code)
	}

	// 5. Debt bakiyeyi artırır (POST /api/customers/:id/transactions)
	debtPayload := []byte(`{"type": "debt", "amount": 1250.50, "note": "Market alışverişi"}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/customers/%d/transactions", custID), bytes.NewReader(debtPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("5. debt tx status = %d, want 201; body: %s", resp.Code, resp.Body.String())
	}

	// Bakiye kontrol et -> 1250.50
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/customers/%d/balance", custID), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("5. balance get status = %d, want 200", resp.Code)
	}

	var balanceRes struct {
		Success bool `json:"success"`
		Data    struct {
			CustomerID      uint            `json:"customer_id"`
			DebtTotal       decimal.Decimal `json:"debt_total"`
			PaymentTotal    decimal.Decimal `json:"payment_total"`
			Balance         decimal.Decimal `json:"balance"`
			LimitExceeded   bool            `json:"limit_exceeded"`
			AvailableCredit decimal.Decimal `json:"available_credit"`
		} `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &balanceRes)
	if !balanceRes.Data.Balance.Equal(decimal.NewFromFloat(1250.50)) {
		t.Fatalf("5. balance after debt = %s, want 1250.50", balanceRes.Data.Balance.String())
	}

	// 6. Payment bakiyeyi azaltır
	paymentPayload := []byte(`{"type": "payment", "amount": 500.00, "note": "Nakit ödeme"}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/customers/%d/transactions", custID), bytes.NewReader(paymentPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("6. payment tx status = %d, want 201", resp.Code)
	}

	// Bakiye kontrol et -> 1250.50 - 500 = 750.50
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/customers/%d/balance", custID), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	json.Unmarshal(resp.Body.Bytes(), &balanceRes)
	if !balanceRes.Data.Balance.Equal(decimal.NewFromFloat(750.50)) {
		t.Fatalf("6. balance after payment = %s, want 750.50", balanceRes.Data.Balance.String())
	}
	if balanceRes.Data.DebtTotal.Equal(decimal.NewFromFloat(1250.50)) == false || balanceRes.Data.PaymentTotal.Equal(decimal.NewFromFloat(500.00)) == false {
		t.Fatalf("6. debt_total or payment_total mismatch: debt=%s, pay=%s", balanceRes.Data.DebtTotal, balanceRes.Data.PaymentTotal)
	}

	// 7. Negatif/0 hareket reddedilir
	zeroPayload := []byte(`{"type": "debt", "amount": 0}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/customers/%d/transactions", custID), bytes.NewReader(zeroPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("7. zero amount tx status = %d, want 400", resp.Code)
	}

	// 8. Invalid transaction type reddedilir
	invalidTypePayload := []byte(`{"type": "refund", "amount": 100}`)
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/customers/%d/transactions", custID), bytes.NewReader(invalidTypePayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("8. invalid type tx status = %d, want 400", resp.Code)
	}

	// Pasif müşteri oluştur
	inactivePayload := []byte(`{"name": "Pasif Müşteri", "phone": "05441112233", "is_active": false}`)
	req = httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(inactivePayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// 9. Pasif müşteri listede admin tarafından görülebilir, cashier göremez
	req = httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var cashierList struct {
		Data []CustomerResponse `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &cashierList)
	if len(cashierList.Data) != 1 {
		t.Fatalf("9. cashier expected 1 active customer, got %d", len(cashierList.Data))
	}

	req = httptest.NewRequest(http.MethodGet, "/api/customers", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var adminList struct {
		Data []CustomerResponse `json:"data"`
	}
	json.Unmarshal(resp.Body.Bytes(), &adminList)
	if len(adminList.Data) != 2 {
		t.Fatalf("9. admin expected 2 customers (active+inactive), got %d", len(adminList.Data))
	}

	// 10. Mobil normal kullanıcı cari bilgisine erişemez (bilinmeyen/kayıtsız veya cari olmayan)
	req = httptest.NewRequest(http.MethodGet, "/api/mobile/customer/profile?phone=05550000000", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var mobileNormalRes struct {
		HasCurrentAccount bool `json:"has_current_account"`
		HasCustomer       bool `json:"has_customer"`
	}
	json.Unmarshal(resp.Body.Bytes(), &mobileNormalRes)
	if mobileNormalRes.HasCurrentAccount {
		t.Fatalf("10. non-customer mobile expected has_current_account = false, got true")
	}

	// 11. Telefon eşleşen aktif cari müşteri erişebilir
	req = httptest.NewRequest(http.MethodGet, "/api/mobile/customer/profile?phone=5321112233", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	var mobileCariRes struct {
		HasCurrentAccount bool `json:"has_current_account"`
		Customer          *struct {
			Name    string          `json:"name"`
			Balance decimal.Decimal `json:"balance"`
		} `json:"customer"`
	}
	json.Unmarshal(resp.Body.Bytes(), &mobileCariRes)
	if !mobileCariRes.HasCurrentAccount || mobileCariRes.Customer == nil || mobileCariRes.Customer.Name != "Ahmet Yılmaz" {
		t.Fatalf("11. matching mobile customer expected has_current_account = true, got %#v", mobileCariRes)
	}

	// 12. Sonradan cari açılan mobil kullanıcı erişebilir
	mobilePhone := "05077778899"
	// İlk kontrol -> false
	req = httptest.NewRequest(http.MethodGet, "/api/mobile/customer/profile?phone="+mobilePhone, nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	json.Unmarshal(resp.Body.Bytes(), &mobileNormalRes)
	if mobileNormalRes.HasCurrentAccount {
		t.Fatalf("12. before admin opens cari: expected has_current_account = false")
	}

	// Admin sonradan aynı telefonla cari hesap açar
	laterPayload := []byte(`{"name": "Sonradan Eklenen Müşteri", "phone": "05077778899"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(laterPayload))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("12. admin create later customer status = %d, want 201", resp.Code)
	}

	// Mobil kullanıcı tekrar sorgular -> true
	req = httptest.NewRequest(http.MethodGet, "/api/mobile/customer/profile?phone="+mobilePhone, nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	json.Unmarshal(resp.Body.Bytes(), &mobileCariRes)
	if !mobileCariRes.HasCurrentAccount || mobileCariRes.Customer == nil || mobileCariRes.Customer.Name != "Sonradan Eklenen Müşteri" {
		t.Fatalf("12. after admin opens cari: expected has_current_account = true, got %#v", mobileCariRes)
	}
}
