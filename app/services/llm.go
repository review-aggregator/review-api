package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/consts"
	"github.com/review-aggregator/review-api/app/models"
)

const (
	model = "deepseek-r1"
)

type PlatformNameWithID struct {
	PlatformName string
	PlatformID   uuid.UUID
}

func GenerateProductStats(ctx context.Context, productID uuid.UUID, userID uuid.UUID) error {
	platforms, err := models.GetPlatformsByProductIDAndUserID(ctx, productID, userID)
	if err != nil {
		return fmt.Errorf("error getting platforms: %w", err)
	}

	// No platforms have been added for this product
	if len(platforms) == 0 {
		return fmt.Errorf("no platforms found")
	}

	// Only one platform has been added for this product so we can store the result with platform as "all" in product_stats table
	platformTypes := []PlatformNameWithID{}
	if len(platforms) == 1 {
		for _, timePeriod := range consts.TimePeriods {
			reviews, err := models.GetReviewsByProductIDAndUserIDAndTimePeriod(ctx, productID, userID, timePeriod)
			if err != nil {
				return fmt.Errorf("error getting reviews: %w", err)
			}

			productStats, err := GetProductStats(ctx, reviews)
			if err != nil {
				return fmt.Errorf("error getting product stats: %w", err)
			}

			productStats.ProductID = productID
			productStats.Platform = consts.PlatformAll
			productStats.TimePeriod = timePeriod

			PrettyPrint(productStats)

			err = models.CreateProductStats(ctx, productStats)
			if err != nil {
				return fmt.Errorf("error creating product stats: %w", err)
			}
		}
	} else {
		platformTypes = append(platformTypes, PlatformNameWithID{
			PlatformName: "all",
			PlatformID:   uuid.Nil,
		})
		for _, platform := range platforms {
			platformTypes = append(platformTypes, PlatformNameWithID{
				PlatformName: string(platform.Name),
				PlatformID:   platform.ID,
			})
		}
	}

	// for _, platformType := range platformTypes {
	// 	for _, timePeriod := range consts.TimePeriods {
	// 		reviews, err := models.GetReviewsByProductIDAndUserIDAndTimePeriod(ctx, productID, userID, timePeriod)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("error getting reviews: %w", err)
	// 		}

	// 		productStats, err := GetProductStats(ctx, reviews)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("error getting product stats: %w", err)
	// 		}

	// 	}
	// }

	return nil
}

func GetProductStats(ctx context.Context, reviews []*models.Review) (*models.ProductStats, error) {
	// Create the system and user messages
	messages := []map[string]string{
		{
			"role": "system",
			"content": `You are a review analyzer. Your task is to analyze and summarize product reviews and provide key highlights and pain points strictly in JSON format.
			Ensure that your response is **only** a valid JSON object and nothing else—no explanations, no introductions, no formatting hints, and no <think> tags.
			Here is the required JSON structure:
	
			{
				"key_highlights": ["highlight1", "highlight2", ...],
				"pain_points": ["issue1", "issue2", ...],
				"overall_sentiment": "brief summary of customer satisfaction"
			}
	
			DO NOT ADD OR USE ANY CURLY BRACKETS i.e. { or } IN THE <think> TAGS.
			Do not include any additional text before or after the JSON object.
			Do not add these fields within another object, field or array.
			Strictly follow the JSON structure and do not add any additional fields or properties.
			Ensure the fields used are "key_highlights", "pain_points" and "overall_sentiment" and if you are unable to find any, return an empty array.`,
		},
		{
			"role":    "user",
			"content": formatReviewsForPrompt(reviews),
		},
	}

	// Prepare the request body
	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	// Convert request body to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Send request to Ollama
	resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	response, err := readStreamingResponse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading streaming response: %w", err)
	}

	productStats := &models.ProductStats{}
	err = json.Unmarshal([]byte(response), productStats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return productStats, nil
}

// readStreamingResponse reads a streaming response from Ollama and returns the complete response
func readStreamingResponse(body io.ReadCloser) (string, error) {
	var fullResponse string
	decoder := json.NewDecoder(body)
	for {
		var result map[string]interface{}
		if err := decoder.Decode(&result); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error decoding response: %w", err)
		}

		if message, ok := result["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				fullResponse += content
			}
		}

		if done, ok := result["done"].(bool); ok && done {
			break
		}
	}

	if fullResponse == "" {
		return "", fmt.Errorf("no valid response content from Ollama")
	}

	// Log the raw response for debugging
	fmt.Println("Raw response from model:", fullResponse)

	jsonResponse, err := extractJSONFromResponse(fullResponse)
	if err != nil {
		return "", fmt.Errorf("error extracting JSON: %w", err)
	}

	if err := validateJSON(jsonResponse); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	return jsonResponse, nil
}

func removeThinkTag(input string) string {
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	return re.ReplaceAllString(input, "")
}

func extractJSONFromResponse(response string) (string, error) {
	// Remove unwanted tags or text
	cleanedResponse := removeThinkTag(response)

	// Look for the JSON object in the cleaned response
	start := strings.Index(cleanedResponse, "{")
	end := strings.LastIndex(cleanedResponse, "}")
	if start == -1 || end == -1 {
		// Return an empty JSON object as a fallback
		return `{"key_highlights": [], "pain_points": [], "overall_sentiment": ""}`, nil
	}
	return cleanedResponse[start : end+1], nil
}

func validateJSON(jsonString string) error {
	var jsonData map[string]interface{}
	return json.Unmarshal([]byte(jsonString), &jsonData)
}

type ReviewData struct {
	RatingValue float64
	ReviewBody  string
}

// formatReviewsForPrompt converts the reviews into a string format suitable for the prompt
func formatReviewsForPrompt(reviews []*models.Review) string {
	var reviewTexts []ReviewData
	for _, review := range reviews {
		reviewTexts = append(reviewTexts, ReviewData{
			RatingValue: review.RatingValue,
			ReviewBody:  review.ReviewBody,
		})
	}
	return fmt.Sprintf("%v", reviewTexts) + "Please analyze these reviews and provide key highlights, pain points and overall sentiment in JSON format as mentioned above\n\n"
}

// PrettyPrint prints any struct in a readable JSON format.
func PrettyPrint(v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(string(b))
}
