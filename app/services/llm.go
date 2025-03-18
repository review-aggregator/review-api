package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/review-aggregator/review-api/app/consts"
	"github.com/review-aggregator/review-api/app/models"
)

const (
	model               = "deepseek-r1"
	groqModel           = "llama-3.1-8b-instant"
	ReviewTypeSummary   = "summary"
	ReviewTypeSentiment = "sentiment"
	openAPIURL          = "http://localhost:11434/api/chat"
	groqAPIURL          = "https://api.groq.com/openai/v1/chat/completions"
)

type LLMProvider string

const (
	ProviderOllama LLMProvider = "ollama"
	ProviderGroq   LLMProvider = "groq"
)

type PlatformNameWithID struct {
	PlatformName consts.PlatformType
	PlatformID   uuid.UUID
}

func GenerateProductStats(ctx context.Context, productID uuid.UUID, userID uuid.UUID) error {
	fmt.Println("started generating product stats")
	product, err := models.GetProductByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("error getting product: %w", err)
	}

	platforms, err := models.GetPlatformsByProductIDAndUserID(ctx, productID, userID)
	if err != nil {
		return fmt.Errorf("error getting platforms: %w", err)
	}

	PrettyPrint(platforms)

	// No platforms have been added for this product
	if len(platforms) == 0 {
		return fmt.Errorf("no platforms found")
	}

	// Only one platform has been added for this product so we can store the result with platform as "all" in product_stats table
	platformTypes := []PlatformNameWithID{}
	if len(platforms) == 1 {
		fmt.Println("Single platform found for product ID:", productID)
		for _, timePeriod := range consts.TimePeriods {
			fmt.Println("Started for time period: ", timePeriod)
			err := processTimePeriodsStats(ctx, productID, userID, product.Description, timePeriod)
			if err != nil {
				return fmt.Errorf("error processing time period %s: %w", timePeriod, err)
			}
		}
		return nil
	} else {
		fmt.Println("Multiple platforms found for product ID:", productID)
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
				fmt.Println("started for platform:", platform.PlatformName, "and time period:", timePeriod)
				time.Sleep(60 * time.Second)
				reviews, err := models.GetReviewsByPlatformIDAndUserIDAndTimePeriod(ctx, platform.PlatformID, userID, timePeriod)
				if err != nil {
					return fmt.Errorf("error getting reviews: %w", err)
				}

				productStats, err := GetProductStats(ctx, reviews, product.Description)
				if err != nil {
					return fmt.Errorf("error getting product stats: %w", err)
				}

				productSentiment, err := GetSentimentAnalysis(ctx, reviews, product.Description)
				if err != nil {
					return fmt.Errorf("error getting product stats: %w", err)
				}

				var productSentimentPQArray pq.StringArray
				err = json.Unmarshal([]byte(productSentiment), &productSentimentPQArray)
				if err != nil {
					return fmt.Errorf("error unmarshalling sentiment count: %w", err)
				}

				productStats.ProductID = productID
				productStats.Platform = platform.PlatformName
				productStats.TimePeriod = timePeriod
				productStats.SentimentCount = productSentimentPQArray

				err = models.CreateProductStats(ctx, productStats)
				if err != nil {
					return fmt.Errorf("error creating product stats: %w", err)
				}
			}
		}
	}

	return nil
}

func processTimePeriodsStats(ctx context.Context, productID uuid.UUID, userID uuid.UUID, productDescription string, timePeriod consts.TimePeriodType) error {
	fmt.Println("processing time periods stats for product ID:", productID, "and time period:", timePeriod)
	reviews, err := models.GetReviewsByProductIDAndUserIDAndTimePeriod(ctx, productID, userID, timePeriod)
	if err != nil {
		return fmt.Errorf("error getting reviews: %w", err)
	}

	productSentiment, err := GetSentimentAnalysis(ctx, reviews, productDescription)
	if err != nil {
		return fmt.Errorf("error getting sentiment analysis: %w", err)
	}

	fmt.Println("product sentiment:")
	PrettyPrint(productSentiment)

	var productSentimentPQArray pq.StringArray
	err = json.Unmarshal([]byte(productSentiment), &productSentimentPQArray)
	if err != nil {
		return fmt.Errorf("error unmarshalling sentiment count: %w", err)
	}

	fmt.Println("sleeping for 30 seconds")
	time.Sleep(60 * time.Second)
	fmt.Println("starting product stats")
	productStats, err := GetProductStats(ctx, reviews, productDescription)
	if err != nil {
		return fmt.Errorf("error getting product stats: %w", err)
	}

	fmt.Println("product stats:")
	PrettyPrint(productStats)

	productStats.ProductID = productID
	productStats.Platform = consts.PlatformAll
	productStats.TimePeriod = timePeriod
	productStats.SentimentCount = productSentimentPQArray

	err = models.CreateProductStats(ctx, productStats)
	if err != nil {
		return fmt.Errorf("error creating product stats: %w", err)
	}

	fmt.Println("product stats created for product ID:", productID, "and time period:", timePeriod)
	return nil
}

func callLLMAPI(ctx context.Context, messages []map[string]string, provider LLMProvider, apiKey string) (string, error) {
	requestBody := map[string]interface{}{
		"messages": messages,
		"stream":   false,
	}

	switch provider {
	case ProviderGroq:
		requestBody["model"] = groqModel
		requestBody["temperature"] = 1
		requestBody["max_completion_tokens"] = 1024
		requestBody["top_p"] = 1
		requestBody["stop"] = nil
	case ProviderOllama:
		requestBody["model"] = model
	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", getAPIURL(provider), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if provider == ProviderGroq {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	response, err := readStreamingResponse(body)
	if err != nil {
		return "", fmt.Errorf("error reading streaming response: %w", err)
	}

	return response, nil
}

func getAPIURL(provider LLMProvider) string {
	switch provider {
	case ProviderGroq:
		return groqAPIURL
	case ProviderOllama:
		return openAPIURL
	default:
		return ""
	}
}

func GetProductStats(ctx context.Context, reviews []*models.Review, productDescription string) (*models.ProductStats, error) {
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

			"key_highlights" and "pain_points" should give an array output. Limit the number of key highlights and pain points to 5.
			"overall_sentiment" should be a string which is a brief summary of all reviews"
	
			DO NOT ADD OR USE ANY CURLY BRACKETS i.e. { or } IN THE <think> TAGS.
			Do not include any additional text before or after the JSON object.
			Do not add these fields within another object, field or array.
			Strictly follow the JSON structure and do not add any additional fields or properties.
			Ensure the fields used are "key_highlights", "pain_points" and "overall_sentiment" and if you are unable to find any, return an empty array.`,
		},
		{
			"role":    "user",
			"content": FormatReviewsForPrompt(reviews, ReviewTypeSummary, productDescription),
		},
	}

	// Use Ollama by default, can be changed to ProviderGroq
	body, err := callLLMAPI(ctx, messages, ProviderGroq, os.Getenv("GROQ_API_KEY_SUMMARY"))
	if err != nil {
		return nil, fmt.Errorf("error calling LLM API: %w", err)
	}

	productStats := &models.ProductStats{}
	err = json.Unmarshal([]byte(body), productStats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return productStats, nil
}

type SentimentCategory struct {
	Category  string `json:"category"`
	Positive  int    `json:"positive"`
	Negative  int    `json:"negative"`
	NoOpinion int    `json:"no_opinion"`
}

func GetSentimentAnalysis(ctx context.Context, reviews []*models.Review, productDescription string) (string, error) {
	categories := []string{"Product Quality", "User Experience", "Price Value", "Customer Service"}
	messages := []map[string]string{
		{
			"role": "system",
			"content": fmt.Sprintf(`You are a sentiment analyzer. Your task is to analyze and count which reviews are positive, negative or have no opinion about "%s".
			Ensure that your response is **only** a valid JSON object and nothing else—no explanations, no introductions, no formatting hints, and no <think> tags.
			Here is the required JSON structure:
	
			[{
				"category": "%s",
				"positive": 0,
				"negative": 0,
				"no_opinion": 0
			},
			{
				"category": "%s",
				"positive": 0,
				"negative": 0,
				"no_opinion": 0
			},
			{
				"category": "%s",
				"positive": 0,
				"negative": 0,
				"no_opinion": 0
			},
			{
				"category": "%s",
				"positive": 0,
				"negative": 0,
				"no_opinion": 0
			}]
	
			DO NOT ADD OR USE ANY CURLY BRACKETS i.e. { or } IN THE <think> TAGS.
			Do not include any additional text before or after the JSON object.
			Do not add these fields within another object, field or array.
			Strictly follow the JSON structure and do not add any additional fields or properties.
			If a review doesn't mention anything about %s, count it as "no_opinion".`, categories[0], categories[0], categories[1], categories[2], categories[3], categories[0]),
		},
		{
			"role":    "user",
			"content": FormatReviewsForPrompt(reviews, ReviewTypeSentiment, productDescription),
		},
	}

	// Use Ollama by default, can be changed to ProviderGroq
	body, err := callLLMAPI(ctx, messages, ProviderGroq, os.Getenv("GROQ_API_KEY_SENTIMENT"))
	if err != nil {
		return "", fmt.Errorf("error calling LLM API: %w", err)
	}

	// Parse the response into our struct
	var sentimentCategories []SentimentCategory
	if err := json.Unmarshal([]byte(body), &sentimentCategories); err != nil {
		return "", fmt.Errorf("error unmarshalling sentiment categories: %w", err)
	}

	// Convert to string array format for PostgreSQL
	var sentimentStrings []string
	for _, category := range sentimentCategories {
		categoryJSON, err := json.Marshal(category)
		if err != nil {
			return "", fmt.Errorf("error marshalling category: %w", err)
		}
		sentimentStrings = append(sentimentStrings, string(categoryJSON))
	}

	// Convert back to JSON string array
	resultJSON, err := json.Marshal(sentimentStrings)
	if err != nil {
		return "", fmt.Errorf("error marshalling final result: %w", err)
	}

	return string(resultJSON), nil
}

// readStreamingResponse reads a non-streaming response and returns the complete response
func readStreamingResponse(body []byte) (string, error) {
	var result map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&result); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	var fullResponse string
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					fullResponse = content
				}
			}
		}
	}

	if fullResponse == "" {
		return "", fmt.Errorf("no valid response content from API")
	}

	// Log the raw response for debugging
	fmt.Println("Raw response from model:", fullResponse)

	return fullResponse, nil
}

type ReviewData struct {
	RatingValue float64
	ReviewBody  string
}

// formatReviewsForPrompt converts the reviews into a string format suitable for the prompt
func FormatReviewsForPrompt(reviews []*models.Review, reviewType string, productDescription string) string {
	var reviewTexts []ReviewData
	for _, review := range reviews {
		reviewTexts = append(reviewTexts, ReviewData{
			RatingValue: review.RatingValue,
			ReviewBody:  review.ReviewBody,
		})
	}

	prompt := ""
	if reviewType == ReviewTypeSummary {
		prompt = fmt.Sprintf("You are a review analyzer. Your task is to analyze and summarize product reviews and provide key highlights and pain points strictly in JSON format.\n\nProduct Description: %s\n\n", productDescription)
	} else if reviewType == ReviewTypeSentiment {
		prompt = fmt.Sprintf("You are a sentiment analyzer. Your task is to analyze and count which reviews are positive, negative or neutral on the following topics 'Product Quality', 'User Experience', 'Price Value' and 'Customer Service'.\n\nProduct Description: %s\n\n", productDescription)
	}

	return prompt + fmt.Sprintf("%v", reviewTexts)
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
