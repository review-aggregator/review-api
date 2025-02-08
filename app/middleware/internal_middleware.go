package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func InternalMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
