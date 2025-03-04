package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/consts"
	"github.com/review-aggregator/review-api/app/models"
)

type InsertTrustpilotReviewsBody struct {
	PlatformID uuid.UUID        `json:"platform_id"`
	Reviews    []*models.Review `json:"reviews"`
}

func HandlerInsertTrustpilotReviews(c *gin.Context) {
	var body InsertTrustpilotReviewsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fmt.Println("Error while binding json", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if len(body.Reviews) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No reviews provided"})
		return
	}

	platform, err := models.GetPlatformByID(context.Background(), body.PlatformID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Platform not found"})
		return
	}
	if err != nil {
		fmt.Println("Error while getting platform", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get platform"})
		return
	}

	if platform.Name != "trustpilot" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Platform is not trustpilot"})
		return
	}

	if err := models.CreateReviews(context.Background(), body.Reviews, platform.ID); err != nil {
		fmt.Println("Error while inserting reviews", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not insert reviews"})
		return
	}

	c.Status(http.StatusCreated)
}

func HandlerGetReviews(c *gin.Context) {
	reviews, err := models.GetReviews(context.Background())
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No reviews found for product"})
		return
	}
	if err != nil {
		fmt.Println("Error while getting reviews", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get reviews"})
		return
	}

	c.JSON(http.StatusOK, reviews)
}

type GetFormattedReviewsBody struct {
	PlatformID uuid.UUID             `json:"platform_id"`
	UserID     uuid.UUID             `json:"user_id"`
	TimePeriod consts.TimePeriodType `json:"time_period"`
}

type GetFormattedReviewsResponse struct {
	ReviewBody string `json:"review_body"`
	Rating     int64  `json:"rating"`
}

func HandlerGetFormattedReviews(c *gin.Context) {
	var body GetFormattedReviewsBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fmt.Println("Error while binding json", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	reviews, err := models.GetReviewsByPlatformIDAndUserIDAndTimePeriod(context.Background(), body.PlatformID, body.UserID, body.TimePeriod)
	if err != nil {
		fmt.Println("Error getting reviews: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get reviews"})
		return
	}

	formattedReviews := make([]GetFormattedReviewsResponse, 0)
	for _, review := range reviews {
		formattedReviews = append(formattedReviews, GetFormattedReviewsResponse{
			ReviewBody: review.ReviewBody,
			Rating:     int64(review.RatingValue),
		})
	}
	// formattedReviews := services.FormatReviewsForPrompt(reviews, "sentiment")

	c.JSON(http.StatusOK, gin.H{"reviews": formattedReviews})
}
