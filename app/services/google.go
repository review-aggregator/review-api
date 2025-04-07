package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Config holds configuration for Google API access
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	APIKey       string
	Scopes       []string
}

// Service provides methods for interacting with Google Reviews
type Service struct {
	config     Config
	db         *sql.DB
	oauthConf  *oauth2.Config
	httpClient *http.Client
}

// NewService creates a new Google Reviews service
func NewService(db *sql.DB, config Config) *Service {
	oauthConf := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint:     google.Endpoint,
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Service{
		config:     config,
		db:         db,
		oauthConf:  oauthConf,
		httpClient: httpClient,
	}
}

// AuthURL returns the URL for OAuth 2.0 authorization
func (s *Service) AuthURL(state string) string {
	return s.oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// Exchange exchanges authorization code for token
func (s *Service) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return s.oauthConf.Exchange(ctx, code)
}

// SaveTokenForBusiness saves OAuth token for a business
func (s *Service) SaveTokenForBusiness(businessID string, token *oauth2.Token) error {
	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return err
	}

	_, err = s.db.Exec("UPDATE businesses SET oauth_token = $1 WHERE id = $2", tokenJSON, businessID)
	return err
}

// GetTokenForBusiness retrieves OAuth token for a business
func (s *Service) GetTokenForBusiness(businessID string) (*oauth2.Token, error) {
	var tokenJSON []byte
	err := s.db.QueryRow("SELECT oauth_token FROM businesses WHERE id = $1", businessID).Scan(&tokenJSON)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	err = json.Unmarshal(tokenJSON, &token)
	return &token, err
}

// Account represents a Google Business Profile account
type Account struct {
	Name          string `json:"name"`
	AccountName   string `json:"accountName"`
	Type          string `json:"type"`
	AccountNumber string `json:"accountNumber"`
}

// Location represents a Google Business Profile location
type Location struct {
	Name         string `json:"name"`
	LocationName string `json:"locationName"`
	PlaceID      string `json:"placeId"`
	StoreCode    string `json:"storeCode"`
}

// GoogleReview represents a review from Google Business Profile API
type GoogleReview struct {
	Name     string `json:"name"`
	ReviewID string `json:"reviewId"`
	Reviewer struct {
		DisplayName     string `json:"displayName"`
		ProfilePhotoURL string `json:"profilePhotoUrl"`
	} `json:"reviewer"`
	StarRating  int    `json:"starRating"`
	Comment     string `json:"comment"`
	CreateTime  string `json:"createTime"`
	UpdateTime  string `json:"updateTime"`
	ReviewReply *struct {
		Comment    string `json:"comment"`
		UpdateTime string `json:"updateTime"`
	} `json:"reviewReply"`
}

// FetchAccounts fetches Google Business Profile accounts for the authenticated user
func (s *Service) FetchAccounts(ctx context.Context, token *oauth2.Token) ([]Account, error) {
	client := s.oauthConf.Client(ctx, token)

	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://mybusinessaccountmanagement.googleapis.com/v1/accounts", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var result struct {
		Accounts []Account `json:"accounts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Accounts, nil
}

// FetchLocations fetches locations for a given account
func (s *Service) FetchLocations(ctx context.Context, token *oauth2.Token, accountID string) ([]Location, error) {
	client := s.oauthConf.Client(ctx, token)

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://mybusinessbusinessinformation.googleapis.com/v1/%s/locations", accountID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var result struct {
		Locations []Location `json:"locations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Locations, nil
}

// FetchReviews fetches reviews for a given location
func (s *Service) FetchReviews(ctx context.Context, token *oauth2.Token, locationID string) ([]GoogleReview, error) {
	client := s.oauthConf.Client(ctx, token)

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://mybusinessreviews.googleapis.com/v1/%s/reviews", locationID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var result struct {
		Reviews []GoogleReview `json:"reviews"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Reviews, nil
}

// FetchReviewsWithPlaceID fetches reviews using the Google Places API (alternative method)
func (s *Service) FetchReviewsWithPlaceID(ctx context.Context, placeID string) ([]GoogleReview, error) {
	reqURL := fmt.Sprintf("https://maps.googleapis.com/maps/api/place/details/json?place_id=%s&fields=reviews&key=%s",
		url.QueryEscape(placeID), s.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	var placesResp struct {
		Result struct {
			Reviews []struct {
				AuthorName              string `json:"author_name"`
				ProfilePhotoURL         string `json:"profile_photo_url"`
				Rating                  int    `json:"rating"`
				Text                    string `json:"text"`
				Time                    int64  `json:"time"`
				RelativeTimeDescription string `json:"relative_time_description"`
			} `json:"reviews"`
		} `json:"result"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&placesResp); err != nil {
		return nil, err
	}

	if placesResp.Status != "OK" {
		return nil, fmt.Errorf("places API error: %s", placesResp.Status)
	}

	// Convert Places API reviews to our GoogleReview format
	reviews := make([]GoogleReview, 0, len(placesResp.Result.Reviews))
	for i, review := range placesResp.Result.Reviews {
		createTime := time.Unix(review.Time, 0).Format(time.RFC3339)
		reviewID := fmt.Sprintf("places-%s-%d", placeID, i) // Generate a synthetic ID

		reviews = append(reviews, GoogleReview{
			Name:     reviewID,
			ReviewID: reviewID,
			Reviewer: struct {
				DisplayName     string `json:"displayName"`
				ProfilePhotoURL string `json:"profilePhotoUrl"`
			}{
				DisplayName:     review.AuthorName,
				ProfilePhotoURL: review.ProfilePhotoURL,
			},
			StarRating: review.Rating,
			Comment:    review.Text,
			CreateTime: createTime,
			UpdateTime: createTime,
		})
	}

	return reviews, nil
}

// StoreReviews stores reviews in the database
func (s *Service) StoreReviews(businessID string, googleReviews []GoogleReview) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO reviews (id, business_id, reviewer_name, reviewer_photo, rating, comment, create_time, update_time, reply_comment, reply_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			rating = $5,
			comment = $6,
			update_time = $8,
			reply_comment = $9,
			reply_time = $10
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, review := range googleReviews {
		createTime, _ := time.Parse(time.RFC3339, review.CreateTime)
		updateTime, _ := time.Parse(time.RFC3339, review.UpdateTime)

		var replyComment string
		var replyTime time.Time
		if review.ReviewReply != nil && review.ReviewReply.Comment != "" {
			replyComment = review.ReviewReply.Comment
			replyTime, _ = time.Parse(time.RFC3339, review.ReviewReply.UpdateTime)
		}

		_, err = stmt.Exec(
			review.Name,
			businessID,
			review.Reviewer.DisplayName,
			review.Reviewer.ProfilePhotoURL,
			review.StarRating,
			review.Comment,
			createTime,
			updateTime,
			replyComment,
			replyTime,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SchedulePeriodicReviewFetch schedules periodic fetching of reviews for all businesses
func (s *Service) SchedulePeriodicReviewFetch(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				err := s.FetchAllBusinessesReviews(ctx)
				if err != nil {
					log.Printf("Error fetching reviews: %v", err)
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// FetchAllBusinessesReviews fetches reviews for all businesses in our system
func (s *Service) FetchAllBusinessesReviews(ctx context.Context) error {
	rows, err := s.db.Query("SELECT id, location_id, place_id FROM businesses WHERE oauth_token IS NOT NULL OR place_id IS NOT NULL")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var businessID, locationID string
		var placeID sql.NullString
		if err := rows.Scan(&businessID, &locationID, &placeID); err != nil {
			log.Printf("Error scanning business row: %v", err)
			continue
		}

		var reviews []GoogleReview
		var err error

		// Try fetching with Business Profile API first if we have a location ID
		if locationID != "" {
			token, err := s.GetTokenForBusiness(businessID)
			if err == nil {
				reviews, err = s.FetchReviews(ctx, token, locationID)
				if err != nil {
					log.Printf("Error fetching reviews with Business Profile API for business %s: %v", businessID, err)
				}
			} else {
				log.Printf("Error getting token for business %s: %v", businessID, err)
			}
		}

		// Fall back to Places API if we have a place ID and couldn't get reviews via Business Profile API
		if len(reviews) == 0 && placeID.Valid && placeID.String != "" {
			reviews, err = s.FetchReviewsWithPlaceID(ctx, placeID.String)
			if err != nil {
				log.Printf("Error fetching reviews with Places API for business %s: %v", businessID, err)
				continue
			}
		}

		if len(reviews) > 0 {
			err = s.StoreReviews(businessID, reviews)
			if err != nil {
				log.Printf("Error storing reviews for business %s: %v", businessID, err)
				continue
			}
			log.Printf("Successfully updated %d reviews for business %s", len(reviews), businessID)
		}
	}

	return rows.Err()
}

// SetupOAuthHandler sets up an HTTP handler for OAuth callback
func (s *Service) SetupOAuthHandler(mux *http.ServeMux, callbackPath string, stateValidator func(string) bool, successHandler func(w http.ResponseWriter, r *http.Request, token *oauth2.Token)) {
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if !stateValidator(state) {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		token, err := s.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		successHandler(w, r, token)
	})
}

// Convert Google reviews to our Review model
func (s *Service) ConvertGoogleReviewsToReviews(platformID uuid.UUID, googleReviews []GoogleReview) ([]*models.Review, error) {
	reviews := make([]*models.Review, 0, len(googleReviews))
	for _, review := range googleReviews {
		createTime, _ := time.Parse(time.RFC3339, review.CreateTime)
		updateTime, _ := time.Parse(time.RFC3339, review.UpdateTime)

		reviews = append(reviews, &models.Review{
			ID:            uuid.New(),
			PlatformID:    platformID,
			RatingValue:   float64(review.StarRating),
			ReviewBody:    review.Comment,
			DatePublished: createTime.Format(time.RFC3339),
			UpdatedAt:     updateTime,
		})
	}

	return reviews, nil
}

// Use the CreateReviews function from models package
// func (s *Service) StoreReviewsInDatabase(ctx context.Context, businessID string, googleReviews []GoogleReview) error {
// 	reviews, err := s.ConvertGoogleReviewsToReviews(businessID, googleReviews)
// 	if err != nil {
// 		return fmt.Errorf("failed to convert reviews: %w", err)
// 	}

// 	err = models.CreateReviews(ctx, s.db, reviews)
// 	if err != nil {
// 		return fmt.Errorf("failed to create reviews: %w", err)
// 	}

// 	return nil
// }
