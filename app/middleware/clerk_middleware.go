package middleware

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/review-aggregator/review-api/app/models"
)

// ClerkMiddleware validates the Clerk auth token
func ClerkMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization format"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Verify the token
		claims, err := verifyClerkToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		clerkUserID, ok := claims["sub"].(string)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		user, err := models.GetUserByClerkID(c, clerkUserID)
		if err == sql.ErrNoRows {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Attach user info to context
		c.Set("user", user)
		c.Next()
	}
}

// verifyClerkToken verifies and parses the JWT token
func verifyClerkToken(tokenString string) (jwt.MapClaims, error) {
	clerkJWTKey := os.Getenv("CLERK_JWT_PUBLIC_KEY")
	if clerkJWTKey == "" {
		return nil, fmt.Errorf("missing Clerk public key")
	}

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Load Clerk public key
		return jwt.ParseRSAPublicKeyFromPEM([]byte(clerkJWTKey))
	})

	if err != nil {
		return nil, err
	}

	// Validate claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
