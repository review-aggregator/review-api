package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/middleware"
	"github.com/review-aggregator/review-api/app/models"
	"github.com/review-aggregator/review-api/app/utils"
	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
)

type SignUpBody struct {
	Name     string `db:"name" json:"name" validate:"min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func HandlerSignUp(c *gin.Context) {
	var body SignUpBody

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userUUID := uuid.New()

	// generate hash of the from byte slice
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.MinCost)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	user := models.User{
		ID:       userUUID,
		Name:     body.Name,
		Email:    body.Email,
		Password: string(hash),
	}

	fmt.Println(user)
	if err := models.CreateUser(context.Background(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"msg": "User has been created",
	})
}

type LoginBody struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func HandlerLogin(c *gin.Context) {
	var body LoginBody

	err := c.ShouldBindJSON(&body)
	if err != nil {
		log.Info("Error while reading request body for login: ", err)
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	err = validator.New().Struct(body)
	if err != nil {
		log.Info("validator error: ", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "Invalid body",
		})
		return
	}

	user, err := models.GetUserByEmail(c.Copy().Request.Context(), body.Email)
	if err != nil {
		log.Error("Error when fetching user by email: ", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	// check if user found with email
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "Invalid email or password. Please try again.",
		})
		return

	}

	fmt.Println("stored: ", user.Password, "body: ", body.Password)
	// check if password matches
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		log.Info("Password does not match")
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "Invalid email or password. Please try again.",
		})
		return
	}

	claims := utils.AuthTokenClaims{
		UserID: user.ID,
		Email:  user.Email,
	}

	token, err := utils.GenerateAuthToken("secret", claims)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	respData := struct {
		models.User
		AccessToken string `json:"access_token"`
	}{
		User:        *user,
		AccessToken: token,
	}

	c.JSON(http.StatusOK, gin.H{
		"msg":  "User logged in successfully",
		"data": respData,
	})
}

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
