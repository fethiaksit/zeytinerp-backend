package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(ctx context.Context, databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	conn, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	return conn, nil
}

func RunMigrations(ctx context.Context, conn *gorm.DB, dirs ...string) error {
	var files []string
	var selectedDir string
	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
		if err != nil {
			return err
		}
		if len(matches) > 0 {
			files = matches
			selectedDir = dir
			break
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		log.Printf("No migration files found in %v", dirs)
		return nil
	}
	log.Printf("Migration directory: %s", selectedDir)

	for _, file := range files {
		log.Printf("Running migration file: %s", file)

		sql, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}
		if err := conn.WithContext(ctx).Exec(string(sql)).Error; err != nil {
			return fmt.Errorf("migration failed in %s: %w", file, err)
		}

		log.Printf("Migration file completed: %s", file)
	}

	return nil
}
