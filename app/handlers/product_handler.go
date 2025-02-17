package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/middleware"
	"github.com/review-aggregator/review-api/app/models"
)

type CreateProductBody struct {
	Name        string `json:"name" validate:"min=3,max=20"`
	Description string `json:"description" validate:"min=1"`
	Platform    string `json:"platform" validate:"oneof=trustpilot amazon"`
	ProductURL  string `json:"product_url" validate:"required,url"`
}

func HandlerCreateProduct(c *gin.Context) {
	contextUser, err := middleware.GetContextUser(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var body CreateProductBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Initialize validator
	validate := validator.New()

	// Perform validation
	if err := validate.Struct(body); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		c.JSON(http.StatusBadRequest, gin.H{"error": validationErrors.Error()})
		return
	}

	// URL platform validation
	switch body.Platform {
	case "trustpilot":
		if !strings.Contains(body.ProductURL, "trustpilot.com") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product URL must be from Trustpilot"})
			return
		}
	case "amazon":
		if !strings.Contains(body.ProductURL, "amazon.") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product URL must be from Amazon"})
			return
		}
	}

	productExists, err := models.GetProductByNameAndUserID(context.Background(), body.Name, contextUser.ID)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch product"})
		return
	}

	if productExists != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product already exists"})
		return
	}

	product := models.Product{
		ID:          uuid.New(),
		Name:        body.Name,
		Description: body.Description,
		UserID:      contextUser.ID,
	}

	if err := models.CreateProduct(context.Background(), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create product"})
		return
	}

	platform := models.Platform{
		ID:        uuid.New(),
		Name:      body.Platform,
		URL:       body.ProductURL,
		ProductID: product.ID,
	}

	if err := models.CreatePlatform(context.Background(), &platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create platform"})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func HandlerGetProducts(c *gin.Context) {
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

func HandlerGetProductByID(c *gin.Context) {
	contextUser, err := middleware.GetContextUser(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	productID := c.Param("product_id")
	product, err := models.GetProductByIDAndUserID(context.Background(), uuid.MustParse(productID), contextUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch product"})
		return
	}

	platforms, err := models.GetPlatformsByProductIDAndUserID(context.Background(), uuid.MustParse(productID), contextUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch platforms"})
		return
	}

	product.Platforms = platforms

	c.JSON(http.StatusOK, product)
}

func HandlerDeleteProduct(c *gin.Context) {
	contextUser, err := middleware.GetContextUser(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	productID := c.Param("product_id")
	err = models.DeleteProduct(context.Background(), uuid.MustParse(productID), contextUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete product"})
		return
	}

	c.Status(http.StatusNoContent)
}

type UpdatePlatformBody struct {
	Name string `json:"name" validate:"oneof=trustpilot amazon"`
	URL  string `json:"url" validate:"required,url"`
}

type UpdateProductBody struct {
	Name        string               `json:"name" validate:"min=3,max=20"`
	Description string               `json:"description" validate:"min=1"`
	Platforms   []UpdatePlatformBody `json:"platforms"`
}

func HandlerUpdateProduct(c *gin.Context) {
	contextUser, err := middleware.GetContextUser(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	productID := c.Param("product_id")
	product, err := models.GetProductByIDAndUserID(context.Background(), uuid.MustParse(productID), contextUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch product"})
		return
	}

	var body UpdateProductBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	product.Name = body.Name
	product.Description = body.Description

	err = models.UpdateProduct(context.Background(), product.ID, body.Name, body.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update product"})
		return
	}

	for _, platform := range body.Platforms {
		existingPlatform, err := models.GetPlatformByNameAndProductID(context.Background(), platform.Name, product.ID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch platform"})
			return
		}

		if err == sql.ErrNoRows {
			existingPlatform = &models.Platform{
				ID:        uuid.New(),
				Name:      platform.Name,
				URL:       platform.URL,
				ProductID: product.ID,
			}
			err = models.CreatePlatform(context.Background(), existingPlatform)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create platform"})
				return
			}
		} else {
			existingPlatform.URL = platform.URL
			err = models.UpdatePlatform(context.Background(), existingPlatform.ID, platform.URL)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update platform"})
				return
			}
		}
	}

	c.Status(http.StatusOK)
}
