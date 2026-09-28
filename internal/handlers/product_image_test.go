package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"image"
	"image/png"
	"market-erp-backend/internal/models"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductUpdatePreservesImageAndFavorite(t *testing.T) {
	db := newStockTestDB(t)
	p := models.Product{Name: "Canga", ImageURL: "/api/uploads/products/saved.jpg", IsBestseller: true, BestsellerOrder: 3, IsActive: true}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.PUT("/products/:id", NewProductHandler(db).Update)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PUT", fmt.Sprintf("/products/%d", p.ID), bytes.NewBufferString(`{"name":"Canga","sale_price":25}`)))
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	var got models.Product
	db.First(&got, p.ID)
	if got.ImageURL != p.ImageURL || !got.IsBestseller || got.BestsellerOrder != 3 {
		t.Fatalf("image/favorite lost: %+v", got)
	}
}

func TestProductCreateRejectsInvalidImageInsteadOfSilentSuccess(t *testing.T) {
	db := newStockTestDB(t)
	r := gin.New()
	r.POST("/products", NewProductHandler(db).Create)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/products", bytes.NewBufferString(`{"name":"Canga","image_url":"not-a-url"}`)))
	if w.Code != 400 {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	var count int64
	db.Model(&models.Product{}).Count(&count)
	if count != 0 {
		t.Fatal("failed image must not create product")
	}
}

func TestProductImageSaveReadAndFailedUpdate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PRODUCT_IMAGE_DIR", dir)
	name := strings.Repeat("b", 64) + ".png"
	var imageData bytes.Buffer
	png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	if err := os.WriteFile(filepath.Join(dir, name), imageData.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	path := "/api/uploads/products/" + name
	db := newStockTestDB(t)
	h := NewProductHandler(db)
	r := gin.New()
	r.POST("/products", h.Create)
	r.PUT("/products/:id", h.Update)
	r.GET("/api/uploads/products/:filename", ServeProductImage)
	body, _ := json.Marshal(map[string]interface{}{"name": "Canga", "image_url": path, "is_bestseller": true})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/products", bytes.NewReader(body)))
	if w.Code != 201 {
		t.Fatalf("create %d: %s", w.Code, w.Body)
	}
	var p models.Product
	db.First(&p)
	if p.ImageURL != path || !p.IsBestseller {
		t.Fatalf("not persisted: %+v", p)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), imageData.Bytes()) {
		t.Fatalf("image not served: %d", w.Code)
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PUT", fmt.Sprintf("/products/%d", p.ID), bytes.NewBufferString(`{"name":"Changed","image_url":"broken"}`)))
	if w.Code != 400 {
		t.Fatalf("invalid replacement: %d", w.Code)
	}
	var got models.Product
	db.First(&got, p.ID)
	if got.ImageURL != path || got.Name != "Canga" {
		t.Fatal("failed update modified product")
	}
}

func TestProductFavoriteRejectsMissingImage(t *testing.T) {
	db := newStockTestDB(t)
	p := models.Product{Name: "Canga", IsActive: true}
	db.Create(&p)
	r := gin.New()
	r.PUT("/products/:id/favorite", NewProductHandler(db).ToggleFavorite)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PUT", fmt.Sprintf("/products/%d/favorite", p.ID), bytes.NewBufferString(`{"isFavorite":true}`)))
	if w.Code != 400 {
		t.Fatalf("favorite without image: %d %s", w.Code, w.Body)
	}
}

func TestProductUpdateRejectsFavoriteWithMissingCachedImage(t *testing.T) {
	t.Setenv("PRODUCT_IMAGE_DIR", t.TempDir())
	db := newStockTestDB(t)
	p := models.Product{Name: "Canga", ImageURL: "/api/uploads/products/" + strings.Repeat("c", 64) + ".jpg", IsActive: true}
	db.Create(&p)
	r := gin.New()
	r.PUT("/products/:id", NewProductHandler(db).Update)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("PUT", fmt.Sprintf("/products/%d", p.ID), bytes.NewBufferString(`{"name":"Canga","is_bestseller":true}`)))
	if w.Code != 400 {
		t.Fatalf("missing cached image accepted as favorite: %d %s", w.Code, w.Body)
	}
	var got models.Product
	db.First(&got, p.ID)
	if got.IsBestseller {
		t.Fatal("failed favorite update changed database")
	}
}
