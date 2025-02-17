package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const (
	queryInsertPlatform = `
	INSERT INTO platforms(id, name, url, product_id, created_at, updated_at)
	VALUES(:id, :name, :url, :product_id, NOW(), NOW())`

	queryGetPlatformByID = `
	SELECT p.id, p.name, p.url, p.product_id, p.created_at, p.updated_at
	FROM platforms p
	WHERE p.id = :id`

	queryGetPlatformsByProductIDAndUserID = `
	SELECT p.id, p.name, p.url, p.product_id, p.created_at, p.updated_at
	FROM platforms p
	JOIN products ON products.id = p.product_id
	JOIN users u ON products.user_id = u.id
	WHERE p.product_id = :product_id AND u.id = :user_id`

	queryUpdatePlatformByID = `
	UPDATE platforms SET 
		url = :url,
		updated_at = NOW()
	WHERE id = :platform_id`

	queryGetPlatformByNameAndProductID = `
	SELECT p.id, p.name, p.url, p.product_id, p.created_at, p.updated_at
	FROM platforms p
	WHERE p.name = :name AND p.product_id = :product_id`
)

type Platform struct {
	ID        uuid.UUID `json:"id" db:"id"`
	URL       string    `json:"url" db:"url"`
	Name      string    `json:"name" db:"name"`
	ProductID uuid.UUID `json:"product_id" db:"product_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func CreatePlatform(ctx context.Context, platform *Platform) error {
	_, err := db.NamedExecContext(ctx, queryInsertPlatform, platform)
	if err != nil {
		log.Error("Error while creating platform", err)
		return err
	}

	return nil
}

func GetPlatformByID(ctx context.Context, platformID uuid.UUID) (*Platform, error) {
	var platform Platform

	err := db.NamedGetContext(ctx, &platform, queryGetPlatformByID, map[string]interface{}{
		"id": platformID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No platform found for id: ", platformID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching platform by id", err)
		return nil, err
	}

	return &platform, nil
}

func GetPlatformsByProductIDAndUserID(ctx context.Context, product_id, userID uuid.UUID) ([]*Platform, error) {
	var platform []*Platform

	err := db.NamedSelectContext(ctx, &platform, queryGetPlatformsByProductIDAndUserID, map[string]interface{}{
		"product_id": product_id,
		"user_id":    userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No platforms found for product id: ", product_id)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching platforms by product id", err)
		return nil, err
	}

	return platform, nil
}

func UpdatePlatform(ctx context.Context, platformID uuid.UUID, url string) error {
	_, err := db.NamedExecContext(ctx, queryUpdatePlatformByID, map[string]interface{}{
		"platform_id": platformID,
		"url":         url,
	})
	if err != nil {
		log.Error("Error while updating platform", err)
		return err
	}

	return nil
}

func GetPlatformByNameAndProductID(ctx context.Context, name string, productID uuid.UUID) (*Platform, error) {
	var platform Platform

	err := db.NamedGetContext(ctx, &platform, queryGetPlatformByNameAndProductID, map[string]interface{}{
		"name":       name,
		"product_id": productID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching platform by name and product id", err)
		return nil, err
	}

	return &platform, nil
}
