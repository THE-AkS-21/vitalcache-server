package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/THE-AkS-21/vitalcache-server/cmd/api/utils"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}
		tokenString := parts[1]
		userID, claims, err := utils.ValidateToken(tokenString, jwtSecret)
		if err != nil {
			slog.Warn("JWT validation failed", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		// Set universal user info in the context
		c.Set("user_id", uint(userID))
		c.Set("user_role", claims["role"])
		c.Next()
	}
}
