package router

import (
	"github.com/gin-gonic/gin"
	"github.com/review-aggregator/review-api/app/handlers"
	"github.com/review-aggregator/review-api/app/middleware"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// User routes group
	userGroup := router.Group("/users")
	userGroup.POST("/signup", handlers.HandlerSignUp)
	userGroup.POST("/login", handlers.HandlerLogin)

	// Product routes group (protected)
	productGroup := router.Group("/products")
	productGroup.Use(middleware.AuthMiddleware())
	productGroup.GET("/", handlers.HandlerGetProducts)
	productGroup.POST("/", handlers.HandlerCreateProduct)

	internalGroup := router.Group("internal")
	internalGroup.POST("/reviews", handlers.HandlerInsertReviews)

	return router
}
