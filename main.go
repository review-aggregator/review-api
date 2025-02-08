package main

import (
	"log"

	"github.com/review-aggregator/review-api/app/config"
	"github.com/review-aggregator/review-api/app/db"
	"github.com/review-aggregator/review-api/app/router"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	err := db.InitDB(cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}

	// Set up the router
	r := router.SetupRouter()

	// Start server
	if err := r.Run(cfg.ServerAddress); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
