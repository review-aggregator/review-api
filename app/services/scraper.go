package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/review-aggregator/review-api/app/models"
)

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
	req, err := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:8001/scrape/trustpilot", bytes.NewBuffer(jsonBody))
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
