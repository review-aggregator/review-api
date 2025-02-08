package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/middleware"
	"github.com/review-aggregator/review-api/app/models"
)

func HandlerCreatePlatform(c *gin.Context) {
	var platform models.Platform
	if err := c.ShouldBindJSON(&platform); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	platform.ID = uuid.New()

	if err := models.CreatePlatform(context.Background(), &platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create platform"})
		return
	}
	c.JSON(http.StatusCreated, platform)
}

func HandlerGetPlatformsByProductID(c *gin.Context) {
	productIDStr := c.Param("productID")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	contextUser, err := middleware.GetContextUser(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	products, err := models.GetPlatformsByProductIDAndUserID(context.Background(), productID, contextUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch products"})
		return
	}
	c.JSON(http.StatusOK, products)
}
