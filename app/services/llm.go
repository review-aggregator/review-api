package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	groqModel           = "llama-3.3-70b-versatile"
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

	errChan := make(chan error, 10)

	// Only one platform has been added for this product so we can store the result with platform as "all" in product_stats table
	platformTypes := []PlatformNameWithID{}
	if len(platforms) == 1 {
		var wg sync.WaitGroup
		errCount := 0
		for _, timePeriod := range consts.TimePeriods {
			fmt.Println("Started for time period: ", timePeriod)
			wg.Add(1)
			go func(tp consts.TimePeriodType) {
				defer wg.Done()
				processTimePeriodsStats(ctx, productID, userID, product.Description, tp, errChan)
			}(timePeriod)
		}

		// Start a goroutine to close errChan after all workers are done
		go func() {
			wg.Wait()
			close(errChan)
		}()

		// Collect any errors from the channel
		for err := range errChan {
			errCount++
			fmt.Printf("Error processing time period: %v\n", err)
		}

		if errCount > 0 {
			return fmt.Errorf("encountered %d errors while processing time periods", errCount)
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

					productStats, err := GetProductStats(ctx, reviews, product.Description)
					if err != nil {
						errChan <- fmt.Errorf("error getting product stats: %w", err)
						return
					}

					productSentiment, err := GetSentimentAnalysis(ctx, reviews, "Product Quality", product.Description)
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

func processTimePeriodsStats(ctx context.Context, productID uuid.UUID, userID uuid.UUID, productDescription string, timePeriod consts.TimePeriodType, errChan chan error) {
	fmt.Println("processing time periods stats for product ID:", productID, "and time period:", timePeriod)
	reviews, err := models.GetReviewsByProductIDAndUserIDAndTimePeriod(ctx, productID, userID, timePeriod)
	if err != nil {
		errChan <- fmt.Errorf("error getting reviews: %w", err)
		return
	}

	productStats, err := GetProductStats(ctx, reviews, productDescription)
	if err != nil {
		errChan <- fmt.Errorf("error getting product stats: %w", err)
		return
	}

	fmt.Println("product stats:")
	PrettyPrint(productStats)

	categories := []string{"Product Quality", "User Experience", "Price Value", "Customer Service"}
	sentimentChan := make(chan struct {
		category string
		result   string
		err      error
	}, len(categories))

	var wg sync.WaitGroup
	for _, category := range categories {
		wg.Add(1)
		go func(cat string) {
			defer wg.Done()
			productSentiment, err := GetSentimentAnalysis(ctx, reviews, cat, productDescription)
			sentimentChan <- struct {
				category string
				result   string
				err      error
			}{
				category: cat,
				result:   productSentiment,
				err:      err,
			}
		}(category)
	}

	// Start a goroutine to close the channel once all workers are done
	go func() {
		wg.Wait()
		fmt.Println("closing sentiment channel")
		close(sentimentChan)
	}()

	// Collect results
	var allSentiments []string
	sentimentMap := make(map[string]string) // To maintain order

	for result := range sentimentChan {
		if result.err != nil {
			errChan <- fmt.Errorf("error getting sentiment analysis for %s: %w", result.category, result.err)
			return
		}
		sentimentMap[result.category] = result.result
	}

	PrettyPrint(sentimentMap)

	// Maintain the order of categories in the final result
	// for _, category := range categories {
	// 	if sentiment, ok := sentimentMap[category]; ok {
	// 		allSentiments = append(allSentiments, sentiment)
	// 	}
	// }

	var productSentimentPQArray pq.StringArray
	productSentimentPQArray = allSentiments

	productStats.ProductID = productID
	productStats.Platform = consts.PlatformAll
	productStats.TimePeriod = timePeriod
	productStats.SentimentCount = productSentimentPQArray

	err = models.CreateProductStats(ctx, productStats)
	if err != nil {
		errChan <- fmt.Errorf("error creating product stats: %w", err)
		return
	}

	fmt.Println("product stats created for product ID:", productID, "and time period:", timePeriod)
}

func callLLMAPI(ctx context.Context, messages []map[string]string, provider LLMProvider) (io.ReadCloser, error) {
	requestBody := map[string]interface{}{
		"messages": messages,
		"stream":   true,
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
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", getAPIURL(provider), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if provider == ProviderGroq {
		groqAPIKey := os.Getenv("GROQ_API_KEY")
		if groqAPIKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY environment variable not set")
		}
		req.Header.Set("Authorization", "Bearer "+groqAPIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
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
			"content": FormatReviewsForPrompt(reviews, ReviewTypeSummary, productDescription),
		},
	}

	// Use Ollama by default, can be changed to ProviderGroq
	body, err := callLLMAPI(ctx, messages, ProviderGroq)
	if err != nil {
		return nil, fmt.Errorf("error calling LLM API: %w", err)
	}
	defer body.Close()

	response, err := readStreamingResponse(body, ReviewTypeSummary)
	if err != nil {
		return nil, fmt.Errorf("error reading streaming response: %w", err)
	}

	PrettyPrint(response)

	productStats := &models.ProductStats{}
	err = json.Unmarshal([]byte(response), productStats)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return productStats, nil
}

func GetSentimentAnalysis(ctx context.Context, reviews []*models.Review, category string, productDescription string) (string, error) {
	fmt.Println("running sentiment analysis for category:", category)
	messages := []map[string]string{
		{
			"role": "system",
			"content": fmt.Sprintf(`You are a sentiment analyzer. Your task is to analyze and count which reviews are positive, negative or have no opinion about "%s".
			Ensure that your response is **only** a valid JSON object and nothing else—no explanations, no introductions, no formatting hints, and no <think> tags.
			Here is the required JSON structure:
	
			{
				"category": "%s",
				"positive": 0,
				"negative": 0,
				"no_opinion": 0
			}
	
			DO NOT ADD OR USE ANY CURLY BRACKETS i.e. { or } IN THE <think> TAGS.
			Do not include any additional text before or after the JSON object.
			Do not add these fields within another object, field or array.
			Strictly follow the JSON structure and do not add any additional fields or properties.
			If a review doesn't mention anything about %s, count it as "no_opinion".`, category, category, category),
		},
		{
			"role":    "user",
			"content": FormatReviewsForPrompt(reviews, ReviewTypeSentiment, productDescription),
		},
	}

	// Use Ollama by default, can be changed to ProviderGroq
	body, err := callLLMAPI(ctx, messages, ProviderGroq)
	if err != nil {
		return "", fmt.Errorf("error calling LLM API: %w", err)
	}
	defer body.Close()

	response, err := readStreamingResponse(body, ReviewTypeSentiment)
	if err != nil {
		return "", fmt.Errorf("error reading streaming response: %w", err)
	}

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
	fmt.Println("Raw response from model:", fullResponse)

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
		if reviewType == ReviewTypeSummary {
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
