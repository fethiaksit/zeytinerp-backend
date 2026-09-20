package routes

import (
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

func TestAdminEmployeeRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", url.QueryEscape(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.AutoMigrate(&models.User{})

	router := SetupRouter(db, "test_secret", []string{"*"})

	routes := router.Routes()
	foundMap := make(map[string]bool)
	for _, r := range routes {
		t.Logf("Registered Route: %s %s", r.Method, r.Path)
		key := fmt.Sprintf("%s %s", r.Method, r.Path)
		foundMap[key] = true
	}

	expectedRouteMap := map[string][]string{
		"/api/admin/employees":          {"GET", "POST"},
		"/api/admin/employees/:id":      {"PUT", "DELETE"},
		"/api/admin/employees/:id/status":   {"PATCH"},
		"/api/admin/employees/:id/password": {"PATCH"},
	}

	for path, methods := range expectedRouteMap {
		for _, m := range methods {
			key := fmt.Sprintf("%s %s", m, path)
			if !foundMap[key] {
				t.Errorf("Expected route %s not found in registered routes!", key)
			}
		}
	}

	// Local HTTP request without token -> must return 401 Unauthorized (not 404 page not found)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/employees", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 Unauthorized; body: %s", resp.Code, resp.Body.String())
	}
}
