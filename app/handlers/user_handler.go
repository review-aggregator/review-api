package handlers

import (
	"context"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
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

	user := models.User{
		ID:       uuid.New(),
		Name:     body.Name,
		Email:    body.Email,
		Password: body.Password,
	}

	if err := models.CreateUser(context.Background(), &user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}
	c.JSON(http.StatusCreated, user)
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
