package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/middleware"
	"github.com/review-aggregator/review-api/app/models"
)

func HandlerCreateProduct(c *gin.Context) {
	contextUser, err := middleware.GetContextUser(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	product.ID = uuid.New()
	product.UserID = contextUser.ID

	if err := models.CreateProduct(context.Background(), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create product"})
		return
	}
	c.JSON(http.StatusCreated, product)
}

func HandlerGetProducts(c *gin.Context) {
	// userID, _ := c.Get("user_id").(string)
	contextUser, err := middleware.GetContextUser(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	products, err := models.GetProductsByUserID(context.Background(), contextUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch products"})
		return
	}
	c.JSON(http.StatusOK, products)
}
