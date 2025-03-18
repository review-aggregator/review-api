package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const (
	queryInsertProduct = `
	INSERT INTO products(id, user_id, name, description, created_at, updated_at)
	VALUES(:id, :user_id, :name, :description, NOW(), NOW())`

	queryGetProductByID = `
	SELECT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at
	FROM products p
	WHERE p.id = :id`

	queryGetProductByIDAndUserID = `
	SELECT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at
	FROM products p
	WHERE p.id = :id AND p.user_id = :user_id`

	queryGetProductsByUserID = `
	SELECT p.id, p.name, p.description, p.created_at, p.updated_at
	FROM products p
	WHERE p.is_deleted = FALSE AND p.user_id = :user_id`

	queryUpdateProductByID = `
	UPDATE products
	SET name = :name, description = :description, updated_at = NOW()
	WHERE id = :product_id`

	queryGetProductByNameAndUserID = `
	SELECT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at
	FROM products p
	WHERE p.is_deleted = FALSE AND p.name = :name AND p.user_id = :user_id`

	queryDeleteProduct = `
	UPDATE products
	SET is_deleted = TRUE
	WHERE id = :id AND user_id = :user_id`

	queryGetAllProducts = `
	SELECT p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at
	FROM products p`

	queryGetProductsWithReviewStats = `
	SELECT 
		p.id, 
		p.user_id, 
		p.name, 
		p.description, 
		p.created_at, 
		p.updated_at,
		COUNT(DISTINCT r.id) as review_count,
		ROUND(COALESCE(AVG(r.rating_value), 0), 2) as average_rating
	FROM products p
	LEFT JOIN platforms plt ON plt.product_id = p.id
	LEFT JOIN reviews r ON r.platform_id = plt.id
	WHERE p.is_deleted = FALSE AND p.user_id = :user_id
	GROUP BY p.id, p.user_id, p.name, p.description, p.created_at, p.updated_at`
)

type Product struct {
	ID          uuid.UUID   `json:"id" db:"id"`
	UserID      uuid.UUID   `json:"user_id" db:"user_id"`
	Name        string      `json:"name" db:"name"`
	Description string      `json:"description" db:"description"`
	Platforms   []*Platform `json:"platforms" db:"platforms"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

type ProductWithReviewStats struct {
	Product
	ReviewCount   int     `json:"review_count" db:"review_count"`
	AverageRating float64 `json:"average_rating" db:"average_rating"`
}

func CreateProduct(ctx context.Context, product *Product) error {
	_, err := db.NamedExecContext(ctx, queryInsertProduct, product)
	if err != nil {
		log.Error("Error while creating product", err)
		return err
	}

	return nil
}

// GetProductByID should be used only for internal purposes
func GetProductByID(ctx context.Context, productID uuid.UUID) (*Product, error) {
	var product Product

	err := db.NamedGetContext(ctx, &product, queryGetProductByID, map[string]interface{}{
		"id": productID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No product found for id: ", productID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching product by id", err)
		return nil, err
	}

	return &product, nil
}

func GetProductByIDAndUserID(ctx context.Context, productID, userID uuid.UUID) (*Product, error) {
	var product Product

	err := db.NamedGetContext(ctx, &product, queryGetProductByIDAndUserID, map[string]interface{}{
		"id":      productID,
		"user_id": userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No product found for id: ", productID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching product by id", err)
		return nil, err
	}

	return &product, nil
}

func GetProductsByUserID(ctx context.Context, userID uuid.UUID) ([]*ProductWithReviewStats, error) {
	var product []*ProductWithReviewStats

	err := db.NamedSelectContext(ctx, &product, queryGetProductsWithReviewStats, map[string]interface{}{
		"user_id": userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No product found for user id: ", userID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching product by id", err)
		return nil, err
	}

	return product, nil
}

func GetProductByNameAndUserID(ctx context.Context, name string, userID uuid.UUID) (*Product, error) {
	var product Product

	err := db.NamedGetContext(ctx, &product, queryGetProductByNameAndUserID, map[string]interface{}{
		"name":    name,
		"user_id": userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No product found for name: ", name)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching product by name and user ID", err)
		return nil, err
	}

	return &product, nil
}

func UpdateProduct(ctx context.Context, productID uuid.UUID, name, description string) error {
	_, err := db.NamedExecContext(ctx, queryUpdateProductByID, map[string]interface{}{
		"product_id":  productID,
		"name":        name,
		"description": description,
	})
	if err != nil {
		log.Error("Error while updating product", err)
		return err
	}

	return nil
}

func DeleteProduct(ctx context.Context, productID, userID uuid.UUID) error {
	_, err := db.NamedExecContext(ctx, queryDeleteProduct, map[string]interface{}{
		"id":      productID,
		"user_id": userID,
	})
	if err != nil {
		log.Error("Error while deleting product", err)
		return err
	}

	return nil
}

func GetAllProducts(ctx context.Context) ([]*Product, error) {
	var product []*Product

	err := db.NamedSelectContext(ctx, &product, queryGetAllProducts, map[string]interface{}{})
	if err != nil {
		log.Error("Error while fetching all products ", err)
		return nil, err
	}

	return product, nil
}
