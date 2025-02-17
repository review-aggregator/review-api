package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	queryInsertReview = `
	INSERT INTO reviews(id, product_id, url, author_name, date_published, headline, review_body, rating_value, language, created_at, updated_at)
	VALUES(:id, :product_id, :url, :author_name, :date_published, :headline, :review_body, :rating_value, :language, NOW(), NOW())`

	queryGetReviewByID = `
	SELECT
		r.id,
		r.product_id,
		r.url,
		r.author_name,
		r.date_published,
		r.headline,
		r.review_body,
		r.rating_value,
		r.language,
		r.created_at,
		r.updated_at
	FROM
		reviews r
	WHERE
		r.id = :id`

	queryGetReviewsByProductID = `
	SELECT
		r.id,
		r.product_id, 
		r.url,
		r.author_name,
		r.date_published,
		r.headline,
		r.review_body,
		r.rating_value,
		r.language,
		r.created_at,
		r.updated_at
	FROM
		reviews r
	WHERE
		r.product_id = :product_id`
)

type Review struct {
	ID            uuid.UUID `db:"id" json:"id"`
	ProductID     uuid.UUID `db:"product_id" json:"product_id"`
	Url           string    `db:"url" json:"url"`
	AuthorName    string    `db:"author_name" json:"author_name"`
	DatePublished time.Time `db:"date_published" json:"date_published"`
	Headline      string    `db:"headline" json:"headline"`
	ReviewBody    string    `db:"review_body" json:"review_body"`
	RatingValue   float64   `db:"rating_value" json:"rating_value"`
	Language      string    `db:"language" json:"language"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

func CreateReview(ctx context.Context, review *Review) error {
	_, err := db.NamedExecContext(ctx, queryInsertReview, review)
	if err != nil {
		log.Error("Error while creating review", err)
		return err
	}

	return nil
}

func CreateReviews(ctx context.Context, reviews []*Review) error {
	if len(reviews) == 0 {
		return errors.New("reviews cannot be empty")
	}

	_, err := db.NamedExecContext(ctx, queryInsertReview, reviews)
	if err != nil {
		log.Error("Error while creating reviews", err)
		return err
	}

	return nil
}

func GetReviewByID(ctx context.Context, reviewID uuid.UUID) (*Review, error) {
	var review Review

	err := db.NamedGetContext(ctx, &review, queryGetReviewByID, map[string]interface{}{
		"id": reviewID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No review found for id: ", reviewID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching review by id", err)
		return nil, err
	}

	return &review, nil
}

func GetReviewsByProductID(ctx context.Context, productID uuid.UUID) ([]*Review, error) {
	var reviews []*Review

	err := db.NamedSelectContext(ctx, &reviews, queryGetReviewsByProductID, map[string]interface{}{
		"product_id": productID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No reviews found for product id: ", productID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching reviews by product id", err)
		return nil, err
	}

	return reviews, nil
}
