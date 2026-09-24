package routes

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

	"market-erp-backend/internal/models"
)

func TestProductFavoriteEndpointAddsAndRemovesFavorite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.Product{}, &models.StockMovement{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	product := models.Product{Name: "Canga", IsActive: true}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}

	router := gin.New()
	api := router.Group("/api")
	RegisterProductRoutes(api, db)

	assertFavoriteRequest(t, router, product.ID, true, http.StatusOK)

	var updated models.Product
	if err := db.First(&updated, product.ID).Error; err != nil {
		t.Fatalf("reload favorite product: %v", err)
	}
	if !updated.IsBestseller || updated.BestsellerOrder != 1 {
		t.Fatalf("favorite state = (%v, %d), want (true, 1)", updated.IsBestseller, updated.BestsellerOrder)
	}

	assertFavoriteRequest(t, router, product.ID, false, http.StatusOK)
	if err := db.First(&updated, product.ID).Error; err != nil {
		t.Fatalf("reload unfavorited product: %v", err)
	}
	if updated.IsBestseller || updated.BestsellerOrder != 0 {
		t.Fatalf("favorite state = (%v, %d), want (false, 0)", updated.IsBestseller, updated.BestsellerOrder)
	}
}

func TestProductFavoriteEndpointRejectsEleventhFavorite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&models.Product{}, &models.StockMovement{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	for order := 1; order <= 10; order++ {
		favorite := models.Product{
			Name:            fmt.Sprintf("Favori %d", order),
			IsActive:        true,
			IsBestseller:    true,
			BestsellerOrder: order,
		}
		if err := db.Create(&favorite).Error; err != nil {
			t.Fatalf("create favorite %d: %v", order, err)
		}
	}

	target := models.Product{Name: "On Birinci Ürün", IsActive: true}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("create target product: %v", err)
	}

	router := gin.New()
	api := router.Group("/api")
	RegisterProductRoutes(api, db)

	assertFavoriteRequest(t, router, target.ID, true, http.StatusConflict)

	if err := db.First(&target, target.ID).Error; err != nil {
		t.Fatalf("reload target product: %v", err)
	}
	if target.IsBestseller || target.BestsellerOrder != 0 {
		t.Fatalf("rejected product changed to favorite: (%v, %d)", target.IsBestseller, target.BestsellerOrder)
	}
}

func assertFavoriteRequest(t *testing.T, router http.Handler, productID uint, favorite bool, wantStatus int) {
	t.Helper()
	body, err := json.Marshal(map[string]bool{"isFavorite": favorite})
	if err != nil {
		t.Fatalf("marshal favorite request: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/products/%d/favorite", productID),
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != wantStatus {
		t.Fatalf("favorite status = %d, want %d; body: %s", response.Code, wantStatus, response.Body.String())
	}
}
