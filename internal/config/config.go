package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv             string
	Port               string
	DatabaseURL        string
	JWTSecret          string
	CORSAllowedOrigins []string
}

func Load() Config {
	_ = godotenv.Load(".env", "../.env", "../../.env")

	return Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		Port:               getEnv("PORT", "8081"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		CORSAllowedOrigins: getCSVEnv("CORS_ALLOWED_ORIGINS", "http://zeytinerp.herevemarket.com,https://zeytinerp.herevemarket.com,http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getCSVEnv(key, fallback string) []string {
	value := getEnv(key, fallback)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
