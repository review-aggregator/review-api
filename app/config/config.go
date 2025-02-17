package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	ServerAddress     string
	DatabaseURL       string
	JWTSecret         string
	InternalAuthToken string
	ClerkPublicKey    string
}

var Config AppConfig

func LoadConfig() *AppConfig {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file", err)
	}

	config := &AppConfig{
		ServerAddress:     getEnv("SERVER_ADDRESS", ""),
		DatabaseURL:       getEnv("DatabaseURL", "user=root password=password dbname=reviews host=localhost port=5432 sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "your_jwt_secret"),
		InternalAuthToken: getEnv("INTERNAL_AUTH_TOKEN", "admin"),
		ClerkPublicKey:    getEnv("CLERK_JWT_PUBLIC_KEY", ""),
	}

	// Check for critical environment variables
	if config.ServerAddress == "" || config.JWTSecret == "" {
		fmt.Println("Error: Required environment variables are not set.")
		return nil // or handle the error as needed
	}

	return config
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		fmt.Printf("Environment variable '%s' exists\n", key)
		return value
	}
	fmt.Printf("Environment variable '%s' does not exist, using fallback\n", key)
	return fallback
}
