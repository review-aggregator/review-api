package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/google/uuid"
)

const (
	queryInsertProduct = `
	INSERT INTO products(id, user_id, name, description, created_at, updated_at)
	VALUES(:id, :user_id, :name, :description, NOW(), NOW())`

	queryGetProductByID = `
	SELECT
		p.id,
		p.user_id,
		p.name,
		p.description,
		p.created_at,
		p.updated_at
	FROM products p
	WHERE p.id = :id`

	queryGetProductByIDAndUserID = `
	SELECT
		p.id,
		p.user_id,
		p.name,
		p.description,
		p.created_at,
		p.updated_at
	FROM products p
	WHERE p.id = :id AND p.user_id = :user_id`

	queryGetProductsByUserID = `
	SELECT
		p.id,
		p.name,
		p.description,
		p.created_at,
		p.updated_at
	FROM
		products p
	WHERE
		p.user_id = :user_id`

	queryUpdateProductByID = `
		UPDATE products SET %s
		WHERE id = :id
		RETURNING *`
)

type Product struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
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

func GetProductsByUserID(ctx context.Context, userID uuid.UUID) ([]*Product, error) {
	var product []*Product

	err := db.NamedSelectContext(ctx, &product, queryGetProductsByUserID, map[string]interface{}{
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

func UpdateProduct(ctx context.Context, product *Product) error {
	productMap := structs.Map(product)
	query := fmt.Sprintf(queryUpdateProductByID, db.GetFormattedColumnNames(db.GetStringMapKeys(productMap), "id"))

	err := db.NamedExecContextReturnObj(ctx, query, productMap, product)
	if err != nil {
		log.Error("Error while updating product", err)
		return err
	}

	return nil
}
