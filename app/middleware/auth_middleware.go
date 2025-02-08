package middleware

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/review-aggregator/review-api/app/models"
	"github.com/review-aggregator/review-api/app/utils"

	"github.com/gin-gonic/gin"
)

var (
	log = utils.CreateLogger()
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parsedToken, err := utils.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !parsedToken.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		userIDstring, ok := claims["user_id"].(string)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		userID, err := uuid.Parse(userIDstring)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		user, err := models.GetUserByID(c, userID)
		if err == sql.ErrNoRows {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		contextUser := models.User{
			ID:    user.ID,
			Email: user.Email,
		}

		c.Set("user", contextUser)
		c.Next()
	}
}

func GetContextUser(c *gin.Context) (models.User, error) {
	var contextUser models.User
	contextUserMap, exists := c.Get("user")

	if !exists {
		error := errors.New("user does not exists in gin.Context")
		return contextUser, error
	}

	contextUser, ok := contextUserMap.(models.User)
	if !ok {
		error := errors.New("not ok while changing type of contextUserMap to contextUser")
		log.Error("Error while changing type of contextUserMap to contextUser: ", error)
		return contextUser, error
	}

	return contextUser, nil

}
