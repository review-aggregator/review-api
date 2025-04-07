package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/consts"
	"github.com/review-aggregator/review-api/app/models"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/mybusinessaccountmanagement/v1"
	"google.golang.org/api/mybusinessbusinessinformation/v1"
	"google.golang.org/api/option"
)

func HandlerGetGoogleOAuth2URL(c *gin.Context) {
	// Get user ID and product ID from request
	userID := c.Query("user_id")
	productID := c.Query("product_id")

	// Create state with user ID and product ID
	stateData := map[string]string{
		"user_id":    userID,
		"product_id": productID,
	}

	// Convert state to JSON and encode to base64
	stateJSON, err := json.Marshal(stateData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create state"})
		return
	}
	state := base64.StdEncoding.EncodeToString(stateJSON)

	oauthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"https://www.googleapis.com/auth/business.manage"},
		Endpoint:     google.Endpoint,
	}

	// Use the encoded state in the auth URL
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func HandlerGoogleOAuth2Callback(c *gin.Context) {
	ctx := c.Request.Context()

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code is required"})
		return
	}

	encodedState := c.Query("state")
	stateJSON, err := base64.StdEncoding.DecodeString(encodedState)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}

	var stateData map[string]string
	if err := json.Unmarshal(stateJSON, &stateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid state format"})
		return
	}

	// userID := stateData["user_id"]
	productID := stateData["product_id"]

	oauthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/business.manage"},
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Endpoint:     google.Endpoint,
	}

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Error("Error while exchanging code", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange code"})
		return
	}

	management, err := mybusinessaccountmanagement.NewService(ctx, option.WithTokenSource(oauthConfig.TokenSource(ctx, token)))
	if err != nil {
		log.Error("Error while creating management service", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create management service"})
		return
	}

	information, err := mybusinessbusinessinformation.NewService(ctx, option.WithTokenSource(oauthConfig.TokenSource(ctx, token)))
	if err != nil {
		log.Error("Error while creating business information service", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create business information service"})
		return
	}

	accounts, err := management.Accounts.List().Do()
	if err != nil {
		log.Error("Error while listing accounts", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to list accounts"})
		return
	}

	googleReviewAccount := &models.GoogleReviewAccount{
		PlatformID:  uuid.MustParse(productID),
		AccountID:   accounts.Accounts[0].Name,
		AccountName: accounts.Accounts[0].AccountName,
	}

	err = models.CreateGoogleReviewAccount(ctx, googleReviewAccount)
	if err != nil {
		log.Error("Error while creating google review account", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to store google review account"})
		return
	}

	for _, account := range accounts.Accounts {
		// By default, Google limit the number of result at 10.
		// We have bump the number to 100 the max limit. If the issue occures again, follow this link:
		locations, err := information.Accounts.Locations.List(account.Name).Do(googleapi.QueryParameter("readMask", "name", "title"), googleapi.QueryParameter("pageSize", "100"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to list locations"})
			return
		}

		log.Info("Locations fetched for account", zap.Any("Account", account.AccountName), zap.Any("AccountID", account.Name), zap.Any("Locations", locations))

		var locationIDS []string
		for _, location := range locations.Locations {
			locationIDS = append(locationIDS, strings.Split(location.Name, "/")[1])
		}

		platform := &models.Platform{
			ID:        uuid.New(),
			URL:       account.AccountName,
			Locations: locationIDS,
			Name:      consts.PlatformGoogle,
			ProductID: uuid.MustParse(productID),
		}

		err = models.CreatePlatform(ctx, platform)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to store platform"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully connected to Google"})
}
