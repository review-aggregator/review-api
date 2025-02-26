package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/consts"
)

const (
	queryInsertReview = `
	INSERT INTO reviews(id, platform_id, url, author_name, date_published, headline, review_body, rating_value, language, created_at, updated_at)
	VALUES(:id, :platform_id, :url, :author_name, :date_published, :headline, :review_body, :rating_value, :language, NOW(), NOW())
	ON CONFLICT (url) DO NOTHING`

	queryGetReviewByID = `
	SELECT r.id, r.platform_id, r.url, r.author_name, r.date_published, r.headline, r.review_body, r.rating_value, r.language, r.created_at, r.updated_at
	FROM reviews r
	WHERE r.id = :id`

	queryGetReviewsByPlatformID = `
	SELECT r.id, r.platform_id, r.url, r.author_name, r.date_published, r.headline, r.review_body, r.rating_value, r.language, r.created_at, r.updated_at
	FROM reviews r
	WHERE r.platform_id = :platform_id`

	queryGetLatestReviewDateByPlatformID = `
	SELECT r.date_published
	FROM reviews r
	WHERE r.platform_id = :platform_id
	ORDER BY r.date_published DESC
	LIMIT 1`

	queryGetReviewsByProductIDAndUserID = `
	SELECT r.id, r.platform_id, r.url, r.author_name, r.date_published, r.headline, r.review_body, r.rating_value, r.language, r.created_at, r.updated_at
	FROM reviews r
	INNER JOIN platforms p ON p.id = r.platform_id
	INNER JOIN products pr ON pr.id = p.product_id
	WHERE pr.id = :product_id AND pr.user_id = :user_id`

	queryGetReviewsByProductIDAndUserIDAndTimePeriod = `
	SELECT r.id, r.platform_id, r.url, r.author_name, r.date_published, r.headline, r.review_body, r.rating_value, r.language, r.created_at, r.updated_at
	FROM reviews r
	INNER JOIN platforms p ON p.id = r.platform_id
	INNER JOIN products pr ON pr.id = p.product_id
	WHERE pr.id = :product_id AND pr.user_id = :user_id AND r.date_published BETWEEN :date_from AND :date_to`

	queryGetReviewRatings = `
	SELECT 
		CAST(rating_value AS INTEGER) as rating,
		COUNT(*) as count
	FROM reviews r
	INNER JOIN platforms p ON p.id = r.platform_id
	INNER JOIN products pr ON pr.id = p.product_id
	WHERE pr.id = :product_id AND r.date_published BETWEEN :date_from AND :date_to
	GROUP BY CAST(rating_value AS INTEGER)
	ORDER BY rating`
)

type Review struct {
	ID            uuid.UUID `db:"id" json:"id"`
	PlatformID    uuid.UUID `db:"platform_id" json:"platform_id"`
	Url           string    `db:"url" json:"url"`
	AuthorName    string    `db:"author_name" json:"author_name"`
	DatePublished string    `db:"date_published" json:"date_published"`
	Headline      string    `db:"headline" json:"headline"`
	ReviewBody    string    `db:"review_body" json:"review_body"`
	RatingValue   float64   `db:"rating_value" json:"rating_value"`
	Language      string    `db:"language" json:"language"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

type ReviewRating struct {
	Rating int64 `db:"rating" json:"rating"`
	Count  int64 `db:"count" json:"count"`
}

func CreateReview(ctx context.Context, review *Review) error {
	if review.ID == uuid.Nil {
		review.ID = uuid.New()
	}

	_, err := db.NamedExecContext(ctx, queryInsertReview, review)
	if err != nil {
		log.Error("Error while creating review", err)
		return err
	}

	return nil
}

func CreateReviews(ctx context.Context, reviews []*Review, platformID uuid.UUID) error {
	if len(reviews) == 0 {
		return errors.New("reviews cannot be empty")
	}

	for _, review := range reviews {
		review.PlatformID = platformID
		if review.ID == uuid.Nil {
			review.ID = uuid.New()
		}
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

func GetReviewsByPlatformID(ctx context.Context, platformID uuid.UUID) ([]*Review, error) {
	var reviews []*Review

	err := db.NamedSelectContext(ctx, &reviews, queryGetReviewsByPlatformID, map[string]interface{}{
		"platform_id": platformID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No reviews found for platform id: ", platformID)
			return nil, sql.ErrNoRows
		}
		log.Error("Error while fetching reviews by platform id", err)
		return nil, err
	}

	return reviews, nil
}

func GetLatestReviewDateByPlatformID(ctx context.Context, platformID uuid.UUID) (string, error) {
	var reviewDate string

	err := db.NamedGetContext(ctx, &reviewDate, queryGetLatestReviewDateByPlatformID, map[string]interface{}{
		"platform_id": platformID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("No review found for platform id: ", platformID)
			return "", sql.ErrNoRows
		}
		log.Error("Error while fetching latest review by platform id", err)
		return "", err
	}

	return reviewDate, nil
}

func GetReviewsByProductIDAndUserID(ctx context.Context, productID uuid.UUID, userID uuid.UUID) ([]*Review, error) {
	var reviews []*Review

	err := db.NamedSelectContext(ctx, &reviews, queryGetReviewsByProductIDAndUserID, map[string]interface{}{
		"product_id": productID,
		"user_id":    userID,
	})
	if err != nil {
		log.Error("Error while fetching reviews by product id and user id", err)
		return nil, err
	}

	return reviews, nil
}

func GetReviewsByProductIDAndUserIDAndTimePeriod(ctx context.Context, productID uuid.UUID, userID uuid.UUID, timePeriod consts.TimePeriodType) ([]*Review, error) {
	var reviews []*Review

	dateFrom, dateTo := getDateFromAndDateTo(timePeriod)

	err := db.NamedSelectContext(ctx, &reviews, queryGetReviewsByProductIDAndUserIDAndTimePeriod, map[string]interface{}{
		"product_id": productID,
		"user_id":    userID,
		"date_from":  dateFrom,
		"date_to":    dateTo,
	})
	if err != nil {
		log.Error("Error while fetching reviews by product id and user id", err)
		return nil, err
	}

	return reviews, nil
}

func GetReviewRatings(ctx context.Context, productID uuid.UUID, platform consts.PlatformType, timePeriod consts.TimePeriodType) ([]*ReviewRating, error) {
	var reviewRatings []*ReviewRating

	dateFrom, dateTo := getDateFromAndDateTo(timePeriod)

	err := db.NamedSelectContext(ctx, &reviewRatings, queryGetReviewRatings, map[string]interface{}{
		"product_id": productID,
		"platform":   platform,
		"date_from":  dateFrom,
		"date_to":    dateTo,
	})
	if err != nil {
		log.Error("Error while fetching review ratings", err)
		return nil, err
	}

	// Fill in missing ratings with zero counts
	filledRatings := make([]*ReviewRating, 5)
	for i := 0; i < 5; i++ {
		filledRatings[i] = &ReviewRating{Rating: int64(i + 1), Count: 0}
	}

	for _, rating := range reviewRatings {
		if rating.Rating >= 1 && rating.Rating <= 5 {
			filledRatings[rating.Rating-1] = rating
		}
	}

	return filledRatings, nil
}

func getDateFromAndDateTo(timePeriod consts.TimePeriodType) (string, string) {
	var dateFrom time.Time
	var dateTo time.Time
	switch timePeriod {
	case consts.TimePeriodThisWeek:
		dateFrom = time.Now().AddDate(0, 0, -7)
		dateTo = time.Now()
	case consts.TimePeriodLastWeek:
		dateFrom = time.Now().AddDate(0, 0, -14)
		dateTo = time.Now().AddDate(0, 0, -7)
	case consts.TimePeriodThisMonth:
		dateFrom = time.Now().AddDate(0, -1, 0)
		dateTo = time.Now()
	case consts.TimePeriodLastMonth:
		dateFrom = time.Now().AddDate(0, -2, 0)
		dateTo = time.Now().AddDate(0, -1, 0)
	case consts.TimePeriodAllTime:
		dateFrom = time.Now().AddDate(-100, 0, 0)
		dateTo = time.Now()
	}

	return dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02")
}
