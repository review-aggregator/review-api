package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/consts"
	"github.com/review-aggregator/review-api/app/models"
)

/*
- fetch all products and their platforms
- Run scraper for each platform
- Generate new stats for each platform and time period
*/

func CronRunScraperAndGetStats() error {
	// fetch all products
	products, err := models.GetAllProducts(context.Background())
	if err != nil {
		return fmt.Errorf("error getting products: %w", err)
	}

	for _, product := range products {
		go func() {
			GetProductStatsForAllPlatformsAndTimePeriods(context.Background(), product)
		}()
	}

	return nil
}

func GetProductStatsForAllPlatformsAndTimePeriods(ctx context.Context, product *models.Product) error {
	platforms, err := models.GetPlatformsByProductID(ctx, product.ID)
	if err != nil {
		return fmt.Errorf("error getting platforms: %w", err)
	}

	// No platforms have been added for this product
	if len(platforms) == 0 {
		return fmt.Errorf("no platforms found")
	}

	errChan := make(chan error, 10)

	// Only one platform has been added for this product so we can store the result with platform as "all" in product_stats table
	platformTypes := []PlatformNameWithID{}
	if len(platforms) == 1 {
		var wg sync.WaitGroup
		for _, timePeriod := range consts.TimePeriods {
			wg.Add(1)
			go func() {
				defer wg.Done()
				reviews, err := models.GetReviewsByProductIDAndUserIDAndTimePeriod(ctx, product.ID, product.UserID, timePeriod)
				if err != nil {
					errChan <- fmt.Errorf("error getting reviews: %w", err)
					return
				}
				PrettyPrint(reviews)

				productStats, err := GetProductStats(ctx, reviews, product.Description)
				if err != nil {
					errChan <- fmt.Errorf("error getting product stats: %w", err)
					return
				}

				productStats.ProductID = product.ID
				productStats.Platform = consts.PlatformAll
				productStats.TimePeriod = timePeriod

				PrettyPrint(productStats)

				err = models.CreateProductStats(ctx, productStats)
				if err != nil {
					errChan <- fmt.Errorf("error creating product stats: %w", err)
					return
				}
			}()
		}
		wg.Wait()

		if len(errChan) > 0 {
			return err
		}

		return nil
	} else {
		var wg sync.WaitGroup

		platformTypes = append(platformTypes, PlatformNameWithID{
			PlatformName: "all",
			PlatformID:   uuid.Nil,
		})
		for _, platform := range platforms {
			platformTypes = append(platformTypes, PlatformNameWithID{
				PlatformName: platform.Name,
				PlatformID:   platform.ID,
			})
		}

		for _, platform := range platformTypes {
			for _, timePeriod := range consts.TimePeriods {
				wg.Add(1)
				go func() {
					defer wg.Done()
					reviews, err := models.GetReviewsByPlatformIDAndUserIDAndTimePeriod(ctx, platform.PlatformID, product.UserID, timePeriod)
					if err != nil {
						errChan <- fmt.Errorf("error getting reviews: %w", err)
						return
					}
					PrettyPrint(reviews)

					productStats, err := GetProductStats(ctx, reviews, product.Description)
					if err != nil {
						errChan <- fmt.Errorf("error getting product stats: %w", err)
						return
					}

					productStats.ProductID = product.ID
					productStats.Platform = platform.PlatformName
					productStats.TimePeriod = timePeriod

					PrettyPrint(productStats)

					err = models.CreateProductStats(ctx, productStats)
					if err != nil {
						errChan <- fmt.Errorf("error creating product stats: %w", err)
						return
					}
				}()
			}
		}

		wg.Wait()

		if len(errChan) > 0 {
			return err
		}
	}

	return nil
}
