package routes

import (
	"log"
	"sort"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"market-erp-backend/internal/handlers"
	"market-erp-backend/internal/middleware"
)

func SetupRouter(db *gorm.DB, jwtSecret string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	router.NoRoute(handlers.EndpointNotFound)
	router.NoMethod(handlers.EndpointNotFound)

	router.GET("/health", handlers.Health)

	api := router.Group("/api")
	authHandler := handlers.NewAuthHandler(db, jwtSecret)
	api.POST("/auth/login", authHandler.Login)

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
		log.Printf("ROUTE %-7s %s", route.Method, route.Path)
	}
}
