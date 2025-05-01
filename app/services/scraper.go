package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/models"
)

type TripAdvisorResponse struct {
	Data struct {
		Locations []struct {
			LocationID     int    `json:"locationId"`
			Name           string `json:"name"`
			ReviewListPage struct {
				TotalCount int `json:"totalCount"`
				Reviews    []struct {
					ID            string `json:"id"`
					Text          string `json:"text"`
					Title         string `json:"title"`
					Rating        int    `json:"rating"`
					CreatedDate   string `json:"createdDate"`
					PublishedDate string `json:"publishedDate"`
					Username      string `json:"username"`
					UserProfile   struct {
						DisplayName string `json:"displayName"`
						Avatar      struct {
							PhotoSizeDynamic struct {
								URLTemplate string `json:"urlTemplate"`
							} `json:"photoSizeDynamic"`
						} `json:"avatar"`
					} `json:"userProfile"`
					MgmtResponse *struct {
						Text          string `json:"text"`
						PublishedDate string `json:"publishedDate"`
						Username      string `json:"username"`
					} `json:"mgmtResponse"`
				} `json:"reviews"`
			} `json:"reviewListPage"`
		} `json:"locations"`
	} `json:"data"`
}

func ScrapeTrustpilot(ctx context.Context, platform *models.Platform, latestReviewDate string) error {
	// Create HTTP client
	client := &http.Client{}

	// Create request body
	requestBody := map[string]string{
		"platform_id":      platform.ID.String(),
		"platform_url":     platform.URL,
		"last_review_date": latestReviewDate,
	}

	fmt.Println("Request body", requestBody)

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %v", err)
	}

	// Create new request
	req, err := http.NewRequestWithContext(ctx, "POST", os.Getenv("SCRAPER_URL")+"/scrape/trustpilot", bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("Error while creating request", err)
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while sending request", err)
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func ScrapeTripadvisor(ctx context.Context, platform *models.Platform, latestReviewDate string, scrapeReviewsCount int) ([]*models.Review, error) {
	client := &http.Client{}
	locationID := extractLocationID(platform.URL)
	limit := 20 // TripAdvisor's default limit
	offset := 0
	allReviews := make([]*models.Review, 0)

	for {
		// Break if we've collected enough reviews
		if len(allReviews) >= scrapeReviewsCount {
			break
		}

		// Adjust limit for last batch if needed
		remainingCount := scrapeReviewsCount - len(allReviews)
		if remainingCount < limit {
			limit = remainingCount
		}

		requestBody := []map[string]interface{}{
			{
				"variables": map[string]interface{}{
					"locationId": locationID,
					"offset":     offset,
					"limit":      limit,
					"language":   "en",
					"filters": []map[string]interface{}{
						{
							"axis":       "LANGUAGE",
							"selections": []string{"en"},
						},
						{
							"axis":       "SORT",
							"selections": []string{"mostRecent"},
						},
					},
					"prefs": map[string]interface{}{
						"showMT":   true,
						"sortBy":   "DATE",
						"sortType": "",
					},
				},
				"extensions": map[string]interface{}{
					"preRegisteredQueryId": "aaff0337570ed0aa",
				},
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://www.tripadvisor.in/data/graphql/ids", bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		// Add any required cookies or headers
		req.Header.Set("Cookie", "YOUR_COOKIE_HERE") // You'll need to handle this securely

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var response []TripAdvisorResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		if len(response) == 0 || len(response[0].Data.Locations) == 0 {
			break
		}

		location := response[0].Data.Locations[0]
		reviews := location.ReviewListPage.Reviews
		totalCount := location.ReviewListPage.TotalCount

		// Convert to review model
		for _, review := range reviews {
			publishedDate, _ := time.Parse("2006-01-02", review.PublishedDate)

			// Check if we've reached reviews older than latestReviewDate
			if latestReviewDate != "" {
				lastReviewDate, _ := time.Parse(time.RFC3339, latestReviewDate)
				if publishedDate.Before(lastReviewDate) {
					return allReviews, nil
				}
			}

			allReviews = append(allReviews, &models.Review{
				ID:            uuid.New(),
				AuthorName:    review.UserProfile.DisplayName,
				RatingValue:   float64(review.Rating),
				ReviewBody:    review.Text,
				DatePublished: publishedDate.Format(time.RFC3339),
			})

			// Check if we've reached the desired count
			if len(allReviews) >= scrapeReviewsCount {
				return allReviews, nil
			}
		}

		// Check if we've fetched all available reviews
		if offset+limit >= totalCount {
			break
		}
		offset += limit
	}

	return allReviews, nil
}

func extractLocationID(url string) int {
	// Split URL by "-" and look for the part starting with "d"
	parts := strings.Split(url, "-")
	for _, part := range parts {
		// Look for the segment starting with "d" followed by numbers
		if strings.HasPrefix(part, "d") {
			// Remove the "d" prefix and any non-numeric characters
			numStr := strings.TrimPrefix(part, "d")
			numStr = strings.Split(numStr, "-")[0] // Handle cases where there might be text after the number

			// Convert to integer
			if id, err := strconv.Atoi(numStr); err == nil {
				return id
			}
		}
	}
	return 0 // Return 0 if no valid ID found
}
