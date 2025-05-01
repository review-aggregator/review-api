package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/middleware"
	"github.com/review-aggregator/review-api/app/models"
	"github.com/review-aggregator/review-api/app/services"
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

	services.ScrapeTrustpilot(context.Background(), &platform, "")

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

func HandlerRunPlatformScraper(c *gin.Context) {
	platformIDStr := c.Param("platform_id")
	platformID, err := uuid.Parse(platformIDStr)
	if err != nil {
		fmt.Println("Error while parsing platform id", err)
		c.Status(http.StatusBadRequest)
		return
	}

	platform, err := models.GetPlatformByID(context.Background(), platformID)
	if err != nil {
		fmt.Println("Error while fetching platform", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch platform"})
		return
	}

	latestReviewDate, err := models.GetLatestReviewDateByPlatformID(context.Background(), platformID)
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error while fetching latest review date", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch latest review date"})
		return
	}

	switch platform.Name {
	case "trustpilot":
		fmt.Println("Scraping trustpilot")
		err = services.ScrapeTrustpilot(context.Background(), platform, latestReviewDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not scrape platform"})
			return
		}
	case "tripadvisor":
		fmt.Println("Scraping tripadvisor")
		reviews, err := services.ScrapeTripadvisor(context.Background(), platform, latestReviewDate, 100)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not scrape platform"})
			return
		}
		err = models.CreateReviews(context.Background(), reviews, platform.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create reviews"})
			return
		}
	}

	c.Status(http.StatusOK)
}
