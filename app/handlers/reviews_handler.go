package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/review-aggregator/review-api/app/models"
)

type InsertReviewsBody struct {
	Reviews []*models.Review `json:"reviews"`
}

func HandlerInsertReviews(c *gin.Context) {
	var body InsertReviewsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if err := models.CreateReviews(context.Background(), body.Reviews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not insert reviews"})
		return
	}

	c.Status(http.StatusCreated)
}
