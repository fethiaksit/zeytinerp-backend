package main

import (
	"context"
	"log"
	"time"

	"market-erp-backend/internal/config"
	"market-erp-backend/internal/db"
	"market-erp-backend/internal/middleware"
	"market-erp-backend/internal/routes"
)

func main() {
	log.Println("config yükleniyor")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	log.Println("DATABASE_URL okundu")
	if cfg.JWTSecret == "" {
		if cfg.AppEnv == "production" {
			log.Fatal("JWT_SECRET is required in production")
		}
		cfg.JWTSecret = "development_jwt_secret_change_me"
		log.Println("JWT_SECRET boş, development secret kullanılacak")
	}
	if cfg.AppEnv == "production" && middleware.HasWildcardOrigin(cfg.CORSAllowedOrigins) {
		log.Fatal("CORS_ALLOWED_ORIGINS cannot contain * in production")
	}
	log.Printf("CORS allowed origins: %v", cfg.CORSAllowedOrigins)

	log.Println("db bağlantısı deneniyor")
	dbCtx, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	database, err := db.Connect(dbCtx, cfg.DatabaseURL)
	cancelDB()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	log.Println("db ping başarılı")

	router := routes.SetupRouter(database, cfg.JWTSecret, cfg.CORSAllowedOrigins)
	routes.LogRoutes(router)

	log.Println("migration başlıyor")
	migrationCtx, cancelMigration := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.RunMigrations(migrationCtx, database, "migrations", "../migrations", "../../migrations"); err != nil {
		cancelMigration()
		log.Fatalf("migration failed: %v", err)
	}
	cancelMigration()
	log.Println("migration bitti")

	log.Printf("server :%s üzerinde başlıyor", cfg.Port)
	log.Printf("Server running on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
