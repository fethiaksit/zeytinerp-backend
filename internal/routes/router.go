package routes

import (
	"log"
	"sort"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
	"market-erp-backend/internal/middleware"
)

func SetupRouter(db *gorm.DB, jwtSecret string, corsAllowedOrigins []string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS(corsAllowedOrigins))
	router.NoRoute(handlers.EndpointNotFound)
	router.NoMethod(handlers.EndpointNotFound)

	authHandler := handlers.NewAuthHandler(db, jwtSecret)

	public := router.Group("/")
	public.GET("/health", handlers.Health)

	auth := router.Group("/api/auth")
	auth.POST("/login", authHandler.Login)

	api := router.Group("/api")
	api.Use(middleware.AuthRequired(jwtSecret))

	api.GET("/auth/me", authHandler.Me)
	RegisterDashboardRoutes(api, db)
	RegisterCashReportRoutes(api, db)
	RegisterSupplierRoutes(api, db)
	RegisterSupplierTransactionRoutes(api, db)
	RegisterExpenseRoutes(api, db)
	RegisterIncomeEntryRoutes(api, db)
	RegisterEmployeeRoutes(api, db)
	RegisterEmployeeTransactionRoutes(api, db)
	RegisterFinancialDebtRoutes(api, db)
	RegisterProductRoutes(api, db)
	RegisterStockMovementRoutes(api, db)

	uploads := router.Group("/uploads")
	uploads.Use(middleware.AuthRequired(jwtSecret))
	uploads.GET("/invoices/*filepath", handlers.ServeInvoiceFile)

	return router
}

func LogRoutes(router *gin.Engine) {
	registeredRoutes := router.Routes()
	sort.Slice(registeredRoutes, func(i, j int) bool {
		if registeredRoutes[i].Path == registeredRoutes[j].Path {
			return registeredRoutes[i].Method < registeredRoutes[j].Method
		}
		return registeredRoutes[i].Path < registeredRoutes[j].Path
	})
	for _, route := range registeredRoutes {
		visibility := "PROTECTED"
		if route.Path == "/health" || route.Path == "/api/auth/login" {
			visibility = "PUBLIC"
		}
		log.Printf("ROUTE %-9s %-7s %s", visibility, route.Method, route.Path)
	}
}
