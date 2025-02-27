package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/middleware"
	"github.com/review-aggregator/review-api/app/models"

	"github.com/gin-gonic/gin"
)

func HandlerGetUser(c *gin.Context) {
	user, err := middleware.GetContextUser(c)
	if err != nil {
		log.Error("Err to get context user: ", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, user)
}

type ClerkUserCreatedWebhook struct {
	Data struct {
		Birthday       string `json:"birthday"`
		CreatedAt      int64  `json:"created_at"`
		EmailAddresses []struct {
			EmailAddress string        `json:"email_address"`
			ID           string        `json:"id"`
			LinkedTo     []interface{} `json:"linked_to"`
			Object       string        `json:"object"`
			Verification struct {
				Status   string `json:"status"`
				Strategy string `json:"strategy"`
			} `json:"verification"`
		} `json:"email_addresses"`
		ExternalAccounts      []interface{} `json:"external_accounts"`
		ExternalID            string        `json:"external_id"`
		FirstName             string        `json:"first_name"`
		Gender                string        `json:"gender"`
		ID                    string        `json:"id"`
		ImageURL              string        `json:"image_url"`
		LastName              string        `json:"last_name"`
		LastSignInAt          int64         `json:"last_sign_in_at"`
		Object                string        `json:"object"`
		PasswordEnabled       bool          `json:"password_enabled"`
		PhoneNumbers          []interface{} `json:"phone_numbers"`
		PrimaryEmailAddressID string        `json:"primary_email_address_id"`
		PrimaryPhoneNumberID  interface{}   `json:"primary_phone_number_id"`
		PrimaryWeb3WalletID   interface{}   `json:"primary_web3_wallet_id"`
		PrivateMetadata       struct {
		} `json:"private_metadata"`
		ProfileImageURL string `json:"profile_image_url"`
		PublicMetadata  struct {
		} `json:"public_metadata"`
		TwoFactorEnabled bool `json:"two_factor_enabled"`
		UnsafeMetadata   struct {
		} `json:"unsafe_metadata"`
		UpdatedAt   int64         `json:"updated_at"`
		Username    interface{}   `json:"username"`
		Web3Wallets []interface{} `json:"web3_wallets"`
	} `json:"data"`
	EventAttributes struct {
		HTTPRequest struct {
			ClientIP  string `json:"client_ip"`
			UserAgent string `json:"user_agent"`
		} `json:"http_request"`
	} `json:"event_attributes"`
	Object    string `json:"object"`
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
}

func HandlerSignUpClerkWebhook(c *gin.Context) {
	var body ClerkUserCreatedWebhook

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userUUID := uuid.New()
	var email string
	if body.Data.EmailAddresses != nil && len(body.Data.EmailAddresses) > 0 && body.Data.EmailAddresses[0].EmailAddress != "" {
		email = body.Data.EmailAddresses[0].EmailAddress
	}

	user := models.User{
		ID:      userUUID,
		ClerkID: body.Data.ID,
		Name:    body.Data.FirstName + " " + body.Data.LastName,
		Email:   email,
	}

	fmt.Println(user)
	if err := models.CreateUser(context.Background(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}
}
