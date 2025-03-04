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
	"sync"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/review-aggregator/review-api/app/consts"
	"github.com/review-aggregator/review-api/app/models"
)

const (
	model               = "deepseek-r1"
	ReviewTypeSummary   = "summary"
	ReviewTypeSentiment = "sentiment"
)

type PlatformNameWithID struct {
	PlatformName consts.PlatformType
	PlatformID   uuid.UUID
}

func GenerateProductStats(ctx context.Context, productID uuid.UUID, userID uuid.UUID) error {
	fmt.Println("started generating product stats")
	platforms, err := models.GetPlatformsByProductIDAndUserID(ctx, productID, userID)
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
			fmt.Println("Started for time period: ", timePeriod)
			wg.Add(1)
			go func() {
				defer wg.Done()
				reviews, err := models.GetReviewsByProductIDAndUserIDAndTimePeriod(ctx, productID, userID, timePeriod)
				if err != nil {
					errChan <- fmt.Errorf("error getting reviews: %w", err)
					return
				}
				// PrettyPrint(reviews)

				productSentiment, err := GetSentimentAnalysis(ctx, reviews)
				if err != nil {
					errChan <- fmt.Errorf("error getting product stats: %w", err)
					return
				}

				PrettyPrint(productSentiment)

				productStats, err := GetProductStats(ctx, reviews)
				if err != nil {
					errChan <- fmt.Errorf("error getting product stats: %w", err)
					return
				}

				var productSentimentPQArray pq.StringArray
				err = json.Unmarshal([]byte(productSentiment), &productSentimentPQArray)
				if err != nil {
					errChan <- fmt.Errorf("error unmarshalling sentiment count: %w", err)
					return
				}

				productStats.ProductID = productID
				productStats.Platform = consts.PlatformAll
				productStats.TimePeriod = timePeriod
				productStats.SentimentCount = productSentimentPQArray

				// PrettyPrint(productStats)

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
					reviews, err := models.GetReviewsByPlatformIDAndUserIDAndTimePeriod(ctx, platform.PlatformID, userID, timePeriod)
					if err != nil {
						errChan <- fmt.Errorf("error getting reviews: %w", err)
						return
					}
					// PrettyPrint(reviews)

					productStats, err := GetProductStats(ctx, reviews)
					if err != nil {
						errChan <- fmt.Errorf("error getting product stats: %w", err)
						return
					}

					productSentiment, err := GetSentimentAnalysis(ctx, reviews)
					if err != nil {
						errChan <- fmt.Errorf("error getting product stats: %w", err)
						return
					}

					PrettyPrint(productSentiment)

					var productSentimentPQArray pq.StringArray
					err = json.Unmarshal([]byte(productSentiment), &productSentimentPQArray)
					if err != nil {
						errChan <- fmt.Errorf("error unmarshalling sentiment count: %w", err)
						return
					}

					productStats.ProductID = productID
					productStats.Platform = platform.PlatformName
					productStats.TimePeriod = timePeriod
					productStats.SentimentCount = productSentimentPQArray

					// PrettyPrint(productStats)

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

			"key_highlights" and "pain_points" should give an array output. 
			"overall_sentiment" should be a string which is a brief summary of all reviews"
	
			DO NOT ADD OR USE ANY CURLY BRACKETS i.e. { or } IN THE <think> TAGS.
			Do not include any additional text before or after the JSON object.
			Do not add these fields within another object, field or array.
			Strictly follow the JSON structure and do not add any additional fields or properties.
			Ensure the fields used are "key_highlights", "pain_points" and "overall_sentiment" and if you are unable to find any, return an empty array.`,
		},
		{
			"role":    "user",
			"content": FormatReviewsForPrompt(reviews, ReviewTypeSummary),
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

	response, err := readStreamingResponse(resp.Body, ReviewTypeSummary)
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

func GetSentimentAnalysis(ctx context.Context, reviews []*models.Review) (string, error) {
	fmt.Println("running sentiment analysis")
	// Create the system and user messages
	messages := []map[string]string{
		{
			"role": "system",
			"content": `You are a sentiment analyzer. Your task is to analyze and count which reviews are positive, negative or neutral on the following 
			topics "Product Quality", "User Experience", "Price Value" and "Customer Service".
			Ensure that your response is **only** a valid JSON object and nothing else—no explanations, no introductions, no formatting hints, and no <think> tags.
			Here is the required JSON structure:
	
			[
				{
					"category": "product_quality",
					"positive": 10,
					"negative": 20,
					"neutral": 5
				},
				{
					"category": "user_experience",
					"positive": 20,
					"negative": 7,
					"neutral": 9
				},
				{
					"category": "price_value",
					"positive": 33,
					"negative": 18,
					"neutral": 23
				},
				{
					"category": "customer_service",
					"positive": 44,
					"negative": 12,
					"neutral": 8
				}	
			]
	
			DO NOT ADD OR USE ANY CURLY BRACKETS i.e. { or } IN THE <think> TAGS.
			Do not include any additional text before or after the JSON object.
			Do not add these fields within another object, field or array.
			Strictly follow the JSON structure and do not add any additional fields or properties.`,
		},
		{
			"role":    "user",
			"content": FormatReviewsForPrompt(reviews, ReviewTypeSentiment),
		},
	}

	// Prepare the request body
	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	fmt.Println(messages[1])

	// PrettyPrint(requestBody)

	// Convert request body to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	// Send request to Ollama
	resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error calling Ollama: %w", err)
	}
	defer resp.Body.Close()

	response, err := readStreamingResponse(resp.Body, ReviewTypeSentiment)
	if err != nil {
		return "", fmt.Errorf("error reading streaming response: %w", err)
	}

	// var sentiments []Sentiments
	// err = json.Unmarshal([]byte(response), &sentiments)
	// if err != nil {
	// 	return []Sentiments{}, fmt.Errorf("error unmarshalling response: %w", err)
	// }

	return response, nil
}

// readStreamingResponse reads a streaming response from Ollama and returns the complete response
func readStreamingResponse(body io.ReadCloser, reviewType string) (string, error) {
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
	// fmt.Println("Raw response from model:", fullResponse)

	jsonResponse, err := extractJSONFromResponse(fullResponse, reviewType)
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

func extractJSONFromResponse(response, reviewType string) (string, error) {
	// Remove unwanted tags or text
	cleanedResponse := removeThinkTag(response)

	// Look for the JSON object in the cleaned response
	start := strings.Index(cleanedResponse, "{")
	end := strings.LastIndex(cleanedResponse, "}")
	if start == -1 || end == -1 {
		// Return an empty JSON object as a fallback
		if reviewType == "summary" {
			return `{"key_highlights": [], "pain_points": [], "overall_sentiment": ""}`, nil
		} else {
			return `{"product_quality": {"positive": 0, "negative": 0, "neutral": 0}, "user_experience": {"positive": 0, "negative": 0, "neutral": 0}, "price_value": {"positive": 0, "negative": 0, "neutral": 0}, "customer_service": {"positive": 0, "negative": 0, "neutral": 0}}`, nil
		}
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
func FormatReviewsForPrompt(reviews []*models.Review, reviewType string) string {
	var reviewTexts []ReviewData
	for _, review := range reviews {
		reviewTexts = append(reviewTexts, ReviewData{
			RatingValue: review.RatingValue,
			ReviewBody:  review.ReviewBody,
		})
	}

	prompt := ""
	if reviewType == ReviewTypeSummary {
		prompt = "You are a review analyzer. Your task is to analyze and summarize product reviews and provide key highlights and pain points strictly in JSON format.\n\n"
	} else if reviewType == ReviewTypeSentiment {
		prompt = "You are a sentiment analyzer. Your task is to analyze and count which reviews are positive, negative or neutral on the following topics 'Product Quality', 'User Experience', 'Price Value' and 'Customer Service'.\n\n"
	}

	return fmt.Sprintf("%v", reviewTexts) + prompt
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
