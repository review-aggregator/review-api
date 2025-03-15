package router

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/review-aggregator/review-api/app/handlers"
	"github.com/review-aggregator/review-api/app/middleware"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow all origins for testing purposes
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests (OPTIONS)
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func SetupRouter() *gin.Engine {
	fmt.Println("Setting up router")
	router := gin.Default()

	router.Use(CORSMiddleware())

	apiRouter := router.Group("/api")

	// User routes group
	userGroup := apiRouter.Group("/users")
	userGroup.POST("/clerk/webhook", handlers.HandlerSignUpClerkWebhook)
	userGroup.Use(middleware.AuthMiddleware())
	userGroup.GET("", handlers.HandlerGetUser)

	// Product routes group (protected)
	productGroup := apiRouter.Group("/product")
	productGroup.GET("/:product_id/generate-stats", handlers.HandlerGenerateProductStats)
	productGroup.GET("/:product_id/stats", handlers.HandlerGetProductStats)
	productGroup.Use(middleware.ClerkMiddleware())
	productGroup.POST("", handlers.HandlerCreateProduct)
	productGroup.GET("", handlers.HandlerGetProducts)
	productGroup.GET("/:product_id", handlers.HandlerGetProductByID)
	productGroup.PUT("/:product_id", handlers.HandlerUpdateProduct)
	productGroup.DELETE("/:product_id", handlers.HandlerDeleteProduct)

	reviewGroup := apiRouter.Group("/review")
	reviewGroup.POST("/formatted", handlers.HandlerGetFormattedReviews)

	internalGroup := apiRouter.Group("internal")
	internalGroup.GET("/platforms/:platform_id/scrape", handlers.HandlerRunPlatformScraper)
	internalGroup.POST("/trustpilot/reviews", handlers.HandlerInsertTrustpilotReviews)
	internalGroup.POST("/product-stats", handlers.HandlerInsertProductStats)

	googleGroup := apiRouter.Group("/google")
	googleGroup.GET("/oauth/url", handlers.HandlerGetGoogleOAuth2URL)
	googleGroup.GET("/oauth/callback", handlers.HandlerGoogleOAuth2Callback)

	return router
}
