package main

import (
	"context"
	"log"
	"time"

	"market-erp-backend/internal/config"
	"market-erp-backend/internal/db"
	"market-erp-backend/internal/routes"
)

func main() {
	log.Println("config yükleniyor")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	log.Println("DATABASE_URL okundu")

	log.Println("db bağlantısı deneniyor")
	dbCtx, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	database, err := db.Connect(dbCtx, cfg.DatabaseURL)
	cancelDB()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	log.Println("db ping başarılı")

	router := routes.SetupRouter(database)
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
