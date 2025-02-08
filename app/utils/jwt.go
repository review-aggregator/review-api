package utils

import (
	"fmt"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

type AuthTokenClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

// GenerateAuthToken generates auth token for a given secret
func GenerateAuthToken(secret string, claimsToAdd AuthTokenClaims) (string, error) {
	var secretKeyInBytes = []byte(secret)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = 0
	claims["user_id"] = claimsToAdd.UserID
	claims["email"] = claimsToAdd.Email

	tokenString, err := token.SignedString(secretKeyInBytes)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT validates the given JWT token and returns the parsed token
func ValidateJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure that the token's signing method is what you expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return token, nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}
