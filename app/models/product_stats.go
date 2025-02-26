package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/review-aggregator/review-api/app/consts"
)

const (
	queryUpsertProductStats = `
	INSERT INTO product_stats (product_id, platform, time_period, key_highlights, pain_points, overall_sentiment)
	VALUES (:product_id, :platform, :time_period, CAST(:key_highlights AS text[]), CAST(:pain_points AS text[]), :overall_sentiment)
	ON CONFLICT (product_id, platform, time_period) DO UPDATE
	SET key_highlights = CAST(:key_highlights AS text[]),
		pain_points = CAST(:pain_points AS text[]),
		overall_sentiment = :overall_sentiment,
		updated_at = CURRENT_TIMESTAMP
	`

	queryGetProductStats = `
	SELECT * FROM product_stats
	WHERE product_id = :product_id
	AND platform = :platform
	AND time_period = :time_period
	`
)

type ProductStats struct {
	ProductID        uuid.UUID             `json:"product_id" db:"product_id"`
	Platform         consts.PlatformType   `json:"platform" db:"platform"`
	TimePeriod       consts.TimePeriodType `json:"time_period" db:"time_period"`
	KeyHighlights    pq.StringArray        `json:"key_highlights" db:"key_highlights"`
	PainPoints       pq.StringArray        `json:"pain_points" db:"pain_points"`
	OverallSentiment string                `json:"overall_sentiment" db:"overall_sentiment"`
	CreatedAt        time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at" db:"updated_at"`
}

func CreateProductStats(ctx context.Context, productStats *ProductStats) error {
	productStats.KeyHighlights = pq.StringArray(productStats.KeyHighlights)
	productStats.PainPoints = pq.StringArray(productStats.PainPoints)
	_, err := db.NamedExecContext(ctx, queryUpsertProductStats, productStats)
	if err != nil {
		log.Error("Error while upserting product stats", err)
		return err
	}

	return nil
}

func GetProductStats(ctx context.Context, productID uuid.UUID, platform consts.PlatformType, timePeriod consts.TimePeriodType) (*ProductStats, error) {
	productStats := &ProductStats{}
	err := db.NamedGetContext(ctx, productStats, queryGetProductStats, map[string]interface{}{
		"product_id":  productID,
		"platform":    platform,
		"time_period": timePeriod,
	})
	if err != nil {
		log.Error("Error while getting product stats", err)
		return nil, err
	}

	return productStats, nil
}
