package config

import (
	"os"
)

type AppConfig struct {
	ServerAddress     string
	DatabaseURL       string
	JWTSecret         string
	InternalAuthToken string
}

var Config AppConfig

func LoadConfig() *AppConfig {
	return &AppConfig{
		ServerAddress:     getEnv("SERVER_ADDRESS", ":8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "user:password@tcp(localhost:3306)/dbname"),
		JWTSecret:         getEnv("JWT_SECRET", "your_jwt_secret"),
		InternalAuthToken: getEnv("INTERNAL_AUTH_TOKEN", "admin"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
